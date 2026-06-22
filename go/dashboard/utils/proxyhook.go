package utils

import (
	"net/http"
	"strings"

	"github.com/TicketsBot-cloud/dashboard/config"
)

// Twilight's HTTP proxy doesn't support the typical HTTP proxy protocol - instead you send the request directly
// to the proxy's host in the URL. This is not how Go's proxy function should be used, but it works :)
//
// OAuth2 endpoints (token exchange / refresh) must go directly to Discord: the Twilight
// proxy only routes bot API paths and 500s on /oauth2/token. The original Basic-auth guard
// checked req.Header (not reliably set when this hook runs), so guard on the URL path and
// the token argument instead.
func ProxyHook(token string, req *http.Request) {
	if strings.Contains(req.URL.Path, "/oauth2/") || strings.HasPrefix(token, "Basic") {
		return
	}

	req.URL.Scheme = "http"
	req.URL.Host = config.Conf.Bot.ProxyUrl
}
