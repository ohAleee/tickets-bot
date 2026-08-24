package api

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/TicketsBot-cloud/dashboard/app/http/validation"
	"github.com/TicketsBot-cloud/dashboard/utils/types"
	"github.com/TicketsBot-cloud/database"
	"github.com/TicketsBot-cloud/gdl/objects/guild/emoji"
	"github.com/TicketsBot-cloud/gdl/objects/interaction/component"
)

// The multi-panel "Components V2" layout is stored as a small semantic model (not raw Discord
// component JSON) so that "ticket buttons" - buttons that open one of the multi-panel's
// sub-panels - can be resolved to the sub-panel's custom id at send time, and placed anywhere
// in the layout. This mirrors the probot /embeds editor's model.
//
// Supported blocks: text, separator, gallery, section, buttons, container.
// Buttons come in two kinds: "link" (a URL button) and "ticket" (opens a sub-panel).

const (
	maxV2Blocks      = 40
	maxV2Children    = 20
	maxV2TextLength  = 4000
	maxV2MediaItems  = 10
	maxV2RowButtons  = 5
	maxV2ButtonLabel = 80
)

type cv2Block struct {
	Type string `json:"type"`

	// text, section
	Content string `json:"content,omitempty"`

	// separator
	Divider *bool  `json:"divider,omitempty"`
	Spacing string `json:"spacing,omitempty"` // "small" | "large"

	// gallery
	Items []cv2GalleryItem `json:"items,omitempty"`

	// section
	Accessory *cv2Accessory `json:"accessory,omitempty"`

	// buttons
	Buttons []cv2Button `json:"buttons,omitempty"`

	// container
	AccentColor *int       `json:"accentColor,omitempty"`
	Children    []cv2Block `json:"children,omitempty"`
}

type cv2GalleryItem struct {
	Url         string  `json:"url"`
	Description *string `json:"description,omitempty"`
}

type cv2Accessory struct {
	Kind        string     `json:"kind"` // "image" | "button"
	Url         string     `json:"url,omitempty"`
	Description *string    `json:"description,omitempty"`
	Button      *cv2Button `json:"button,omitempty"`
}

type cv2Button struct {
	Kind    string  `json:"kind"` // "link" | "ticket"
	Label   string  `json:"label,omitempty"`
	Emoji   *string `json:"emoji,omitempty"`
	Url     string  `json:"url,omitempty"`     // link
	PanelId *int    `json:"panelId,omitempty"` // ticket
	Style   string  `json:"style,omitempty"`   // ticket: primary|secondary|success|danger
}

// cv2ButtonStyle maps a semantic colour name to a Discord button style, falling back to the
// sub-panel's own style when unset.
func cv2ButtonStyle(name string, fallback component.ButtonStyle) component.ButtonStyle {
	switch name {
	case "primary":
		return component.ButtonStylePrimary
	case "secondary":
		return component.ButtonStyleSecondary
	case "success":
		return component.ButtonStyleSuccess
	case "danger":
		return component.ButtonStyleDanger
	}
	return fallback
}

// parseCV2Blocks decodes the stored/submitted layout. An empty/null payload means the classic
// embed rendering is used.
func parseCV2Blocks(raw json.RawMessage) ([]cv2Block, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var blocks []cv2Block
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, err
	}

	return blocks, nil
}

// blocksContainTicketButton reports whether the layout already places at least one ticket
// button. When it does, the renderer does not auto-append the default category buttons.
func blocksContainTicketButton(blocks []cv2Block) bool {
	for _, b := range blocks {
		switch b.Type {
		case "buttons":
			for _, btn := range b.Buttons {
				if btn.Kind == "ticket" {
					return true
				}
			}
		case "section":
			if b.Accessory != nil && b.Accessory.Kind == "button" && b.Accessory.Button != nil && b.Accessory.Button.Kind == "ticket" {
				return true
			}
		case "container":
			if blocksContainTicketButton(b.Children) {
				return true
			}
		}
	}
	return false
}

// ---- rendering ----

func parseCV2Emoji(raw *string) *emoji.Emoji {
	if raw == nil {
		return nil
	}

	v := strings.TrimSpace(*raw)
	if v == "" {
		return nil
	}

	// Custom emoji: <:name:id> or <a:name:id>
	if strings.HasPrefix(v, "<") && strings.HasSuffix(v, ">") {
		parts := strings.Split(strings.Trim(v, "<>"), ":")
		if len(parts) == 3 {
			if id, err := strconv.ParseUint(parts[2], 10, 64); err == nil {
				name := parts[1]
				return types.NewEmoji(&name, &id).IntoGdl()
			}
		}
		return nil
	}

	// Unicode emoji
	return types.NewEmoji(&v, nil).IntoGdl()
}

func buildCV2Button(b cv2Button, panelMap map[int]database.PanelWithCustomization) *component.Button {
	switch b.Kind {
	case "link":
		url := strings.TrimSpace(b.Url)
		label := strings.TrimSpace(b.Label)
		emj := parseCV2Emoji(b.Emoji)
		if url == "" || (label == "" && emj == nil) {
			return nil
		}
		return &component.Button{
			Style: component.ButtonStyleLink,
			Url:   &url,
			Label: label,
			Emoji: emj,
		}

	case "ticket":
		if b.PanelId == nil {
			return nil
		}
		pwc, ok := panelMap[*b.PanelId]
		if !ok || pwc.CustomId == "" {
			return nil
		}

		label := strings.TrimSpace(b.Label)
		if label == "" {
			label = getEffectiveLabel(pwc.Panel, pwc.CustomLabel)
		}

		var emj *emoji.Emoji
		if b.Emoji != nil && strings.TrimSpace(*b.Emoji) != "" {
			emj = parseCV2Emoji(b.Emoji)
		} else {
			emj = types.NewEmoji(
				getEffectiveEmoji(pwc.Panel, pwc.CustomEmojiName, pwc.CustomEmojiId),
				getEffectiveEmojiId(pwc.Panel, pwc.CustomEmojiName, pwc.CustomEmojiId),
			).IntoGdl()
		}

		style := cv2ButtonStyle(b.Style, component.ButtonStyle(pwc.ButtonStyle))
		if style == 0 || style == component.ButtonStyleLink {
			style = component.ButtonStylePrimary
		}

		return &component.Button{
			Style:    style,
			CustomId: pwc.CustomId,
			Label:    label,
			Emoji:    emj,
		}
	}

	return nil
}

// buildCV2Leaf converts one block into a container-compatible component. The bool is false when
// the block has no renderable content.
func buildCV2Leaf(b cv2Block, panelMap map[int]database.PanelWithCustomization) (component.Component, bool) {
	switch b.Type {
	case "text":
		if strings.TrimSpace(b.Content) == "" {
			return component.Component{}, false
		}
		return component.BuildTextDisplay(component.TextDisplay{Content: b.Content}), true

	case "separator":
		sep := component.Separator{}
		if b.Divider != nil {
			sep.Divider = b.Divider
		}
		spacing := 1
		if b.Spacing == "large" {
			spacing = 2
		}
		sep.Spacing = &spacing
		return component.BuildSeparator(sep), true

	case "gallery":
		var items []component.MediaGalleryItem
		for _, it := range b.Items {
			url := strings.TrimSpace(it.Url)
			if url == "" {
				continue
			}
			item := component.MediaGalleryItem{Media: component.UnfurledMediaItem{Url: url}}
			if it.Description != nil && *it.Description != "" {
				item.Description = it.Description
			}
			items = append(items, item)
			if len(items) >= maxV2MediaItems {
				break
			}
		}
		if len(items) == 0 {
			return component.Component{}, false
		}
		return component.BuildMediaGallery(component.MediaGallery{Items: items}), true

	case "buttons":
		var buttons []component.Component
		for _, bb := range b.Buttons {
			if btn := buildCV2Button(bb, panelMap); btn != nil {
				buttons = append(buttons, component.BuildButton(*btn))
			}
			if len(buttons) >= maxV2RowButtons {
				break
			}
		}
		if len(buttons) == 0 {
			return component.Component{}, false
		}
		return component.BuildActionRow(buttons...), true

	case "section":
		text := b.Content
		if strings.TrimSpace(text) == "" {
			return component.Component{}, false
		}

		var accessory component.Component
		hasAccessory := false
		if b.Accessory != nil {
			switch b.Accessory.Kind {
			case "image":
				url := strings.TrimSpace(b.Accessory.Url)
				if url != "" {
					thumb := component.Thumbnail{Media: component.UnfurledMediaItem{Url: url}}
					if b.Accessory.Description != nil && *b.Accessory.Description != "" {
						thumb.Description = b.Accessory.Description
					}
					accessory = component.BuildThumbnail(thumb)
					hasAccessory = true
				}
			case "button":
				if b.Accessory.Button != nil {
					if btn := buildCV2Button(*b.Accessory.Button, panelMap); btn != nil {
						accessory = component.BuildButton(*btn)
						hasAccessory = true
					}
				}
			}
		}

		// Discord requires a section to have an accessory; without a usable one, fall back to a
		// plain text block.
		if !hasAccessory {
			return component.BuildTextDisplay(component.TextDisplay{Content: text}), true
		}

		return component.BuildSection(component.Section{
			Components: []component.Component{component.BuildTextDisplay(component.TextDisplay{Content: text})},
			Accessory:  accessory,
		}), true

	case "container":
		var children []component.Component
		for _, ch := range b.Children {
			if c, ok := buildCV2Leaf(ch, panelMap); ok {
				children = append(children, c)
			}
		}
		if len(children) == 0 {
			return component.Component{}, false
		}
		cont := component.Container{Components: children}
		if b.AccentColor != nil {
			cont.AccentColor = b.AccentColor
		}
		return component.BuildContainer(cont), true
	}

	return component.Component{}, false
}

func buildCV2Blocks(blocks []cv2Block, panelMap map[int]database.PanelWithCustomization) []component.Component {
	var out []component.Component
	for _, b := range blocks {
		if c, ok := buildCV2Leaf(b, panelMap); ok {
			out = append(out, c)
		}
	}
	return out
}

// ---- validation ----

func validateCV2(blocks []cv2Block, validPanelIds map[int]bool) error {
	if len(blocks) > maxV2Blocks {
		return validation.NewInvalidInputError("The message has too many components (maximum 40)")
	}
	return validateCV2Nodes(blocks, validPanelIds, 0)
}

func validateCV2Nodes(blocks []cv2Block, validPanelIds map[int]bool, depth int) error {
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if err := validateCV2Text(b.Content); err != nil {
				return err
			}
		case "separator":
			// nothing to validate
		case "gallery":
			if len(b.Items) > maxV2MediaItems {
				return validation.NewInvalidInputError("Media galleries cannot contain more than 10 images")
			}
			for _, it := range b.Items {
				if err := validateCV2URL(it.Url); err != nil {
					return err
				}
			}
		case "section":
			if err := validateCV2Text(b.Content); err != nil {
				return err
			}
			if b.Accessory != nil {
				switch b.Accessory.Kind {
				case "image":
					if err := validateCV2URL(b.Accessory.Url); err != nil {
						return err
					}
				case "button":
					if b.Accessory.Button != nil {
						if err := validateCV2Button(*b.Accessory.Button, validPanelIds); err != nil {
							return err
						}
					}
				case "":
					// treated as no accessory
				default:
					return validation.NewInvalidInputError("Unknown section accessory type")
				}
			}
		case "buttons":
			if len(b.Buttons) > maxV2RowButtons {
				return validation.NewInvalidInputError("A button row cannot contain more than 5 buttons")
			}
			for _, btn := range b.Buttons {
				if err := validateCV2Button(btn, validPanelIds); err != nil {
					return err
				}
			}
		case "container":
			if depth > 0 {
				return validation.NewInvalidInputError("Containers cannot be nested")
			}
			if len(b.Children) > maxV2Children {
				return validation.NewInvalidInputError("A container cannot hold more than 20 components")
			}
			if err := validateCV2Nodes(b.Children, validPanelIds, depth+1); err != nil {
				return err
			}
		default:
			return validation.NewInvalidInputError("The message contains an unsupported component type")
		}
	}
	return nil
}

func validateCV2Text(content string) error {
	if len([]rune(content)) > maxV2TextLength {
		return validation.NewInvalidInputError("A text component is too long (maximum 4000 characters)")
	}
	return nil
}

func validateCV2Button(b cv2Button, validPanelIds map[int]bool) error {
	if len([]rune(b.Label)) > maxV2ButtonLabel {
		return validation.NewInvalidInputError("A button label is too long (maximum 80 characters)")
	}
	switch b.Kind {
	case "link":
		if b.Url != "" {
			if err := validateCV2URL(b.Url); err != nil {
				return err
			}
		}
	case "ticket":
		if b.PanelId == nil || !validPanelIds[*b.PanelId] {
			return validation.NewInvalidInputError("A ticket button references a panel that is not part of this multi-panel")
		}
	case "":
		// lenient: an unset draft button
	default:
		return validation.NewInvalidInputError("Unknown button type")
	}
	return nil
}

func validateCV2URL(url string) error {
	if url == "" {
		return nil
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return validation.NewInvalidInputError("URLs must start with http:// or https://")
	}
	return nil
}
