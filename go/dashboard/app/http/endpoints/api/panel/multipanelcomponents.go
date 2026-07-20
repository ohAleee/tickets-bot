package api

import (
	"strings"

	"github.com/TicketsBot-cloud/dashboard/app/http/validation"
	"github.com/TicketsBot-cloud/gdl/objects/interaction/component"
)

const (
	maxV2Components   = 40
	maxV2TextLength   = 4000
	maxV2MediaItems   = 10
	maxV2NestingDepth = 2
)

// validateComponentsV2 walks a Components V2 layout submitted by the dashboard editor and
// rejects anything the multi-panel renderer can't safely turn into a Discord message.
//
// Only non-interactive display components are permitted: containers, text displays,
// sections (with an image thumbnail accessory), media galleries and separators. Interactive
// components (buttons, select menus, action rows) are intentionally disallowed — the panel's
// category picker is appended automatically, and stray interactive components would either be
// dead or clash with the "multipanel" custom id.
func validateComponentsV2(components []component.Component) error {
	count := 0
	return validateV2Nodes(components, 0, &count)
}

func validateV2Nodes(components []component.Component, depth int, count *int) error {
	for _, c := range components {
		*count++
		if *count > maxV2Components {
			return validation.NewInvalidInputError("The message has too many components (maximum 40)")
		}

		switch data := c.ComponentData.(type) {
		case component.Container:
			if depth >= maxV2NestingDepth-1 {
				return validation.NewInvalidInputError("Containers cannot be nested")
			}
			if err := validateV2Nodes(data.Components, depth+1, count); err != nil {
				return err
			}
		case component.Section:
			if err := validateV2Section(data, count); err != nil {
				return err
			}
		case component.TextDisplay:
			if err := validateV2Text(data.Content); err != nil {
				return err
			}
		case component.MediaGallery:
			if err := validateV2MediaGallery(data); err != nil {
				return err
			}
		case component.Separator:
			// nothing to validate
		default:
			return validation.NewInvalidInputError("The message contains an unsupported component type")
		}
	}

	return nil
}

func validateV2Section(section component.Section, count *int) error {
	if len(section.Components) == 0 {
		return validation.NewInvalidInputError("Sections must contain at least one text component")
	}

	for _, sub := range section.Components {
		*count++
		text, ok := sub.ComponentData.(component.TextDisplay)
		if !ok {
			return validation.NewInvalidInputError("Sections may only contain text components")
		}
		if err := validateV2Text(text.Content); err != nil {
			return err
		}
	}

	thumbnail, ok := section.Accessory.ComponentData.(component.Thumbnail)
	if !ok {
		return validation.NewInvalidInputError("Sections must have an image accessory")
	}
	*count++
	if err := validateV2MediaURL(thumbnail.Media.Url); err != nil {
		return err
	}

	return nil
}

func validateV2Text(content string) error {
	if strings.TrimSpace(content) == "" {
		return validation.NewInvalidInputError("Text components cannot be empty")
	}
	if len([]rune(content)) > maxV2TextLength {
		return validation.NewInvalidInputError("A text component is too long (maximum 4000 characters)")
	}
	return nil
}

func validateV2MediaGallery(gallery component.MediaGallery) error {
	if len(gallery.Items) == 0 {
		return validation.NewInvalidInputError("Media galleries must contain at least one image")
	}
	if len(gallery.Items) > maxV2MediaItems {
		return validation.NewInvalidInputError("Media galleries cannot contain more than 10 images")
	}
	for _, item := range gallery.Items {
		if err := validateV2MediaURL(item.Media.Url); err != nil {
			return err
		}
	}
	return nil
}

func validateV2MediaURL(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return validation.NewInvalidInputError("Image URLs must start with http:// or https://")
	}
	return nil
}
