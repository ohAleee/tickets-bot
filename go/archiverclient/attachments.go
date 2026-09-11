package archiverclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/TicketsBot-cloud/gdl/objects/channel/message"
	"github.com/TicketsBot/common/encryption"
)

// Discord CDN links are signed and expire within hours, so a transcript that just points at them
// loses every image once the ticket is closed. Store() mirrors the files into the archive bucket
// and rewrites the URLs to MediaPathPrefix paths, which the dashboard signs and serves.
const MediaPathPrefix = "/media/"

// ARCHIVER_MEDIA_MAX_BYTES caps the size of a single mirrored attachment. 0 disables mirroring.
var maxAttachmentBytes = parseMaxAttachmentBytes(os.Getenv("ARCHIVER_MEDIA_MAX_BYTES"))

const defaultMaxAttachmentBytes = 25 * 1024 * 1024

var attachmentDownloader = &http.Client{Timeout: 60 * time.Second}

func parseMaxAttachmentBytes(raw string) int64 {
	if raw == "" {
		return defaultMaxAttachmentBytes
	}

	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed < 0 {
		return defaultMaxAttachmentBytes
	}

	return parsed
}

// AttachmentKey must match the key logarchiver writes (pkg/s3client/attachments.go).
func AttachmentKey(guildId uint64, ticketId int, attachmentId uint64, filename string) string {
	return fmt.Sprintf("%d/%d/attachments/%d/%s", guildId, ticketId, attachmentId, filename)
}

func MediaPath(guildId uint64, ticketId int, attachmentId uint64, filename string) string {
	return fmt.Sprintf("%s%d/%d/%d/%s", MediaPathPrefix, guildId, ticketId, attachmentId, url.PathEscape(filename))
}

// mirrorAttachments copies every attachment small enough into the archive bucket and points the
// messages at the mirrored copy. Best effort: an attachment that cannot be downloaded or stored
// keeps its original CDN URL rather than failing the whole archive.
// ponytail: downloads are sequential, parallelise if closing media-heavy tickets gets slow.
func (c *ArchiverClient) mirrorAttachments(ctx context.Context, guildId uint64, ticketId int, messages []message.Message) {
	if maxAttachmentBytes == 0 {
		return
	}

	for _, msg := range messages {
		for i, attachment := range msg.Attachments {
			if int64(attachment.Size) > maxAttachmentBytes {
				continue
			}

			data, err := c.downloadAttachment(ctx, attachment.Url)
			if err != nil {
				continue
			}

			encrypted, err := encryption.Encrypt(c.key, data)
			if err != nil {
				continue
			}

			if err := c.retriever.StoreAttachment(ctx, guildId, ticketId, attachment.Id, attachment.Filename, encrypted); err != nil {
				continue
			}

			path := MediaPath(guildId, ticketId, attachment.Id, attachment.Filename)
			msg.Attachments[i].Url = path
			msg.Attachments[i].ProxyUrl = path
		}
	}
}

func (c *ArchiverClient) downloadAttachment(ctx context.Context, source string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
	if err != nil {
		return nil, err
	}

	res, err := attachmentDownloader.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("attachment download returned %d", res.StatusCode)
	}

	// +1 byte: a body larger than the cap (Size lied, or a redirect to something else) is rejected
	// rather than silently truncated.
	data, err := io.ReadAll(io.LimitReader(res.Body, maxAttachmentBytes+1))
	if err != nil {
		return nil, err
	}

	if int64(len(data)) > maxAttachmentBytes {
		return nil, fmt.Errorf("attachment exceeds %d bytes", maxAttachmentBytes)
	}

	return data, nil
}

func (c *ArchiverClient) GetAttachment(ctx context.Context, guildId uint64, ticketId int, attachmentId uint64, filename string) ([]byte, error) {
	data, err := c.retriever.GetAttachment(ctx, guildId, ticketId, attachmentId, filename)
	if err != nil {
		return nil, err
	}

	return encryption.Decrypt(c.key, data)
}
