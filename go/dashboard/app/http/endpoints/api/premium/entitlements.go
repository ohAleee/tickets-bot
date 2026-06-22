package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetEntitlements previously listed the user's purchased subscriptions so they could be
// assigned to servers. Premium is now unconditional for every guild, the premium tables
// have been removed from the database, and there is nothing to manage — so this returns
// an empty, table-free response. The unchanged dashboard renders it as "no subscriptions".
func GetEntitlements(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"entitlements":       []any{},
		"legacy_entitlement": nil,
	})
}
