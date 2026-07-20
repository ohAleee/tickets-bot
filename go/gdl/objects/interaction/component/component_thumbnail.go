package component

import (
	"encoding/json"
)

type Thumbnail struct {
	Media       UnfurledMediaItem `json:"media,omitempty"`
	Description *string           `json:"description,omitempty"`
	Spoiler     *bool             `json:"spoiler,omitempty"`
}

type UnfurledMediaItem struct {
	Url string `json:"url"`
	// The remaining fields are populated by Discord on responses and must be omitted when
	// sending, otherwise Discord rejects the empty proxy_url / zero dimensions.
	ProxyUrl    string `json:"proxy_url,omitempty"`
	Height      int    `json:"height,omitempty"`
	Width       int    `json:"width,omitempty"`
	ContentType string `json:"content_type,omitempty"`
}

func (i Thumbnail) Type() ComponentType {
	return ComponentThumbnail
}

func (i Thumbnail) MarshalJSON() ([]byte, error) {
	type WrappedThumbnail Thumbnail

	return json.Marshal(struct {
		Type ComponentType `json:"type"`
		WrappedThumbnail
	}{
		Type:             ComponentThumbnail,
		WrappedThumbnail: WrappedThumbnail(i),
	})
}

func BuildThumbnail(data Thumbnail) Component {
	return Component{
		Type:          ComponentThumbnail,
		ComponentData: data,
	}
}
