package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/supernand/docubot/backend/internal/database"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
	"github.com/supernand/docubot/backend/internal/util"
)

func TestUserCreate_InsertsBot(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "t.db"), filepath.Join(dir, "up"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	users := repository.NewUserRepo(db)
	auth := service.NewAuthService(users, "secret")
	u, err := auth.Register(context.Background(), "Toko Kita", "t@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	bots := repository.NewBotRepo(db)
	bot, err := bots.GetByUserID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("get bot: %v", err)
	}
	if bot.Slug != "toko-kita" {
		t.Fatalf("slug = %q", bot.Slug)
	}
	if bot.Name != "Toko Kita" {
		t.Fatalf("name = %q", bot.Name)
	}
	if !util.ValidSlug(bot.Slug) {
		t.Fatalf("invalid slug %q", bot.Slug)
	}
}

func TestBotRepo_GetBySlugMissing(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "t.db"), filepath.Join(dir, "up"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	_, err = repository.NewBotRepo(db).GetBySlug(context.Background(), "tidak-ada")
	if err != repository.ErrNotFound {
		t.Fatalf("err = %v", err)
	}
}
