package main

// Unified TicketsBot entrypoint.
//
// Boots, in a single process, every Go subsystem that the upstream deployment ran
// as separate containers:
//   - worker interactions HTTP server  (Discord interaction receipt, for http-gateway)
//   - worker gateway RPC consumer       (Redis-stream Discord events, from the sharder)
//   - worker messagequeue listeners     (ticket close / autoclose / close-request / reason)
//   - dashboard REST API + websockets   (+ chatrelay -> livechat bridge)
//   - autoclose sweep                    (ported to cloud libs; see autoclose.go)
//   - database view refresher loop       (see viewrefresher.go)
//
// Premium is force-unlocked for every guild: both the worker and dashboard premium
// lookup clients are pinned to a mock that always returns the Whitelabel tier, so all
// feature gates pass and no "Powered by" branding is added.
//
// Logarchiver and the Rust gateway services (sharder/http-gateway/cache-sync) remain
// separate containers — logarchiver because it drags the incompatible legacy
// TicketsBot/{common,database} module trees; the Rust services because they own the
// Discord gateway connection. This binary talks to logarchiver over HTTP via
// archiverclient and consumes the sharder's events over the Redis stream.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/TicketsBot-cloud/archiverclient"
	"github.com/TicketsBot-cloud/common/chatrelay"
	"github.com/TicketsBot-cloud/common/model"
	"github.com/TicketsBot-cloud/common/observability"
	"github.com/TicketsBot-cloud/common/premium"
	"github.com/TicketsBot-cloud/common/rpc"
	"github.com/TicketsBot-cloud/common/secureproxy"
	"github.com/TicketsBot-cloud/common/sentry"
	"github.com/TicketsBot-cloud/gdl/rest/request"
	"github.com/TicketsBot-cloud/worker/i18n"
	"go.uber.org/zap"

	// Worker subsystem
	"github.com/TicketsBot-cloud/worker/bot/blacklist"
	workercache "github.com/TicketsBot-cloud/worker/bot/cache"
	"github.com/TicketsBot-cloud/worker/bot/dbclient"
	"github.com/TicketsBot-cloud/worker/bot/integrations"
	"github.com/TicketsBot-cloud/worker/bot/listeners/messagequeue"
	workerredis "github.com/TicketsBot-cloud/worker/bot/redis"
	rpclisteners "github.com/TicketsBot-cloud/worker/bot/rpc/listeners"
	workerutils "github.com/TicketsBot-cloud/worker/bot/utils"
	workerconfig "github.com/TicketsBot-cloud/worker/config"
	"github.com/TicketsBot-cloud/worker/event"

	// Dashboard subsystem
	dashhttp "github.com/TicketsBot-cloud/dashboard/app/http"
	"github.com/TicketsBot-cloud/dashboard/app/http/endpoints/api/ticket/livechat"
	dashconfig "github.com/TicketsBot-cloud/dashboard/config"
	dashdb "github.com/TicketsBot-cloud/dashboard/database"
	dashlog "github.com/TicketsBot-cloud/dashboard/log"
	dashredis "github.com/TicketsBot-cloud/dashboard/redis"
	dashrpc "github.com/TicketsBot-cloud/dashboard/rpc"
	dashcache "github.com/TicketsBot-cloud/dashboard/rpc/cache"
	dashutils "github.com/TicketsBot-cloud/dashboard/utils"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	startPprof()

	// --- Shared logger (worker observability stack) ---
	logger, err := observability.Configure(nil, workerconfig.Conf.JsonLogs, workerconfig.Conf.LogLevel)
	if err != nil {
		panic(err)
	}

	if len(workerconfig.Conf.DebugMode) == 0 {
		if err := sentry.Initialise(sentry.Options{
			Dsn:              workerconfig.Conf.Sentry.Dsn,
			SampleRate:       workerconfig.Conf.Sentry.SampleRate,
			EnableTracing:    workerconfig.Conf.Sentry.UseTracing,
			TracesSampleRate: workerconfig.Conf.Sentry.TracingSampleRate,
		}); err != nil {
			logger.Error("Failed to connect to sentry", zap.Error(err))
		}
	}

	// =========================================================================
	// Worker subsystem
	// =========================================================================
	logger.Info("Connecting to Redis (worker)")
	if err := workerredis.Connect(); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	logger.Info("Connecting to DB (worker)")
	dbclient.Connect(logger.With(zap.String("service", "database")))

	logger.Info("Loading i18n files")
	i18n.Init()

	logger.Info("Connecting to cache")
	pgCache, err := workercache.Connect(logger.With(zap.String("service", "cache")))
	if err != nil {
		logger.Fatal("Failed to connect to cache", zap.Error(err))
	}
	workercache.Client = &pgCache

	logger.Info("Connecting to clickhouse")
	dbclient.ConnectAnalytics(logger.With(zap.String("service", "clickhouse")))

	// Configure HTTP proxy for the Discord REST client
	if workerconfig.Conf.Discord.ProxyUrl != "" {
		request.Client.Timeout = workerconfig.Conf.Discord.RequestTimeout
		request.RegisterPreRequestHook(workerutils.ProxyHook)
	}

	// Force-unlock premium: every guild is treated as Whitelabel tier.
	mockWorker := premium.NewMockLookupClient(premium.Whitelabel, model.EntitlementSourcePatreon)
	workerutils.PremiumClient = &mockWorker

	workerutils.ArchiverClient = archiverclient.NewArchiverClient(
		archiverclient.NewProxyRetriever(workerconfig.Conf.Archiver.Url),
		[]byte(workerconfig.Conf.Archiver.AesKey),
	)

	// Metrics gathering (StatsD + Prometheus) intentionally removed. statsd.Client is left
	// uninitialised — its IncrementKey is nil-safe, so the call sites across the worker
	// no-op — and the Prometheus server/REST hooks are no longer started or registered.

	logger.Info("Initialising integrations")
	integrations.InitIntegrations()

	go messagequeue.ListenTicketClose()
	go messagequeue.ListenAutoClose(logger.With(zap.String("service", "autoclose")))
	go messagequeue.ListenCloseRequestTimer(logger.With(zap.String("service", "close-request-timer")))
	go messagequeue.ListenCloseReasonUpdate()

	go blacklist.StartCacheRefreshLoop(logger.With(zap.String("service", "blacklist_refresh")))

	// =========================================================================
	// Dashboard subsystem
	// =========================================================================
	logger.Info("Loading dashboard config")
	dcfg, err := dashconfig.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load dashboard config", zap.Error(err))
	}
	dashconfig.Conf = dcfg
	dashlog.Logger = logger

	logger.Info("Connecting to database (dashboard)")
	dashdb.ConnectToDatabase()

	logger.Info("Connecting to cache (dashboard)")
	dashcache.Instance = dashcache.NewCache()

	dashutils.ArchiverClient = archiverclient.NewArchiverClient(
		archiverclient.NewProxyRetriever(dashconfig.Conf.Bot.ObjectStore),
		[]byte(dashconfig.Conf.Bot.AesKey),
	)
	dashutils.SecureProxyClient = secureproxy.NewSecureProxy(dashconfig.Conf.SecureProxyUrl)

	if dashconfig.Conf.Bot.ProxyUrl != "" {
		request.RegisterHook(dashutils.ProxyHook)
	}

	logger.Info("Connecting to Redis (dashboard)")
	dashredis.Client = dashredis.NewRedisClient()

	socketManager := livechat.NewSocketManager()
	go socketManager.Run()
	go listenChat(dashredis.Client, socketManager)

	// Force-unlock premium on the dashboard side too.
	mockDash := premium.NewMockLookupClient(premium.Whitelabel, model.EntitlementSourcePatreon)
	dashrpc.PremiumClient = &mockDash

	logger.Info("Starting dashboard HTTP server")
	dashSrv := dashhttp.StartServer(logger, socketManager)

	// =========================================================================
	// Background sweeps (ported from autoclosedaemon + viewrefresher containers)
	// =========================================================================
	go runAutoCloseSweep(logger.With(zap.String("service", "autoclose-sweep")))
	go runViewRefresher(logger.With(zap.String("service", "view-refresher")))

	// =========================================================================
	// Worker event ingress: interactions HTTP + gateway RPC consumer
	// =========================================================================
	logger.Info("Starting interaction HTTP server")
	go event.HttpListen(workerredis.Client, &pgCache)

	logger.Info("Starting gateway RPC consumer")
	hostname, _ := os.Hostname()
	rpcClient, err := rpc.NewClient(
		logger.With(zap.String("service", "rpc")),
		rpc.Config{
			Redis:               workerredis.Client,
			ConsumerGroup:       "worker",
			ConsumerName:        hostname,
			ConsumerConcurrency: workerconfig.Conf.Streams.GoroutineLimit,
			MaxLen:              50000,
		},
		map[string]rpc.Listener{
			"stream:gateway-events": event.NewEventListener(
				logger.With(zap.String("service", "gateway-events")),
				&pgCache,
			),
			"stream:rpc:categoryupdate": rpclisteners.NewTicketStatusUpdater(&pgCache, logger),
		})
	if err != nil {
		logger.Fatal("Failed to create RPC client", zap.Error(err))
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		rpcClient.StartConsumer()
	}()

	// --- Shutdown ---
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownCh

	logger.Info("Received shutdown signal")
	rpcClient.Shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := dashSrv.Shutdown(ctx); err != nil {
		logger.Error("Dashboard HTTP server shutdown error", zap.Error(err))
	}

	if waitTimeout(&wg, 10*time.Second) {
		logger.Info("Shutdown completed gracefully")
	} else {
		logger.Warn("Graceful shutdown timed out, exiting now")
	}

	if !sentry.Flush(2 * time.Second) {
		logger.Warn("Sentry flush timed out, some events may be lost")
	}
}

// listenChat bridges the Redis chatrelay stream to dashboard livechat websockets.
func listenChat(client *dashredis.RedisClient, sm *livechat.SocketManager) {
	ch := make(chan chatrelay.MessageData)
	go chatrelay.Listen(client.Client, ch)
	for ev := range ch {
		sm.BroadcastMessage(ev)
	}
}

func startPprof() {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/{action}", pprof.Index)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	go func() {
		if err := http.ListenAndServe(":6060", mux); err != nil {
			fmt.Printf("pprof server exited: %v\n", err)
		}
	}()
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	ch := make(chan struct{})
	go func() {
		defer close(ch)
		wg.Wait()
	}()
	select {
	case <-ch:
		return true
	case <-time.After(timeout):
		return false
	}
}
