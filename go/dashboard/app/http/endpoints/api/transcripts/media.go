package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TicketsBot-cloud/archiverclient"
	"github.com/TicketsBot-cloud/dashboard/config"
	"github.com/TicketsBot-cloud/dashboard/utils"
	v2 "github.com/TicketsBot-cloud/logarchiver/pkg/model/v2"
	"github.com/gin-gonic/gin"
)

// Mirrored attachments are stored with a MediaPathPrefix path instead of the (expiring) Discord
// CDN URL. The browser loads them with a plain <img> tag, which cannot send the dashboard's
// Authorization header, so the transcript endpoints hand out short-lived signed URLs instead.
const mediaLinkTtl = time.Hour

func mediaSignature(guildId uint64, ticketId int, attachmentId uint64, filename string, expiry int64) string {
	mac := hmac.New(sha256.New, []byte(config.Conf.Server.Secret))
	fmt.Fprintf(mac, "%d:%d:%d:%s:%d", guildId, ticketId, attachmentId, filename, expiry)
	return hex.EncodeToString(mac.Sum(nil))
}

func mediaBaseUrl(ctx *gin.Context) string {
	if config.Conf.Server.MediaBaseUrl != "" {
		return strings.TrimSuffix(config.Conf.Server.MediaBaseUrl, "/")
	}

	scheme := "http"
	if proto := ctx.GetHeader("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if ctx.Request.TLS != nil {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, ctx.Request.Host)
}

func signedMediaUrl(ctx *gin.Context, guildId uint64, ticketId int, attachmentId uint64, filename string) string {
	expiry := time.Now().Add(mediaLinkTtl).Unix()

	return fmt.Sprintf(
		"%s%s?exp=%d&sig=%s",
		mediaBaseUrl(ctx),
		archiverclient.MediaPath(guildId, ticketId, attachmentId, filename),
		expiry,
		mediaSignature(guildId, ticketId, attachmentId, filename, expiry),
	)
}

// SignAttachmentUrls rewrites mirrored attachments in place. Attachments that were never mirrored
// (too large, or archived before mirroring existed) keep their original CDN URL.
func SignAttachmentUrls(ctx *gin.Context, guildId uint64, ticketId int, messages []v2.Message) {
	for _, msg := range messages {
		for i, attachment := range msg.Attachments {
			if !strings.HasPrefix(attachment.Url, archiverclient.MediaPathPrefix) {
				continue
			}

			signed := signedMediaUrl(ctx, guildId, ticketId, attachment.Id, attachment.Filename)
			msg.Attachments[i].Url = signed
			msg.Attachments[i].ProxyUrl = signed
		}
	}
}

// GetAttachmentHandler serves a mirrored attachment. It is registered outside the /api group: the
// signature in the query string is the authentication, since an <img> sends no headers.
func GetAttachmentHandler(ctx *gin.Context) {
	guildId, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(400, utils.ErrorStr("Invalid guild ID"))
		return
	}

	ticketId, err := strconv.Atoi(ctx.Param("ticketId"))
	if err != nil {
		ctx.JSON(400, utils.ErrorStr("Invalid ticket ID"))
		return
	}

	attachmentId, err := strconv.ParseUint(ctx.Param("attachmentId"), 10, 64)
	if err != nil {
		ctx.JSON(400, utils.ErrorStr("Invalid attachment ID"))
		return
	}

	// gin matches on the decoded path, so the filename arrives unescaped.
	filename := strings.TrimPrefix(ctx.Param("filename"), "/")

	expiry, err := strconv.ParseInt(ctx.Query("exp"), 10, 64)
	if err != nil {
		ctx.JSON(400, utils.ErrorStr("Invalid expiry"))
		return
	}

	if time.Now().Unix() > expiry {
		ctx.JSON(403, utils.ErrorStr("Link has expired, reload the transcript"))
		return
	}

	expected := mediaSignature(guildId, ticketId, attachmentId, filename, expiry)
	if !hmac.Equal([]byte(expected), []byte(ctx.Query("sig"))) {
		ctx.JSON(403, utils.ErrorStr("Invalid signature"))
		return
	}

	data, err := utils.ArchiverClient.GetAttachment(ctx, guildId, ticketId, attachmentId, filename)
	if err != nil {
		if errors.Is(err, archiverclient.ErrNotFound) {
			ctx.JSON(404, utils.ErrorStr("Attachment not found"))
		} else {
			ctx.JSON(500, utils.ErrorStr("Failed to fetch attachment. Please try again."))
		}

		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filename}))
	ctx.Header("Cache-Control", "private, max-age=3600")
	ctx.Data(200, contentType, data)
}
