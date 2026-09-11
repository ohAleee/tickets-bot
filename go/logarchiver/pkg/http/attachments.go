package http

import (
	"errors"
	"strconv"

	"github.com/TicketsBot-cloud/logarchiver/pkg/s3client"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Attachment blobs are stored already encrypted by the archiver client, exactly like transcripts.
func parseAttachmentParams(ctx *gin.Context) (guildId uint64, ticketId int, attachmentId uint64, filename string, ok bool) {
	guildId, err := strconv.ParseUint(ctx.Query("guild"), 10, 64)
	if err != nil {
		ctx.JSON(400, gin.H{"message": "missing guild ID"})
		return
	}

	ticketId, err = strconv.Atoi(ctx.Query("id"))
	if err != nil {
		ctx.JSON(400, gin.H{"message": "missing ticket ID"})
		return
	}

	attachmentId, err = strconv.ParseUint(ctx.Query("attachment"), 10, 64)
	if err != nil {
		ctx.JSON(400, gin.H{"message": "missing attachment ID"})
		return
	}

	filename = ctx.Query("filename")
	if filename == "" {
		ctx.JSON(400, gin.H{"message": "missing filename"})
		return
	}

	return guildId, ticketId, attachmentId, filename, true
}

func (s *Server) attachmentGetHandler(ctx *gin.Context) {
	guildId, ticketId, attachmentId, filename, ok := parseAttachmentParams(ctx)
	if !ok {
		return
	}

	client, found, err := s.getClientForObject(ctx, guildId, ticketId)
	if err != nil {
		s.Logger.Error("Failed to get client for object", zap.Error(err))
		ctx.JSON(500, gin.H{"message": err.Error()})
		return
	}

	if !found {
		ctx.JSON(404, gin.H{"message": "ticket not found"})
		return
	}

	data, err := client.GetAttachment(ctx, guildId, ticketId, attachmentId, filename)
	if err != nil {
		if errors.Is(err, s3client.ErrTicketNotFound) {
			ctx.JSON(404, gin.H{"message": "attachment not found"})
			return
		}

		s.Logger.Error("Failed to get attachment", zap.Error(err), zap.Uint64("guild", guildId), zap.Int("id", ticketId))
		ctx.JSON(500, gin.H{"message": err.Error()})
		return
	}

	ctx.Data(200, "application/octet-stream", data)
}

func (s *Server) attachmentUploadHandler(ctx *gin.Context) {
	guildId, ticketId, attachmentId, filename, ok := parseAttachmentParams(ctx)
	if !ok {
		return
	}

	body, err := ctx.GetRawData()
	if err != nil {
		ctx.JSON(400, gin.H{"message": err.Error()})
		return
	}

	// The transcript itself is uploaded straight after, so both land in the same active bucket.
	client, _, err := s.getActiveClient(ctx)
	if err != nil {
		s.Logger.Error("Failed to get active client", zap.Error(err))
		ctx.JSON(500, gin.H{"message": err.Error()})
		return
	}

	if err := client.StoreAttachment(ctx, guildId, ticketId, attachmentId, filename, body); err != nil {
		s.Logger.Error("Failed to store attachment", zap.Error(err), zap.Uint64("guild", guildId), zap.Int("id", ticketId))
		ctx.JSON(500, gin.H{"message": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{})
}
