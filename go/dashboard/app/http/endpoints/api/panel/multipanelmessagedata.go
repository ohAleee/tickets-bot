package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/TicketsBot-cloud/dashboard/botcontext"
	"github.com/TicketsBot-cloud/dashboard/config"
	"github.com/TicketsBot-cloud/dashboard/utils/types"
	"github.com/TicketsBot-cloud/database"
	"github.com/TicketsBot-cloud/gdl/objects/channel/embed"
	"github.com/TicketsBot-cloud/gdl/objects/channel/message"
	"github.com/TicketsBot-cloud/gdl/objects/interaction/component"
	"github.com/TicketsBot-cloud/gdl/rest"
	"github.com/TicketsBot-cloud/gdl/utils"
)

type multiPanelMessageData struct {
	IsPremium bool

	ChannelId uint64

	SelectMenu            bool
	SelectMenuPlaceholder *string

	Embed *embed.Embed

	// Components, when non-empty, is a Discord "Components V2" layout designed in the
	// dashboard editor. It is rendered in place of the embed; the category picker (select
	// menu / buttons) is appended to it automatically.
	Components []component.Component
}

func multiPanelDiscordSubPanelError(action, detail string) string {
	return fmt.Sprintf(
		"Failed to %s multi-panel message. One of the included panels may be misconfigured: %s",
		action,
		detail,
	)
}

// parseComponentsV2 decodes the stored/submitted Components V2 layout. An empty or null
// payload means the panel uses the classic embed rendering.
func parseComponentsV2(raw json.RawMessage) ([]component.Component, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var components []component.Component
	if err := json.Unmarshal(raw, &components); err != nil {
		return nil, err
	}

	return components, nil
}

func multiPanelIntoMessageData(panel database.MultiPanel, isPremium bool) multiPanelMessageData {
	// Best-effort: a malformed stored layout falls back to the classic embed rendering
	// rather than breaking resends.
	components, _ := parseComponentsV2(panel.Components)

	return multiPanelMessageData{
		IsPremium: isPremium,

		ChannelId: panel.ChannelId,

		SelectMenu:            panel.SelectMenu,
		SelectMenuPlaceholder: panel.SelectMenuPlaceholder,
		Embed:                 types.NewCustomEmbed(panel.Embed.CustomEmbed, panel.Embed.Fields).IntoDiscordEmbed(),
		Components:            components,
	}
}

func getEffectiveLabel(panel database.Panel, customLabel *string) string {
	if customLabel != nil && *customLabel != "" {
		return *customLabel
	}
	return panel.ButtonLabel
}

func getEffectiveEmoji(panel database.Panel, customEmojiName *string, customEmojiId *uint64) *string {
	if customEmojiId != nil && *customEmojiId != 0 {
		if customEmojiName != nil && *customEmojiName != "" {
			return customEmojiName
		}
		return nil
	}
	if customEmojiName != nil && *customEmojiName != "" {
		return customEmojiName
	}
	return panel.EmojiName
}

func getEffectiveEmojiId(panel database.Panel, customEmojiName *string, customEmojiId *uint64) *uint64 {
	if customEmojiId != nil && *customEmojiId != 0 {
		return customEmojiId
	}
	if customEmojiName != nil && *customEmojiName != "" {
		return nil
	}
	return panel.EmojiId
}

func (d *multiPanelMessageData) usesComponentsV2() bool {
	return len(d.Components) > 0
}

// buildCategoryComponents builds the interactive category picker: a single select menu
// (dropdown mode) or one or more rows of buttons.
func (d *multiPanelMessageData) buildCategoryComponents(panels []database.PanelWithCustomization) []component.Component {
	if d.SelectMenu {
		options := make([]component.SelectOption, len(panels))
		for i, pwc := range panels {
			effectiveEmojiName := getEffectiveEmoji(pwc.Panel, pwc.CustomEmojiName, pwc.CustomEmojiId)
			effectiveEmojiId := getEffectiveEmojiId(pwc.Panel, pwc.CustomEmojiName, pwc.CustomEmojiId)
			emoji := types.NewEmoji(effectiveEmojiName, effectiveEmojiId).IntoGdl()

			options[i] = component.SelectOption{
				Label:       getEffectiveLabel(pwc.Panel, pwc.CustomLabel),
				Value:       pwc.CustomId,
				Description: pwc.Description,
				Emoji:       emoji,
			}
		}

		placeholder := "Select a topic..."
		if d.SelectMenuPlaceholder != nil {
			placeholder = *d.SelectMenuPlaceholder
		}

		return []component.Component{
			component.BuildActionRow(
				component.BuildSelectMenu(
					component.SelectMenu{
						CustomId:    "multipanel",
						Options:     options,
						Placeholder: placeholder,
						MinValues:   utils.IntPtr(1),
						MaxValues:   utils.IntPtr(1),
						Disabled:    false,
					}),
			),
		}
	}

	buttons := make([]component.Component, len(panels))
	for i, pwc := range panels {
		effectiveEmojiName := getEffectiveEmoji(pwc.Panel, pwc.CustomEmojiName, pwc.CustomEmojiId)
		effectiveEmojiId := getEffectiveEmojiId(pwc.Panel, pwc.CustomEmojiName, pwc.CustomEmojiId)
		emoji := types.NewEmoji(effectiveEmojiName, effectiveEmojiId).IntoGdl()

		buttons[i] = component.BuildButton(component.Button{
			Label:    getEffectiveLabel(pwc.Panel, pwc.CustomLabel),
			CustomId: pwc.CustomId,
			Style:    component.ButtonStyle(pwc.ButtonStyle),
			Emoji:    emoji,
			Disabled: pwc.Disabled,
		})
	}

	var rows []component.Component
	for i := 0; i <= int(math.Ceil(float64(len(buttons)/5))); i++ {
		lb := i * 5
		ub := lb + 5

		if ub >= len(buttons) {
			ub = len(buttons)
		}

		if lb >= ub {
			break
		}

		row := component.BuildActionRow(buttons[lb:ub]...)
		rows = append(rows, row)
	}

	return rows
}

// assembleComponentsV2 combines the custom layout with the category picker. The picker is
// placed inside the trailing container when the layout ends with one (so it sits inside the
// coloured card), otherwise it is appended at the top level.
func (d *multiPanelMessageData) assembleComponentsV2(panels []database.PanelWithCustomization) []component.Component {
	category := d.buildCategoryComponents(panels)

	components := make([]component.Component, len(d.Components))
	copy(components, d.Components)

	if n := len(components); n > 0 {
		if container, ok := components[n-1].ComponentData.(component.Container); ok {
			container.Components = append(container.Components, category...)
			components[n-1] = component.BuildContainer(container)
		} else {
			components = append(components, category...)
		}
	} else {
		components = append(components, category...)
	}

	if !d.IsPremium {
		components = append(components, component.BuildTextDisplay(component.TextDisplay{
			Content: fmt.Sprintf("-# Powered by %s", config.Conf.Bot.PoweredBy),
		}))
	}

	return components
}

func (d *multiPanelMessageData) send(ctx *botcontext.BotContext, panels []database.PanelWithCustomization) (uint64, error) {
	var data rest.CreateMessageData
	if d.usesComponentsV2() {
		data = rest.CreateMessageData{
			Components: d.assembleComponentsV2(panels),
			Flags:      uint(message.FlagComponentsV2),
		}
	} else {
		if !d.IsPremium {
			d.Embed.SetFooter(fmt.Sprintf("Powered by %s", config.Conf.Bot.PoweredBy), config.Conf.Bot.IconUrl)
		}

		data = rest.CreateMessageData{
			Embeds:     []*embed.Embed{d.Embed},
			Components: d.buildCategoryComponents(panels),
		}
	}

	// TODO: Use proper context
	msg, err := rest.CreateMessage(context.Background(), ctx.Token, ctx.RateLimiter, d.ChannelId, data)
	if err != nil {
		return 0, err
	}

	return msg.Id, nil
}

func (d *multiPanelMessageData) edit(ctx *botcontext.BotContext, messageId uint64, panels []database.PanelWithCustomization) error {
	var data rest.EditMessageData
	if d.usesComponentsV2() {
		data = rest.EditMessageData{
			Components: d.assembleComponentsV2(panels),
			Flags:      uint(message.FlagComponentsV2),
		}
	} else {
		if !d.IsPremium {
			d.Embed.SetFooter(fmt.Sprintf("Powered by %s", config.Conf.Bot.PoweredBy), config.Conf.Bot.IconUrl)
		}

		data = rest.EditMessageData{
			Embeds:     []*embed.Embed{d.Embed},
			Components: d.buildCategoryComponents(panels),
		}
	}

	_, err := rest.EditMessage(context.Background(), ctx.Token, ctx.RateLimiter, d.ChannelId, messageId, data)
	return err
}
