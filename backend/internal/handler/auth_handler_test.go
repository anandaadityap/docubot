package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/database"
	"github.com/supernand/docubot/backend/internal/handler"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
)

func setupAuthRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "test.db"), filepath.Join(dir, "uploads"))
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	secret := "test-secret"
	authSvc := service.NewAuthService(repository.NewUserRepo(db), secret)
	authH := handler.NewAuthHandler(authSvc)

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.POST("/auth/register", authH.Register)
	v1.POST("/auth/login", authH.Login)
	authed := v1.Group("")
	authed.Use(middleware.Auth(secret))
	authed.GET("/auth/me", authH.Me)
	return r
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthRegister_Created(t *testing.T) {
	r := setupAuthRouter(t)
	w := postJSON(r, "/api/v1/auth/register", map[string]string{
		"name": "Nanda", "email": "nanda@example.com", "password": "secret123",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID    int64  `json:"id"`
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data.Email != "nanda@example.com" || resp.Data.Name != "Nanda" || resp.Data.ID == 0 {
		t.Fatalf("unexpected data: %+v", resp.Data)
	}
}

func TestAuthRegister_ShortPassword(t *testing.T) {
	r := setupAuthRouter(t)
	w := postJSON(r, "/api/v1/auth/register", map[string]string{
		"email": "a@b.com", "password": "short",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestAuthRegister_EmailTaken(t *testing.T) {
	r := setupAuthRouter(t)
	body := map[string]string{"email": "dup@example.com", "password": "secret123"}
	if w := postJSON(r, "/api/v1/auth/register", body); w.Code != http.StatusCreated {
		t.Fatalf("first: %d %s", w.Code, w.Body.String())
	}
	w := postJSON(r, "/api/v1/auth/register", body)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 body=%s", w.Code, w.Body.String())
	}
}

func TestAuthLogin_SuccessAndMe(t *testing.T) {
	r := setupAuthRouter(t)
	postJSON(r, "/api/v1/auth/register", map[string]string{
		"name": "Nanda", "email": "login@example.com", "password": "secret123",
	})

	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email": "login@example.com", "password": "secret123",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", w.Code, w.Body.String())
	}
	var loginResp struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID    int64  `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &loginResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if loginResp.Data.Token == "" {
		t.Fatal("expected token")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Data.Token)
	meW := httptest.NewRecorder()
	r.ServeHTTP(meW, req)
	if meW.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", meW.Code, meW.Body.String())
	}
	var meResp struct {
		Data struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(meW.Body.Bytes(), &meResp); err != nil {
		t.Fatalf("me unmarshal: %v", err)
	}
	if meResp.Data.Email != "login@example.com" {
		t.Fatalf("email = %q", meResp.Data.Email)
	}
}

func TestAuthLogin_BadPassword(t *testing.T) {
	r := setupAuthRouter(t)
	postJSON(r, "/api/v1/auth/register", map[string]string{
		"email": "bad@example.com", "password": "secret123",
	})
	w := postJSON(r, "/api/v1/auth/login", map[string]string{
		"email": "bad@example.com", "password": "wrong-pass",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuthMe_Unauthorized(t *testing.T) {
	r := setupAuthRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
