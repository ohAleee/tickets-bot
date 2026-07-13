package utils

import (
	"net/http"
	"strings"

	"github.com/TicketsBot-cloud/worker/config"
)

// Twilight's HTTP proxy doesn't support the typical HTTP proxy protocol - instead you send the request directly
// to the proxy's host in the URL. This is not how Go's proxy function should be used, but it works :)
//
// In the unified binary this hook is registered globally alongside the dashboard's, so it
// also sees the dashboard's OAuth2 token exchange. OAuth2 endpoints must go directly to
// Discord (the Twilight proxy only routes bot API paths and 500s on /oauth2/token).
func ProxyHook(token string, req *http.Request) {
	// OAuth2 and application-scoped endpoints must reach Discord directly; the
	// Twilight proxy only routes bot API paths. Match on path segments rather
	// than a hardcoded API version (BaseUrl has since moved from v9 to v10, which
	// silently broke the old "/api/v9/applications/" prefix check).
	if strings.Contains(req.URL.Path, "/oauth2/") || strings.Contains(req.URL.Path, "/applications/") {
		return
	}

	req.URL.Scheme = "http"
	req.URL.Host = config.Conf.Discord.ProxyUrl
}
