package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Health responds with a simple OK payload for Docker healthchecks.
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
