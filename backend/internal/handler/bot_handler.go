package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/service"
	"github.com/supernand/docubot/backend/internal/util"
)

// BotHandler serves public bot profile, demo pointer, and admin identity.
type BotHandler struct {
	bots *service.BotService
}

// NewBotHandler constructs a BotHandler.
func NewBotHandler(bots *service.BotService) *BotHandler {
	return &BotHandler{bots: bots}
}

// Public handles GET /api/v1/bots/:slug.
func (h *BotHandler) Public(c *gin.Context) {
	bot, err := h.bots.PublicProfile(c.Request.Context(), c.Param("slug"))
	if err != nil {
		h.mapError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, bot)
}

// Demo handles GET /api/v1/demo. Always 200; configured=false when no users exist.
func (h *BotHandler) Demo(c *gin.Context) {
	// ORDER BY user_id ASC LIMIT 1 is only for this landing pointer — never used to answer chat.
	bot, err := h.bots.Demo(c.Request.Context())
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	util.JSON(c, http.StatusOK, bot)
}

// AdminGet handles GET /api/v1/admin/bot.
func (h *BotHandler) AdminGet(c *gin.Context) {
	bot, err := h.bots.GetByUser(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		h.mapError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, bot)
}

type adminBotRequest struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	WelcomeMessage string `json:"welcome_message"`
	Active         bool   `json:"active"`
}

// AdminPut handles PUT /api/v1/admin/bot.
func (h *BotHandler) AdminPut(c *gin.Context) {
	var req adminBotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "invalid JSON body")
		return
	}
	bot, err := h.bots.Update(c.Request.Context(), middleware.UserID(c), service.UpdateBotInput{
		Slug:           req.Slug,
		Name:           req.Name,
		WelcomeMessage: req.WelcomeMessage,
		Active:         req.Active,
	})
	if err != nil {
		h.mapError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, bot)
}

func (h *BotHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		msg := err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = msg[i+2:]
		}
		util.BadRequest(c, msg)
	case errors.Is(err, service.ErrBotNotFound):
		util.Error(c, http.StatusNotFound, "NOT_FOUND", "bot tidak ditemukan")
	case errors.Is(err, service.ErrSlugTaken):
		util.Error(c, http.StatusConflict, "CONFLICT", "slug sudah dipakai")
	default:
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
}
