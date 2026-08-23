package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/service"
	"github.com/supernand/docubot/backend/internal/util"
)

// ChatHandler serves the public SSE chat endpoint.
type ChatHandler struct {
	chat *service.ChatService
}

// NewChatHandler constructs a ChatHandler.
func NewChatHandler(chat *service.ChatService) *ChatHandler {
	return &ChatHandler{chat: chat}
}

type chatRequest struct {
	ConversationID *string `json:"conversation_id"`
	Message        string  `json:"message"`
	Channel        string  `json:"channel"`
}

// LegacyGone handles POST /api/v1/chat — the slug-less path is retired.
func (h *ChatHandler) LegacyGone(c *gin.Context) {
	util.Error(c, http.StatusBadRequest, "GONE", "gunakan POST /api/v1/b/{slug}/chat")
}

// Chat handles POST /api/v1/b/:slug/chat (SSE).
func (h *ChatHandler) Chat(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		util.BadRequest(c, "message is required")
		return
	}

	slug := strings.TrimSpace(c.Param("slug"))
	if _, err := h.chat.LookupSlug(c.Request.Context(), slug); err != nil {
		if errors.Is(err, service.ErrBotNotFound) {
			util.Error(c, http.StatusNotFound, "NOT_FOUND", "bot tidak ditemukan")
			return
		}
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "streaming not supported")
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flusher.Flush()

	emit := &sseEmitter{w: c.Writer, flusher: flusher}
	err := h.chat.Chat(c.Request.Context(), service.ChatInput{
		Slug:           slug,
		Message:        req.Message,
		ConversationID: req.ConversationID,
		Channel:        req.Channel,
	}, emit)
	if err != nil {
		h.writeChatError(emit, err)
	}
}

func (h *ChatHandler) writeChatError(emit *sseEmitter, err error) {
	code, message := "INTERNAL", "internal server error"
	switch {
	case errors.Is(err, service.ErrValidation):
		code = "VALIDATION_ERROR"
		message = err.Error()
		if i := strings.Index(message, ": "); i >= 0 {
			message = message[i+2:]
		}
	case errors.Is(err, service.ErrBotNotFound):
		code = "NOT_FOUND"
		message = "bot tidak ditemukan"
	case errors.Is(err, service.ErrNotConfigured):
		code = "NOT_CONFIGURED"
		message = "belum ada admin; silakan register dulu"
	case errors.Is(err, service.ErrBotInactive):
		code = "BOT_INACTIVE"
		message = "Bot sedang tidak aktif"
	default:
		if strings.Contains(err.Error(), "llm:") {
			code = "LLM_ERROR"
			message = "gagal memanggil model AI"
		}
	}
	_ = emit.write("error", gin.H{"code": code, "message": message})
}

type sseEmitter struct {
	w       gin.ResponseWriter
	flusher http.Flusher
}

func (e *sseEmitter) Sources(sources []models.Source) error {
	if sources == nil {
		sources = []models.Source{}
	}
	return e.write("sources", gin.H{"sources": sources})
}

func (e *sseEmitter) Token(content string) error {
	return e.write("token", gin.H{"content": content})
}

func (e *sseEmitter) Done(conversationID string, messageID int64, totalTokens int, latencyMS int64) error {
	return e.write("done", gin.H{
		"conversation_id": conversationID,
		"message_id":      messageID,
		"total_tokens":    totalTokens,
		"latency_ms":      latencyMS,
	})
}

func (e *sseEmitter) Inactive(message string) error {
	return e.write("inactive", gin.H{"message": message})
}

func (e *sseEmitter) write(event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(e.w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	e.flusher.Flush()
	return nil
}
