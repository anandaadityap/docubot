package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/supernand/docubot/backend/internal/ai"
	"github.com/supernand/docubot/backend/internal/config"
	"github.com/supernand/docubot/backend/internal/database"
	"github.com/supernand/docubot/backend/internal/handler"
	"github.com/supernand/docubot/backend/internal/middleware"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/service"
)

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" || cfg.JWTSecret == "change-me-please" {
		log.Printf("warning: JWT_SECRET is using the insecure default; set a strong secret in production")
	}

	db, err := database.Open(cfg.DatabasePath, cfg.UploadDir)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	authH := handler.NewAuthHandler(authSvc)

	var embedder ai.Embedder
	if config.HasRealAPIKey(cfg.EmbedAPIKey) {
		embedder = ai.NewOpenAIEmbedder(cfg.EmbedAPIKey, cfg.EmbedBaseURL, cfg.EmbedModel)
	} else {
		log.Printf("warning: EMBED_API_KEY missing or placeholder — using StubEmbedder")
		embedder = ai.NewStubEmbedder()
	}

	var llm ai.LLMProvider
	if config.HasRealAPIKey(cfg.LLMAPIKey) {
		llm = ai.NewOpenAIChat(cfg.LLMAPIKey, cfg.LLMBaseURL, cfg.LLMModel)
	} else {
		log.Printf("warning: LLM_API_KEY missing or placeholder — using StubLLM (short extractive answers)")
		llm = &ai.StubLLM{}
	}

	docSvc := service.NewDocumentService(
		repository.NewDocumentRepo(db),
		repository.NewChunkRepo(db),
		embedder,
		cfg.UploadDir,
	)
	docH := handler.NewDocumentHandler(docSvc)

	chunkRepo := repository.NewChunkRepo(db)
	convoRepo := repository.NewConversationRepo(db)
	msgRepo := repository.NewMessageRepo(db)
	settingsRepo := repository.NewSettingsRepo(db)

	chatSvc := service.NewChatService(userRepo, chunkRepo, convoRepo, msgRepo, settingsRepo, embedder, llm)
	chatH := handler.NewChatHandler(chatSvc)

	settingsSvc := service.NewSettingsService(userRepo, settingsRepo)
	settingsH := handler.NewSettingsHandler(settingsSvc)

	convoSvc := service.NewConversationService(convoRepo, msgRepo)
	convoH := handler.NewConversationHandler(convoSvc)

	analyticsSvc := service.NewAnalyticsService(repository.NewAnalyticsRepo(db))
	analyticsH := handler.NewAnalyticsHandler(analyticsSvc)

	r := gin.Default()
	_ = r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.1", "::1"})
	r.Use(middleware.CORS(cfg.CORSOrigins))
	r.GET("/healthz", handler.Health)

	limiter := middleware.NewRateLimiter(10, 0)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/register", authH.Register)
		v1.POST("/auth/login", authH.Login)
		v1.GET("/bot", settingsH.PublicBot)
		v1.POST("/chat", middleware.ChatRateLimit(limiter), chatH.Chat)

		authed := v1.Group("")
		authed.Use(middleware.AuthWithParser(authSvc))
		authed.GET("/auth/me", authH.Me)

		authed.POST("/documents", docH.Upload)
		authed.GET("/documents", docH.List)
		authed.GET("/documents/:id", docH.Get)
		authed.DELETE("/documents/:id", docH.Delete)
		authed.POST("/documents/:id/reprocess", docH.Reprocess)

		authed.GET("/conversations", convoH.List)
		authed.GET("/conversations/:id", convoH.Get)

		authed.GET("/analytics/overview", analyticsH.Overview)
		authed.GET("/analytics/top-questions", analyticsH.TopQuestions)

		authed.GET("/settings", settingsH.Get)
		authed.PUT("/settings", settingsH.Put)
	}

	addr := ":" + cfg.Port
	log.Printf("DocuBot API listening on %s", addr)
	if err := r.Run(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}
