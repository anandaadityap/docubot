package handler_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/ai"
	"github.com/supernand/docubot/backend/internal/database"
	"github.com/supernand/docubot/backend/internal/handler"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
)

func setupDocRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	uploadDir := filepath.Join(dir, "uploads")
	db, err := database.Open(filepath.Join(dir, "test.db"), uploadDir)
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	secret := "test-secret"
	authSvc := service.NewAuthService(repository.NewUserRepo(db), secret)
	authH := handler.NewAuthHandler(authSvc)

	docSvc := service.NewDocumentService(
		repository.NewDocumentRepo(db),
		repository.NewChunkRepo(db),
		ai.NewStubEmbedder(),
		uploadDir,
	)
	docH := handler.NewDocumentHandler(docSvc)

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", authH.Register)
	v1.POST("/auth/login", authH.Login)
	authed := v1.Group("")
	authed.Use(middleware.Auth(secret))
	authed.GET("/auth/me", authH.Me)
	authed.POST("/documents", docH.Upload)
	authed.GET("/documents", docH.List)
	authed.GET("/documents/:id", docH.Get)
	authed.DELETE("/documents/:id", docH.Delete)
	authed.POST("/documents/:id/reprocess", docH.Reprocess)
	return r
}

func loginToken(t *testing.T, r *gin.Engine, email string) string {
	t.Helper()
	postJSON(r, "/api/v1/auth/register", map[string]string{
		"name": "Nanda", "email": email, "password": "secret123",
	})
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email": email, "password": "secret123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return resp.Data.Token
}

func uploadMultipart(r *gin.Engine, token, fieldName, filename string, content []byte) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(fieldName, filename)
	if err != nil {
		panic(err)
	}
	_, _ = fw.Write(content)
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func authGet(r *gin.Engine, token, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func authDelete(r *gin.Engine, token, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func authPost(r *gin.Engine, token, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDocuments_UploadListGetDelete(t *testing.T) {
	r := setupDocRouter(t)
	token := loginToken(t, r, "docs@example.com")

	sample := readSampleManual(t)

	w := uploadMultipart(r, token, "file", "manual-pengguna.md", sample)
	if w.Code != http.StatusCreated {
		t.Fatalf("upload status = %d body=%s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID       int64  `json:"id"`
			Filename string `json:"filename"`
			Status   string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if created.Data.ID == 0 || created.Data.Status != models.DocumentStatusPending {
		t.Fatalf("created = %+v", created.Data)
	}

	idPath := "/api/v1/documents/" + strconv.FormatInt(created.Data.ID, 10)
	var status string
	var chunkCount int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		gw := authGet(r, token, idPath)
		if gw.Code != http.StatusOK {
			t.Fatalf("get status = %d body=%s", gw.Code, gw.Body.String())
		}
		var detail struct {
			Data struct {
				Status     string `json:"status"`
				ChunkCount int    `json:"chunk_count"`
				ErrorMsg   string `json:"error_msg"`
				Chunks     []struct {
					Position int    `json:"position"`
					Content  string `json:"content"`
				} `json:"chunks"`
			} `json:"data"`
		}
		if err := json.Unmarshal(gw.Body.Bytes(), &detail); err != nil {
			t.Fatalf("get unmarshal: %v", err)
		}
		status = detail.Data.Status
		chunkCount = detail.Data.ChunkCount
		if status == models.DocumentStatusReady || status == models.DocumentStatusFailed {
			if status == models.DocumentStatusFailed {
				t.Fatalf("failed: %s", detail.Data.ErrorMsg)
			}
			if chunkCount < 1 || len(detail.Data.Chunks) < 1 {
				t.Fatalf("chunks empty: count=%d", chunkCount)
			}
			if bytes.Contains(gw.Body.Bytes(), []byte(`"embedding"`)) {
				t.Fatal("embedding key present in get response")
			}
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if status != models.DocumentStatusReady {
		t.Fatalf("status = %s after poll", status)
	}

	lw := authGet(r, token, "/api/v1/documents")
	if lw.Code != http.StatusOK {
		t.Fatalf("list = %d %s", lw.Code, lw.Body.String())
	}
	var list struct {
		Data []struct {
			ID       int64  `json:"id"`
			Filename string `json:"filename"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lw.Body.Bytes(), &list); err != nil {
		t.Fatalf("list unmarshal: %v", err)
	}
	if len(list.Data) < 1 {
		t.Fatal("expected list items")
	}

	dw := authDelete(r, token, idPath)
	if dw.Code != http.StatusNoContent {
		t.Fatalf("delete = %d %s", dw.Code, dw.Body.String())
	}
	gw := authGet(r, token, idPath)
	if gw.Code != http.StatusNotFound {
		t.Fatalf("get after delete = %d", gw.Code)
	}
}

func TestDocuments_Unauthorized(t *testing.T) {
	r := setupDocRouter(t)
	w := uploadMultipart(r, "", "file", "a.md", []byte("hello"))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestDocuments_InvalidType(t *testing.T) {
	r := setupDocRouter(t)
	token := loginToken(t, r, "badtype@example.com")
	w := uploadMultipart(r, token, "file", "photo.png", []byte("not-a-png"))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}

func TestDocuments_Reprocess(t *testing.T) {
	r := setupDocRouter(t)
	token := loginToken(t, r, "repro@example.com")
	w := uploadMultipart(r, token, "file", "faq.txt", []byte("Bagaimana cara reset password? Lihat Settings."))
	if w.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	waitReady(t, r, token, created.Data.ID)

	rw := authPost(r, token, "/api/v1/documents/"+strconv.FormatInt(created.Data.ID, 10)+"/reprocess")
	if rw.Code != http.StatusOK {
		t.Fatalf("reprocess = %d %s", rw.Code, rw.Body.String())
	}
	var body struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rw.Body.Bytes(), &body)
	if body.Data.Status != models.DocumentStatusProcessing {
		t.Fatalf("status = %s", body.Data.Status)
	}
	waitReady(t, r, token, created.Data.ID)
}

func waitReady(t *testing.T, r *gin.Engine, token string, id int64) {
	t.Helper()
	path := "/api/v1/documents/" + strconv.FormatInt(id, 10)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		gw := authGet(r, token, path)
		if gw.Code != http.StatusOK {
			t.Fatalf("get: %d %s", gw.Code, gw.Body.String())
		}
		var detail struct {
			Data struct {
				Status   string `json:"status"`
				ErrorMsg string `json:"error_msg"`
			} `json:"data"`
		}
		_ = json.Unmarshal(gw.Body.Bytes(), &detail)
		switch detail.Data.Status {
		case models.DocumentStatusReady:
			return
		case models.DocumentStatusFailed:
			t.Fatalf("failed: %s", detail.Data.ErrorMsg)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timeout waiting for ready")
}
