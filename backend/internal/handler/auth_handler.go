package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
	"github.com/supernand/docubot/backend/internal/util"
)

// AuthHandler serves /auth/* endpoints.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "invalid JSON body")
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		h.mapAuthError(c, err)
		return
	}
	util.JSON(c, http.StatusCreated, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.BadRequest(c, "invalid JSON body")
		return
	}

	res, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.mapAuthError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, gin.H{
		"token": res.Token,
		"user": gin.H{
			"id":    res.User.ID,
			"email": res.User.Email,
			"name":  res.User.Name,
		},
	})
}

// Me handles GET /api/v1/auth/me (requires JWT middleware).
func (h *AuthHandler) Me(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		util.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}

	user, err := h.auth.Me(c.Request.Context(), userID)
	if err != nil {
		h.mapAuthError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}

func (h *AuthHandler) mapAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		msg := err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = msg[i+2:]
		}
		util.BadRequest(c, msg)
	case errors.Is(err, repository.ErrEmailTaken):
		util.Error(c, http.StatusConflict, "EMAIL_TAKEN", "email already registered")
	case errors.Is(err, service.ErrInvalidCredentials):
		util.Error(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
	case errors.Is(err, service.ErrUnauthorized):
		util.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
	default:
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
}
