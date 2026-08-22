package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supernand/docubot/backend/internal/database"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
	"golang.org/x/crypto/bcrypt"
)

func newTestAuthService(t *testing.T) *service.AuthService {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	uploadDir := filepath.Join(dir, "uploads")
	db, err := database.Open(dbPath, uploadDir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return service.NewAuthService(repository.NewUserRepo(db), "test-secret")
}

func TestRegisterAndLogin_BcryptAndJWT(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, "Nanda", "Nanda@Example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "nanda@example.com" {
		t.Fatalf("email = %q, want normalized lowercase", user.Email)
	}
	if user.PasswordHash == "" || user.PasswordHash == "secret123" {
		t.Fatal("password should be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("secret123")); err != nil {
		t.Fatalf("bcrypt verify: %v", err)
	}

	res, err := svc.Login(ctx, "nanda@example.com", "secret123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if res.Token == "" {
		t.Fatal("expected JWT token")
	}

	id, err := svc.ParseToken(res.Token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if id != user.ID {
		t.Fatalf("token sub = %d, want %d", id, user.ID)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	svc := newTestAuthService(t)
	_, err := svc.Register(context.Background(), "Nanda", "a@b.com", "short")
	if !errors.Is(err, service.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, "A", "dup@example.com", "secret123"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := svc.Register(ctx, "B", "dup@example.com", "secret123")
	if !errors.Is(err, repository.ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()
	if _, err := svc.Register(ctx, "A", "user@example.com", "secret123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err := svc.Login(ctx, "user@example.com", "wrong-password")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("wrong password: %v, want ErrInvalidCredentials", err)
	}

	_, err = svc.Login(ctx, "missing@example.com", "secret123")
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Fatalf("missing email: %v, want ErrInvalidCredentials", err)
	}
}

func TestParseToken_ExpiredRejected(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()
	user, err := svc.Register(ctx, "A", "exp@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, &service.Claims{
		Email: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(user.ID, 10),
			Issuer:    "docubot",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	})
	tokenStr, err := expired.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	_, err = svc.ParseToken(tokenStr)
	if !errors.Is(err, service.ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}
