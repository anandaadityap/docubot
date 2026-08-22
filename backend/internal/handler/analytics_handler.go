package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/service"
	"github.com/supernand/docubot/backend/internal/util"
)

// AnalyticsHandler serves admin analytics endpoints.
type AnalyticsHandler struct {
	analytics *service.AnalyticsService
}

// NewAnalyticsHandler constructs an AnalyticsHandler.
func NewAnalyticsHandler(analytics *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analytics: analytics}
}

// Overview handles GET /api/v1/analytics/overview.
func (h *AnalyticsHandler) Overview(c *gin.Context) {
	userID := middleware.UserID(c)
	ov, err := h.analytics.OverviewLast14Days(c.Request.Context(), userID)
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	util.JSON(c, http.StatusOK, ov)
}

// TopQuestions handles GET /api/v1/analytics/top-questions.
func (h *AnalyticsHandler) TopQuestions(c *gin.Context) {
	userID := middleware.UserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	rows, err := h.analytics.TopQuestions(c.Request.Context(), userID, limit)
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	util.JSON(c, http.StatusOK, rows)
}
