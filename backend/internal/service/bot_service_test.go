package service_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/supernand/docubot/backend/internal/database"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
)

func TestBotService_UpdateSlugAndDualWrite(t *testing.T) {
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "t.db"), filepath.Join(dir, "up"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	users := repository.NewUserRepo(db)
	bots := repository.NewBotRepo(db)
	settings := repository.NewSettingsRepo(db)
	auth := service.NewAuthService(users, "secret")
	u, err := auth.Register(context.Background(), "Nanda", "n@example.com", "secret123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	svc := service.NewBotService(bots, settings, repository.NewDocumentRepo(db))
	got, err := svc.Update(context.Background(), u.ID, service.UpdateBotInput{
		Slug: "toko-kita", Name: "Asisten Toko Kita", WelcomeMessage: "Halo!", Active: true,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if got.Slug != "toko-kita" || got.PublicPath != "/b/toko-kita" {
		t.Fatalf("admin bot %+v", got)
	}

	pub, err := svc.PublicProfile(context.Background(), "toko-kita")
	if err != nil {
		t.Fatalf("public: %v", err)
	}
	if pub.BotName != "Asisten Toko Kita" {
		t.Fatalf("name = %q", pub.BotName)
	}

	cfg, err := settings.GetByUserID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if cfg.BotName != "Asisten Toko Kita" {
		t.Fatalf("settings not mirrored: %q", cfg.BotName)
	}

	_, err = svc.Update(context.Background(), u.ID, service.UpdateBotInput{
		Slug: "admin", Name: "X", WelcomeMessage: "Halo!", Active: true,
	})
	if err == nil {
		t.Fatal("reserved slug should fail")
	}
}
