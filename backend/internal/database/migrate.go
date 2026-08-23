package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/supernand/docubot/backend/internal/util"
)

// Migrate applies the DocuBot schema (BRD §8.2) idempotently, then additive columns.
func Migrate(db *sql.DB) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name          TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS documents (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename    TEXT NOT NULL,
    file_type   TEXT NOT NULL,
    size_bytes  INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'pending',
    error_msg   TEXT,
    chunk_count INTEGER NOT NULL DEFAULT 0,
    embed_model TEXT NOT NULL DEFAULT '',
    embed_dim   INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chunks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id  INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    position     INTEGER NOT NULL,
    content      TEXT NOT NULL,
    token_count  INTEGER NOT NULL DEFAULT 0,
    embedding    TEXT NOT NULL,
    embed_model  TEXT NOT NULL DEFAULT '',
    embed_dim    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_chunks_document ON chunks(document_id);

CREATE TABLE IF NOT EXISTS conversations (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_id     TEXT NOT NULL DEFAULT '',
    title         TEXT NOT NULL DEFAULT 'Chat baru',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    content         TEXT NOT NULL,
    sources         TEXT,
    latency_ms      INTEGER,
    token_usage     INTEGER,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);

CREATE TABLE IF NOT EXISTS settings (
    user_id         INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bot_name        TEXT NOT NULL DEFAULT 'DocuBot',
    welcome_message TEXT NOT NULL DEFAULT 'Halo! Ada yang bisa saya bantu?',
    bot_active      INTEGER NOT NULL DEFAULT 1,
    temperature     REAL NOT NULL DEFAULT 0.3,
    max_tokens      INTEGER NOT NULL DEFAULT 500,
    top_k           INTEGER NOT NULL DEFAULT 5,
    min_score       REAL NOT NULL DEFAULT 0.3,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bots (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    slug             TEXT NOT NULL UNIQUE,
    name             TEXT NOT NULL DEFAULT 'DocuBot',
    welcome_message  TEXT NOT NULL DEFAULT 'Halo! Ada yang bisa saya bantu?',
    active           INTEGER NOT NULL DEFAULT 1,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_bots_slug ON bots(slug);
CREATE UNIQUE INDEX IF NOT EXISTS idx_bots_user ON bots(user_id);
`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("exec schema ddl: %w", err)
	}

	if err := addColumn(db, "documents", "embed_model", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumn(db, "documents", "embed_dim", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumn(db, "chunks", "embed_model", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := addColumn(db, "chunks", "embed_dim", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := addColumn(db, "conversations", "public_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}

	if _, err := db.Exec(`
		UPDATE conversations
		SET public_id = lower(hex(randomblob(16)))
		WHERE public_id IS NULL OR public_id = ''`); err != nil {
		return fmt.Errorf("backfill conversation public_id: %w", err)
	}

	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_public_id ON conversations(public_id)`); err != nil {
		return fmt.Errorf("index conversations public_id: %w", err)
	}

	if err := addColumn(db, "conversations", "channel", "TEXT NOT NULL DEFAULT 'public'"); err != nil {
		return err
	}

	if err := backfillBots(db); err != nil {
		return err
	}

	// Interrupted ingest after a crash/restart would otherwise stay "processing" forever.
	if _, err := db.Exec(`
		UPDATE documents
		SET status = 'failed',
		    error_msg = 'ingest interrupted; proses ulang dokumen ini',
		    updated_at = CURRENT_TIMESTAMP
		WHERE status = 'processing'`); err != nil {
		return fmt.Errorf("reset interrupted ingest: %w", err)
	}

	return nil
}

func addColumn(db *sql.DB, table, column, decl string) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, decl))
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate column") {
		return nil
	}
	return fmt.Errorf("add column %s.%s: %w", table, column, err)
}

func backfillBots(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT u.id, u.name,
		       COALESCE(s.bot_name, ''),
		       COALESCE(s.welcome_message, ''),
		       COALESCE(s.bot_active, 1)
		FROM users u
		LEFT JOIN settings s ON s.user_id = u.id
		WHERE NOT EXISTS (SELECT 1 FROM bots b WHERE b.user_id = u.id)
		ORDER BY u.id ASC`)
	if err != nil {
		return fmt.Errorf("select users without bots: %w", err)
	}
	defer rows.Close()

	type pending struct {
		userID  int64
		name    string
		welcome string
		active  int
		seed    string
	}
	var list []pending
	for rows.Next() {
		var p pending
		var userName, botName string
		if err := rows.Scan(&p.userID, &userName, &botName, &p.welcome, &p.active); err != nil {
			return fmt.Errorf("scan user for bot backfill: %w", err)
		}
		p.name = strings.TrimSpace(botName)
		if p.name == "" {
			p.name = strings.TrimSpace(userName)
		}
		if p.name == "" {
			p.name = "DocuBot"
		}
		p.seed = strings.TrimSpace(botName)
		if p.seed == "" {
			p.seed = strings.TrimSpace(userName)
		}
		if p.seed == "" {
			p.seed = "bot"
		}
		if strings.TrimSpace(p.welcome) == "" {
			p.welcome = "Halo! Ada yang bisa saya bantu?"
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("users without bots rows: %w", err)
	}

	taken := func(slug string) bool {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM bots WHERE slug = ?`, slug).Scan(&n); err != nil {
			return true
		}
		return n > 0
	}

	for _, p := range list {
		slug := util.AllocateSlug(p.seed, p.userID, taken)
		if _, err := db.Exec(`
			INSERT INTO bots (user_id, slug, name, welcome_message, active)
			VALUES (?, ?, ?, ?, ?)`,
			p.userID, slug, p.name, p.welcome, p.active,
		); err != nil {
			return fmt.Errorf("backfill bot for user %d: %w", p.userID, err)
		}
	}
	return nil
}
