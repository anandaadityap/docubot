package database_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/supernand/docubot/backend/internal/database"
	_ "modernc.org/sqlite"
)

func TestMigrate_BackfillBots(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE settings (
			user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			bot_name TEXT NOT NULL DEFAULT 'DocuBot',
			welcome_message TEXT NOT NULL DEFAULT 'Halo!',
			bot_active INTEGER NOT NULL DEFAULT 1,
			temperature REAL NOT NULL DEFAULT 0.3,
			max_tokens INTEGER NOT NULL DEFAULT 500,
			top_k INTEGER NOT NULL DEFAULT 5,
			min_score REAL NOT NULL DEFAULT 0.3,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO users (email, password_hash, name) VALUES ('old@example.com', 'x', 'Toko Lama');
		INSERT INTO settings (user_id, bot_name, welcome_message, bot_active) VALUES (1, 'Asisten Toko', 'Halo toko', 1);
	`)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bots WHERE user_id = 1`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("bots = %d, want 1", n)
	}
	var slug, name string
	if err := db.QueryRow(`SELECT slug, name FROM bots WHERE user_id = 1`).Scan(&slug, &name); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if slug != "asisten-toko" {
		t.Fatalf("slug = %q", slug)
	}
	if name != "Asisten Toko" {
		t.Fatalf("name = %q", name)
	}
}
