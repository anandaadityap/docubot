package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/service"
	"github.com/supernand/docubot/backend/internal/util"
)

// ConversationHandler serves admin conversation endpoints.
type ConversationHandler struct {
	convos *service.ConversationService
}

// NewConversationHandler constructs a ConversationHandler.
func NewConversationHandler(convos *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{convos: convos}
}

// List handles GET /api/v1/conversations.
func (h *ConversationHandler) List(c *gin.Context) {
	userID := middleware.UserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	res, err := h.convos.List(c.Request.Context(), userID, page, limit)
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	util.JSON(c, http.StatusOK, res)
}

// Get handles GET /api/v1/conversations/:id.
func (h *ConversationHandler) Get(c *gin.Context) {
	userID := middleware.UserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		util.BadRequest(c, "invalid conversation id")
		return
	}
	detail, err := h.convos.Get(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			util.Error(c, http.StatusNotFound, "NOT_FOUND", "conversation not found")
			return
		}
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	util.JSON(c, http.StatusOK, detail)
}
