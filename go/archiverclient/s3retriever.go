package archiverclient

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/TicketsBot-cloud/logarchiver/pkg/s3client"
	"github.com/minio/minio-go/v7"
)

type S3Retriever struct {
	client *s3client.S3Client
}

var _ Retriever = (*S3Retriever)(nil)

func NewS3Retriever(client *s3client.S3Client) *S3Retriever {
	return &S3Retriever{client: client}
}

func (r *S3Retriever) GetTicket(ctx context.Context, guildId uint64, ticketId int) ([]byte, error) {
	res, err := r.client.GetTicket(ctx, guildId, ticketId)
	if err != nil && errors.Is(err, s3client.ErrTicketNotFound) {
		return nil, ErrNotFound
	}

	return res, err
}

func (r *S3Retriever) StoreTicket(ctx context.Context, guildId uint64, ticketId int, data []byte) error {
	return r.client.StoreTicket(ctx, guildId, ticketId, data)
}

func (r *S3Retriever) DeleteTicket(ctx context.Context, guildId uint64, ticketId int) error {
	return r.client.DeleteTicket(ctx, guildId, ticketId)
}

// Attachment objects are addressed directly through minio: the pinned logarchiver module has no
// helper for them, only this fork's build of the service does.
func (r *S3Retriever) GetAttachment(ctx context.Context, guildId uint64, ticketId int, attachmentId uint64, filename string) ([]byte, error) {
	key := AttachmentKey(guildId, ticketId, attachmentId, filename)

	object, err := r.client.Minio().GetObject(ctx, r.client.BucketName(), key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		var resp minio.ErrorResponse
		if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return data, nil
}

func (r *S3Retriever) StoreAttachment(ctx context.Context, guildId uint64, ticketId int, attachmentId uint64, filename string, data []byte) error {
	key := AttachmentKey(guildId, ticketId, attachmentId, filename)

	_, err := r.client.Minio().PutObject(ctx, r.client.BucketName(), key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})

	return err
}
