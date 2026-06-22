package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetActiveGuilds previously assigned a multi-server subscription to specific guilds.
// Premium is now unconditional for every guild and the premium tables have been removed,
// so there are no per-guild entitlements to set — this is a table-free no-op that keeps
// the unchanged dashboard happy (it expects 204 on success).
func SetActiveGuilds(ctx *gin.Context) {
	ctx.Status(http.StatusNoContent)
}
