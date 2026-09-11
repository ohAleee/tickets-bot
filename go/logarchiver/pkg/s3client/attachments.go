package s3client

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// Ticket attachments live under the ticket's own key prefix, so a guild purge (which lists
// "{guild}/") and a ticket delete pick them up along with the transcript.
func attachmentKey(guildId uint64, ticketId int, attachmentId uint64, filename string) string {
	return fmt.Sprintf("%d/%d/attachments/%d/%s", guildId, ticketId, attachmentId, filename)
}

func attachmentPrefix(guildId uint64, ticketId int) string {
	return fmt.Sprintf("%d/%d/attachments/", guildId, ticketId)
}

func (c *S3Client) GetAttachment(ctx context.Context, guildId uint64, ticketId int, attachmentId uint64, filename string) ([]byte, error) {
	object, err := c.client.GetObject(ctx, c.bucketName, attachmentKey(guildId, ticketId, attachmentId, filename), minio.GetObjectOptions{})
	if err != nil {
		if isNotFoundErr(err) {
			return nil, ErrTicketNotFound
		}

		return nil, err
	}

	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		// Minio defers the request until the first read, so a missing object surfaces here.
		if isNotFoundErr(err) {
			return nil, ErrTicketNotFound
		}

		return nil, err
	}

	return data, nil
}

func (c *S3Client) StoreAttachment(ctx context.Context, guildId uint64, ticketId int, attachmentId uint64, filename string, data []byte) error {
	key := attachmentKey(guildId, ticketId, attachmentId, filename)

	_, err := c.client.PutObject(ctx, c.bucketName, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})

	return err
}

func (c *S3Client) DeleteAttachments(ctx context.Context, guildId uint64, ticketId int) error {
	objects := c.client.ListObjects(ctx, c.bucketName, minio.ListObjectsOptions{
		Prefix:    attachmentPrefix(guildId, ticketId),
		Recursive: true,
	})

	var firstErr error
	for res := range c.client.RemoveObjects(ctx, c.bucketName, objects, minio.RemoveObjectsOptions{}) {
		if firstErr == nil && res.Err != nil {
			firstErr = res.Err
		}
	}

	return firstErr
}
