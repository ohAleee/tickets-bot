package server

import (
	"strings"
	"time"

	permcache "github.com/TicketsBot-cloud/common/permission"
	"github.com/TicketsBot-cloud/worker/bot/button/registry"
	"github.com/TicketsBot-cloud/worker/bot/button/registry/matcher"
	"github.com/TicketsBot-cloud/worker/bot/command/context"
	"github.com/TicketsBot-cloud/worker/bot/customisation"
)

type AdminDebugServerEntitlementsHandler struct{}

func (h *AdminDebugServerEntitlementsHandler) Matcher() matcher.Matcher {
	return matcher.NewFuncMatcher(func(customId string) bool {
		return strings.HasPrefix(customId, "admin_debug_entitlements_")
	})
}

func (h *AdminDebugServerEntitlementsHandler) Properties() registry.Properties {
	return registry.Properties{
		Flags:           registry.SumFlags(registry.GuildAllowed, registry.CanEdit),
		Timeout:         time.Second * 10,
		PermissionLevel: permcache.Support,
		HelperOnly:      true,
	}
}

func (h *AdminDebugServerEntitlementsHandler) Execute(ctx *context.ButtonContext) {
	// Premium is force-unlocked and the entitlements table has been removed — nothing to show.
	// (The debug embed no longer renders this button, but the handler stays registered.)
	ctx.ReplyRaw(customisation.Orange, "No Entitlements", "Premium is enabled for all servers; there are no entitlements to display.")
}
