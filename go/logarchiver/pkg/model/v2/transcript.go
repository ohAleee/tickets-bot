package v2

import (
	"time"

	"github.com/TicketsBot-cloud/gdl/objects/channel"
	"github.com/TicketsBot-cloud/gdl/objects/channel/embed"
	"github.com/TicketsBot-cloud/gdl/objects/channel/message"
	"github.com/TicketsBot-cloud/gdl/objects/interaction/component"
	"github.com/TicketsBot-cloud/logarchiver/pkg/model"
)

type Transcript struct {
	Version  model.Version `json:"version"`
	Entities Entities      `json:"entities"`
	Messages []Message     `json:"messages"`
}

type Message struct {
	Id          uint64                `json:"id"`
	AuthorId    uint64                `json:"author"`
	Content     string                `json:"content"`
	Timestamp   time.Time             `json:"timestamp"`
	Embeds      []embed.Embed         `json:"embeds,omitempty"`
	Components  []component.Component `json:"components,omitempty"`
	Attachments []channel.Attachment  `json:"attachments,omitempty"`
}

func MessageFromGdl(message message.Message) Message {
	return Message{
		Id:          message.Id,
		AuthorId:    message.Author.Id,
		Content:     message.Content,
		Timestamp:   message.Timestamp,
		Embeds:      message.Embeds,
		Components:  message.Components,
		Attachments: message.Attachments,
	}
}
