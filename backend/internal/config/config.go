package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all application settings loaded from environment variables.
type Config struct {
	Port         string
	DatabasePath string
	UploadDir    string
	JWTSecret    string
	CORSOrigins  string

	LLMAPIKey      string
	LLMBaseURL     string
	LLMModel       string
	LLMTemperature float64
	LLMMaxTokens   int

	EmbedAPIKey  string
	EmbedBaseURL string
	EmbedModel   string

	BotTopK     int
	BotMinScore float64
}

// Load reads configuration from environment with sensible local defaults.
func Load() Config {
	applyDotEnvLLMKeys()
	return Config{
		Port:         getEnv("PORT", "8080"),
		DatabasePath: getEnv("DATABASE_PATH", "./data/docubot.db"),
		UploadDir:    getEnv("UPLOAD_DIR", "./data/uploads"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me-please"),
		CORSOrigins:  getEnv("CORS_ORIGINS", "http://localhost:5173"),

		LLMAPIKey:      realKey(getEnv("LLM_API_KEY", "")),
		LLMBaseURL:     getEnv("LLM_BASE_URL", "https://api.deepseek.com/v1"),
		LLMModel:       getEnv("LLM_MODEL", "deepseek-chat"),
		LLMTemperature: getEnvFloat("LLM_TEMPERATURE", 0.3),
		LLMMaxTokens:   getEnvInt("LLM_MAX_TOKENS", 500),

		EmbedAPIKey:  realKey(getEnv("EMBED_API_KEY", "")),
		EmbedBaseURL: getEnv("EMBED_BASE_URL", "https://api.openai.com/v1"),
		EmbedModel:   getEnv("EMBED_MODEL", "text-embedding-3-small"),

		BotTopK:     getEnvInt("BOT_TOP_K", 5),
		BotMinScore: getEnvFloat("BOT_MIN_SCORE", 0.3),
	}
}

// HasRealAPIKey reports whether key looks like a real secret (not empty / placeholder).
func HasRealAPIKey(key string) bool {
	return realKey(key) != ""
}

func realKey(key string) string {
	k := strings.TrimSpace(key)
	if k == "" {
		return ""
	}
	lower := strings.ToLower(k)
	if k == "sk-..." || strings.Contains(lower, "your-key") || strings.Contains(lower, "changeme") {
		return ""
	}
	if strings.HasSuffix(k, "...") && len(k) < 24 {
		return ""
	}
	return k
}

// applyDotEnvLLMKeys reads .env / ../.env for LLM_* and EMBED_* only (does not override
// DATABASE_PATH so Docker paths in the repo .env do not break local `go run`).
func applyDotEnvLLMKeys() {
	for _, path := range []string{".env", "../.env"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, val, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			if !strings.HasPrefix(key, "LLM_") && !strings.HasPrefix(key, "EMBED_") {
				continue
			}
			if os.Getenv(key) != "" {
				continue
			}
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"'`)
			if realKey(val) == "" {
				continue
			}
			_ = os.Setenv(key, val)
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
