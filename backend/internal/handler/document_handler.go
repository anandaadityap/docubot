package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/service"
	"github.com/supernand/docubot/backend/internal/util"
)

const (
	maxUploadBytes = 5 << 20 // 5 MiB
	ingestTimeout  = 3 * time.Minute
)

// DocumentHandler serves /documents/* admin endpoints.
type DocumentHandler struct {
	docs       *service.DocumentService
	processing sync.Map // documentID -> struct{}
}

// NewDocumentHandler constructs a DocumentHandler.
func NewDocumentHandler(docs *service.DocumentService) *DocumentHandler {
	return &DocumentHandler{docs: docs}
}

// Upload handles POST /api/v1/documents (multipart field "file").
func (h *DocumentHandler) Upload(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		util.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+1024)
	file, err := c.FormFile("file")
	if err != nil {
		if isBodyTooLarge(err) {
			util.BadRequest(c, "file exceeds 5 MB")
			return
		}
		util.BadRequest(c, "file is required (multipart field \"file\")")
		return
	}
	if file.Size > maxUploadBytes {
		util.BadRequest(c, "file exceeds 5 MB")
		return
	}

	src, err := file.Open()
	if err != nil {
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}
	defer src.Close()

	doc, err := h.docs.Upload(c.Request.Context(), userID, file.Filename, file.Size, src)
	if err != nil {
		h.mapError(c, err)
		return
	}

	h.startProcess(doc.ID)

	util.JSON(c, http.StatusCreated, gin.H{
		"id":       doc.ID,
		"filename": doc.Filename,
		"status":   doc.Status,
	})
}

// List handles GET /api/v1/documents.
func (h *DocumentHandler) List(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		util.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}
	list, err := h.docs.List(c.Request.Context(), userID)
	if err != nil {
		h.mapError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, list)
}

// Get handles GET /api/v1/documents/:id.
func (h *DocumentHandler) Get(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		util.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	detail, err := h.docs.Get(c.Request.Context(), userID, id)
	if err != nil {
		h.mapError(c, err)
		return
	}
	util.JSON(c, http.StatusOK, detail)
}

// Delete handles DELETE /api/v1/documents/:id.
func (h *DocumentHandler) Delete(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		util.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.docs.Delete(c.Request.Context(), userID, id); err != nil {
		h.mapError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Reprocess handles POST /api/v1/documents/:id/reprocess.
func (h *DocumentHandler) Reprocess(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == 0 {
		util.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	doc, err := h.docs.Reprocess(c.Request.Context(), userID, id)
	if err != nil {
		h.mapError(c, err)
		return
	}
	h.startProcess(doc.ID)
	util.JSON(c, http.StatusOK, gin.H{"status": models.DocumentStatusProcessing})
}

func (h *DocumentHandler) startProcess(documentID int64) {
	if _, loaded := h.processing.LoadOrStore(documentID, struct{}{}); loaded {
		return
	}
	go func() {
		defer h.processing.Delete(documentID)
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("document process panic id=%d: %v", documentID, rec)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
		defer cancel()
		if err := h.docs.Process(ctx, documentID); err != nil {
			log.Printf("document process id=%d: %v", documentID, err)
		}
	}()
}

func parseIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		util.BadRequest(c, "invalid document id")
		return 0, false
	}
	return id, true
}

func (h *DocumentHandler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		msg := err.Error()
		if i := strings.Index(msg, ": "); i >= 0 {
			msg = msg[i+2:]
		}
		util.BadRequest(c, msg)
	case errors.Is(err, service.ErrNotFound):
		util.Error(c, http.StatusNotFound, "NOT_FOUND", "document not found")
	case errors.Is(err, service.ErrBusy):
		util.Error(c, http.StatusConflict, "BUSY", "dokumen sedang diproses")
	default:
		util.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
}

func isBodyTooLarge(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "request body too large") ||
		strings.Contains(msg, "http: request body too large")
}
