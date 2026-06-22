module github.com/TicketsBot-cloud/ticketbot

go 1.25.0

// The TicketsBot modules are resolved from local source. The require versions mirror the
// commits worker/dashboard pinned (so the module graph parses), and the replace
// directives force local source — matching what the workspace (../../go.work) does, but
// also making `go build` of this module deterministic on its own.
require (
	github.com/TicketsBot-cloud/archiverclient v0.0.0-20251015181023-f0b66a074704
	github.com/TicketsBot-cloud/common v0.0.0-20260620182815-55fda9a14c01
	github.com/TicketsBot-cloud/dashboard v0.0.0
	github.com/TicketsBot-cloud/database v0.0.0-20260423165031-495c2e8a5bc7
	github.com/TicketsBot-cloud/gdl v0.0.0-20260426095953-999472e6e538
	github.com/TicketsBot-cloud/worker v0.0.0-20260423165809-3a23e8fb9fc3
	github.com/joho/godotenv v1.5.1
	go.uber.org/zap v1.27.1
)

replace (
	github.com/TicketsBot-cloud/archiverclient => ../archiverclient
	github.com/TicketsBot-cloud/common => ../common
	github.com/TicketsBot-cloud/dashboard => ../dashboard
	github.com/TicketsBot-cloud/database => ../database
	github.com/TicketsBot-cloud/gdl => ../gdl
	github.com/TicketsBot-cloud/worker => ../worker
)
