package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/TicketsBot-cloud/common/premium"
	"github.com/TicketsBot-cloud/dashboard/app"
	"github.com/TicketsBot-cloud/dashboard/app/http/audit"
	"github.com/TicketsBot-cloud/dashboard/app/http/validation"
	"github.com/TicketsBot-cloud/dashboard/botcontext"
	dbclient "github.com/TicketsBot-cloud/dashboard/database"
	"github.com/TicketsBot-cloud/dashboard/rpc"
	"github.com/TicketsBot-cloud/dashboard/rpc/cache"
	"github.com/TicketsBot-cloud/dashboard/utils"
	"github.com/TicketsBot-cloud/dashboard/utils/types"
	"github.com/TicketsBot-cloud/database"
	"github.com/TicketsBot-cloud/gdl/objects/channel"
	"github.com/TicketsBot-cloud/gdl/rest/request"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/sync/errgroup"
)

type panelConfiguration struct {
	PanelId         int     `json:"panel_id"`
	CustomEmojiName *string `json:"custom_emoji_name" validate:"omitempty,max=32"`
	CustomEmojiId   *uint64 `json:"custom_emoji_id,string"`
	CustomLabel     *string `json:"custom_label" validate:"omitempty,max=80"`
	Description     *string `json:"description" validate:"omitempty,max=100"`
}

func getEffectiveLabelForValidation(buttonLabel string, customLabel *string) string {
	if customLabel != nil && *customLabel != "" {
		return *customLabel
	}
	return buttonLabel
}

type multiPanelCreateData struct {
	ChannelId             uint64               `json:"channel_id,string"`
	SelectMenu            bool                 `json:"select_menu"`
	SelectMenuPlaceholder *string              `json:"select_menu_placeholder,omitempty" validate:"omitempty,max=150"`
	Panels                []panelConfiguration `json:"panels" validate:"dive"`
	Embed                 *types.CustomEmbed   `json:"embed" validate:"omitempty,dive"`
	// Components is an optional Discord "Components V2" layout designed in the dashboard
	// editor. When present, it replaces the embed as the panel message body.
	Components json.RawMessage `json:"components,omitempty"`
}

func (d *multiPanelCreateData) IntoMessageData(isPremium bool) multiPanelMessageData {
	// The layout is validated before we reach here, so a parse error is treated as "no V2
	// layout" and we fall back to the embed.
	blocks, _ := parseCV2Blocks(d.Components)

	data := multiPanelMessageData{
		IsPremium:             isPremium,
		ChannelId:             d.ChannelId,
		SelectMenu:            d.SelectMenu,
		SelectMenuPlaceholder: d.SelectMenuPlaceholder,
		Blocks:                blocks,
	}

	if len(blocks) == 0 && d.Embed != nil {
		data.Embed = d.Embed.IntoDiscordEmbed()
	}

	return data
}

func MultiPanelCreate(c *gin.Context) {
	guildId := c.Keys["guildid"].(uint64)
	userId := c.Keys["userid"].(uint64)

	var data multiPanelCreateData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, utils.ErrorStr("Invalid request data. Please check your input and try again."))
		return
	}

	if err := validate.Struct(data); err != nil {
		var validationErrors validator.ValidationErrors
		if ok := errors.As(err, &validationErrors); !ok {
			_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "An error occurred while validating the panel"))
			return
		}

		formatted := "Your input contained the following errors:\n" + utils.FormatValidationErrors(validationErrors)
		c.JSON(400, utils.ErrorStr(formatted))
		return
	}

	// validate body & get sub-panels
	panels, err := data.doValidations(guildId)
	if err != nil {
		var validationError *validation.InvalidInputError
		if errors.As(err, &validationError) {
			c.JSON(400, utils.ErrorStr("%s", validationError.Error()))
		} else {
			_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to create multi-panel"))
		}
		return
	}

	// Validate labels for dropdown mode
	if data.SelectMenu {
		for _, panel := range panels {
			panelConfig := data.Panels[0]
			for i, cfg := range data.Panels {
				if panels[i].PanelId == cfg.PanelId {
					panelConfig = cfg
					break
				}
			}

			effectiveLabel := getEffectiveLabelForValidation(panel.ButtonLabel, panelConfig.CustomLabel)

			if effectiveLabel == "" {
				c.JSON(400, utils.ErrorStr(fmt.Sprintf("Panel '%s' must have a label when using dropdown mode. Please add a custom label or ensure the panel has a button label.", panel.Title)))
				return
			}
		}
	}

	botContext, err := botcontext.ContextForGuild(guildId)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Unable to connect to Discord. Please try again later."))
		return
	}

	premiumTier, err := rpc.PremiumClient.GetTierByGuildId(c, guildId, true, botContext.Token, botContext.RateLimiter)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to create multi-panel"))
		return
	}

	// Create PanelWithCustomization by combining panels with their configurations
	panelsWithCustom := make([]database.PanelWithCustomization, len(panels))
	for i, panel := range panels {
		panelsWithCustom[i] = database.PanelWithCustomization{
			Panel:           panel,
			CustomLabel:     data.Panels[i].CustomLabel,
			Description:     data.Panels[i].Description,
			CustomEmojiName: data.Panels[i].CustomEmojiName,
			CustomEmojiId:   data.Panels[i].CustomEmojiId,
		}
	}

	messageData := data.IntoMessageData(premiumTier > premium.None)
	messageId, err := messageData.send(botContext, panelsWithCustom)
	if err != nil {
		var unwrapped request.RestError
		if errors.As(err, &unwrapped) {
			if unwrapped.StatusCode == 403 {
				c.JSON(http.StatusBadRequest, utils.ErrorStr("I do not have permission to send messages in the provided channel"))
			} else {
				c.JSON(http.StatusBadRequest, utils.ErrorStr("%s", multiPanelDiscordSubPanelError("send", unwrapped.ApiError.Message)))
			}
		} else {
			_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to create multi-panel"))
		}

		return
	}

	dbEmbed, dbEmbedFields := data.Embed.IntoDatabaseStruct()
	multiPanel := database.MultiPanel{
		MessageId:             messageId,
		ChannelId:             data.ChannelId,
		GuildId:               guildId,
		SelectMenu:            data.SelectMenu,
		SelectMenuPlaceholder: data.SelectMenuPlaceholder,
		Embed: &database.CustomEmbedWithFields{
			CustomEmbed: dbEmbed,
			Fields:      dbEmbedFields,
		},
		Components: data.Components,
	}

	multiPanel.Id, err = dbclient.Client.MultiPanels.Create(c, multiPanel)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to create multi-panel"))
		return
	}

	group, _ := errgroup.WithContext(context.Background())
	for i, panel := range panels {
		i := i
		panel := panel

		// Find matching panel config by panel_id
		var panelConfig *panelConfiguration
		for _, cfg := range data.Panels {
			if cfg.PanelId == panel.PanelId {
				panelConfig = &cfg
				break
			}
		}

		group.Go(func() error {
			if panelConfig != nil {
				return dbclient.Client.MultiPanelTargets.Insert(c, multiPanel.Id, panel.PanelId, i, panelConfig.CustomLabel, panelConfig.Description, panelConfig.CustomEmojiName, panelConfig.CustomEmojiId)
			} else {
				return dbclient.Client.MultiPanelTargets.Insert(c, multiPanel.Id, panel.PanelId, i, nil, nil, nil, nil)
			}
		})
	}

	if err := group.Wait(); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, app.NewError(err, "Failed to create multi-panel"))
		return
	}

	audit.Log(audit.LogEntry{
		GuildId:      audit.Uint64Ptr(guildId),
		UserId:       userId,
		ActionType:   database.AuditActionMultiPanelCreate,
		ResourceType: database.AuditResourceMultiPanel,
		ResourceId:   audit.StringPtr(fmt.Sprintf("%d", multiPanel.Id)),
		NewData:      data,
	})
	c.JSON(200, gin.H{
		"success": true,
		"data":    multiPanel,
	})
}

func (d *multiPanelCreateData) doValidations(guildId uint64) (panels []database.Panel, err error) {
	blocks, parseErr := parseCV2Blocks(d.Components)
	if parseErr != nil {
		return nil, validation.NewInvalidInputError("The message layout is not valid JSON")
	}

	isV2 := len(blocks) > 0
	if !isV2 {
		if err := validateEmbed(d.Embed); err != nil {
			return nil, err
		}
	}

	group, _ := errgroup.WithContext(context.Background())

	group.Go(d.validateChannel(guildId))
	group.Go(func() (e error) {
		panels, e = d.validatePanels(guildId)
		return
	})

	if err = group.Wait(); err != nil {
		return
	}

	// Ticket buttons reference sub-panels by id, so the layout can only be validated once the
	// panel set is known.
	if isV2 {
		validIds := make(map[int]bool, len(panels))
		for _, p := range panels {
			validIds[p.PanelId] = true
		}
		if err = validateCV2(blocks, validIds); err != nil {
			return
		}
	}

	return
}

func (d *multiPanelCreateData) validateChannel(guildId uint64) func() error {
	return func() error {
		// TODO: Use proper context
		channels, err := cache.Instance.GetGuildChannels(context.Background(), guildId)
		if err != nil {
			return err
		}

		var valid bool
		for _, ch := range channels {
			if ch.Id == d.ChannelId && (ch.Type == channel.ChannelTypeGuildText || ch.Type == channel.ChannelTypeGuildNews) {
				valid = true
				break
			}
		}

		if !valid {
			return validation.NewInvalidInputError("The selected channel does not exist")
		}

		return nil
	}
}

func (d *multiPanelCreateData) validatePanels(guildId uint64) (panels []database.Panel, err error) {
	if len(d.Panels) < 2 {
		err = validation.NewInvalidInputError("a multi-panel must contain at least 2 sub-panels")
		return
	}

	if len(d.Panels) > 15 {
		err = validation.NewInvalidInputError("multi-panels cannot contain more than 15 sub-panels")
		return
	}

	existingPanels, err := dbclient.Client.Panel.GetByGuild(context.Background(), guildId)
	if err != nil {
		return nil, err
	}

	for _, panelConfig := range d.Panels {
		var valid bool
		// find panel struct
		for _, panel := range existingPanels {
			if panel.PanelId == panelConfig.PanelId {
				valid = true
				panels = append(panels, panel)
			}
		}

		if !valid {
			return nil, validation.NewInvalidInputError("invalid panel ID")
		}
	}

	return
}
