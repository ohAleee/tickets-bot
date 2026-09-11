package archiverclient

import (
	"context"
)

type Retriever interface {
	GetTicket(ctx context.Context, guildId uint64, ticketId int) ([]byte, error)
	StoreTicket(ctx context.Context, guildId uint64, ticketId int, data []byte) error
	DeleteTicket(ctx context.Context, guildId uint64, ticketId int) error
	GetAttachment(ctx context.Context, guildId uint64, ticketId int, attachmentId uint64, filename string) ([]byte, error)
	StoreAttachment(ctx context.Context, guildId uint64, ticketId int, attachmentId uint64, filename string, data []byte) error
}
