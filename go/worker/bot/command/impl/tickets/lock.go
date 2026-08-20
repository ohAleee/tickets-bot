package tickets

import (
	"fmt"

	permcache "github.com/TicketsBot-cloud/common/permission"
	"github.com/TicketsBot-cloud/gdl/objects/channel"
	"github.com/TicketsBot-cloud/gdl/objects/interaction"
	discordpermission "github.com/TicketsBot-cloud/gdl/permission"
	"github.com/TicketsBot-cloud/gdl/rest"
	"github.com/TicketsBot-cloud/gdl/rest/request"
	"github.com/TicketsBot-cloud/worker/bot/command"
	"github.com/TicketsBot-cloud/worker/bot/command/registry"
	"github.com/TicketsBot-cloud/worker/bot/constants"
	"github.com/TicketsBot-cloud/worker/bot/customisation"
	"github.com/TicketsBot-cloud/worker/bot/dbclient"
	"github.com/TicketsBot-cloud/worker/i18n"
)

// sendMessages is the permission revoked from the ticket opener and /add'ed members while locked.
// Discord requires SEND_MESSAGES for attachments and voice messages too, so denying it is enough.
var sendMessages = discordpermission.BuildPermissions(discordpermission.SendMessages)

type LockCommand struct {
}

func (LockCommand) Properties() registry.Properties {
	return registry.Properties{
		Name:            "lock",
		Description:     i18n.HelpLock,
		Type:            interaction.ApplicationCommandTypeChatInput,
		PermissionLevel: permcache.Support,
		Category:        command.Tickets,
		Timeout:         constants.TimeoutOpenTicket,
	}
}

func (c LockCommand) GetExecutor() interface{} {
	return c.Execute
}

func (LockCommand) Execute(ctx registry.CommandContext) {
	setTicketLock(ctx, true)
}

type UnlockCommand struct {
}

func (UnlockCommand) Properties() registry.Properties {
	return registry.Properties{
		Name:            "unlock",
		Description:     i18n.HelpUnlock,
		Type:            interaction.ApplicationCommandTypeChatInput,
		PermissionLevel: permcache.Support,
		Category:        command.Tickets,
		Timeout:         constants.TimeoutOpenTicket,
	}
}

func (c UnlockCommand) GetExecutor() interface{} {
	return c.Execute
}

func (UnlockCommand) Execute(ctx registry.CommandContext) {
	setTicketLock(ctx, false)
}

// ponytail: lock state lives purely in the channel overwrites, so /claim and /unclaim - which
// rebuild overwrites from scratch - reset it. Persist a flag on the ticket if that becomes a problem.
func setTicketLock(ctx registry.CommandContext, locked bool) {
	ticket, err := dbclient.Client.Tickets.GetByChannelAndGuild(ctx, ctx.ChannelId(), ctx.GuildId())
	if err != nil {
		ctx.HandleError(err)
		return
	}

	if ticket.UserId == 0 || ticket.ChannelId == nil {
		ctx.Reply(customisation.Red, i18n.Error, i18n.MessageNotATicketChannel)
		return
	}

	// Threads have no per-member permission overwrites
	if ticket.IsThread {
		ctx.Reply(customisation.Red, i18n.Error, i18n.MessageLockThread)
		return
	}

	members, err := dbclient.Client.TicketMembers.Get(ctx, ctx.GuildId(), ticket.Id)
	if err != nil {
		ctx.HandleError(err)
		return
	}

	targets := map[uint64]bool{ticket.UserId: true}
	for _, member := range members {
		targets[member] = true
	}

	ch, err := ctx.Worker().GetChannel(*ticket.ChannelId)
	if err != nil {
		ctx.HandleError(err)
		return
	}

	verb := "Unlocked"
	if locked {
		verb = "Locked"
	}

	auditReason := fmt.Sprintf("%s ticket %d", verb, ticket.Id)
	if member, err := ctx.Member(); err == nil {
		auditReason = fmt.Sprintf("%s ticket %d by %s", verb, ticket.Id, member.User.Username)
	}

	data := rest.ModifyChannelData{
		PermissionOverwrites: applyLock(ch.PermissionOverwrites, targets, locked),
	}

	reasonCtx := request.WithAuditReason(ctx, auditReason)
	if _, err := ctx.Worker().ModifyChannel(reasonCtx, *ticket.ChannelId, data); err != nil {
		ctx.HandleError(err)
		return
	}

	if locked {
		ctx.ReplyPermanent(customisation.Red, i18n.TitleLocked, i18n.MessageLocked, ctx.UserId())
	} else {
		ctx.ReplyPermanent(customisation.Green, i18n.TitleUnlocked, i18n.MessageUnlocked, ctx.UserId())
	}
}

// applyLock flips SEND_MESSAGES on the member overwrites of the given users, leaving staff
// (and any role overwrites) untouched.
func applyLock(overwrites []channel.PermissionOverwrite, targets map[uint64]bool, locked bool) []channel.PermissionOverwrite {
	out := make([]channel.PermissionOverwrite, len(overwrites))
	copy(out, overwrites)

	for i, overwrite := range out {
		if overwrite.Type != channel.PermissionTypeMember || !targets[overwrite.Id] {
			continue
		}

		if locked {
			out[i].Allow &^= sendMessages
			out[i].Deny |= sendMessages
		} else {
			out[i].Allow |= sendMessages
			out[i].Deny &^= sendMessages
		}
	}

	return out
}
