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

// SettingsHandler serves public bot profile and admin settings.
type SettingsHandler struct {
	settings *service.SettingsService
}

// NewSettingsHandler constructs a SettingsHandler.
func NewSettingsHandler(settings *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{settings: settings}
}

// PublicBot handles GET /api/v1/bot.
func (h *SettingsHandler) PublicBot(c *gin.Context) {
	bot, err := h.settings.GetPublic(c.Request.Context())
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	util.JSON(c, http.StatusOK, bot)
}

// Get handles GET /api/v1/settings.
func (h *SettingsHandler) Get(c *gin.Context) {
	userID := middleware.UserID(c)
	cfg, err := h.settings.Get(c.Request.Context(), userID)
	if err != nil {
		h.mapError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, cfg)
}

type settingsRequest struct {
	BotName        string  `json:"bot_name"`
	WelcomeMessage string  `json:"welcome_message"`
	BotActive      bool    `json:"bot_active"`
	Temperature    float64 `json:"temperature"`
	MaxTokens      int     `json:"max_tokens"`
	TopK           int     `json:"top_k"`
	MinScore       float64 `json:"min_score"`
}

// Put handles PUT /api/v1/settings.
func (h *SettingsHandler) Put(c *gin.Context) {
	userID := middleware.UserID(c)
	var req settingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "invalid JSON body")
		return
	}
	cfg, err := h.settings.Update(c.Request.Context(), userID, service.UpdateInput{
		BotName:        req.BotName,
		WelcomeMessage: req.WelcomeMessage,
		BotActive:      req.BotActive,
		Temperature:    req.Temperature,
		MaxTokens:      req.MaxTokens,
		TopK:           req.TopK,
		MinScore:       req.MinScore,
	})
	if err != nil {
		h.mapError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, cfg)
}

func (h *SettingsHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		msg := err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = msg[i+2:]
		}
		util.BadRequest(c, msg)
	case errors.Is(err, service.ErrNotFound):
		util.Error(c, http.StatusNotFound, "NOT_FOUND", "settings not found")
	default:
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
}
