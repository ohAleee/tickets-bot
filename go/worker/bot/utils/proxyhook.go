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
	if strings.Contains(req.URL.Path, "/oauth2/") {
		return
	}

	if !strings.HasPrefix(req.URL.Path, "/api/v9/applications/") {
		req.URL.Scheme = "http"
		req.URL.Host = config.Conf.Discord.ProxyUrl
	}
}
