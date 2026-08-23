package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/supernand/docubot/backend/internal/ai"
	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/repository"
)

const (
	maxChatMessageLen = 2000
	titleMaxRunes     = 60
	snippetMaxRunes   = 180
	historyMaxMsgs    = 6
	historyMaxRunes   = 400
	inactiveMessage   = "Bot sedang tidak aktif. Silakan coba lagi nanti."
	emptyKBMessage    = "Belum ada dokumen knowledge base yang siap. Saya tidak bisa menjawab sebelum pengelola mengunggah file dan statusnya Ready."
	noHitMessage      = "Maaf, saya tidak menemukan informasi itu di dokumen knowledge base. Silakan tanya hal lain yang ada di panduan."
	mismatchMessage   = "Dokumen knowledge base diproses dengan model embedding yang berbeda. Pengelola perlu menekan Proses ulang pada halaman Dokumen."
	llmFailMessage    = "Maaf, terjadi gangguan saat menjawab. Silakan coba lagi."
)

var (
	// ErrBotInactive is returned when the admin has turned the bot off.
	ErrBotInactive = errors.New("bot inactive")
	// ErrNotConfigured is returned when no admin account exists yet.
	ErrNotConfigured = errors.New("bot not configured")
)

// ChatEmitter receives RAG stream events.
type ChatEmitter interface {
	Sources(sources []models.Source) error
	Token(content string) error
	Done(conversationPublicID string, messageID int64, totalTokens int, latencyMS int64) error
	Inactive(message string) error
}

// ChatService orchestrates retrieval + LLM streaming + persistence.
type ChatService struct {
	bots     repository.BotRepository
	docs     repository.DocumentRepository
	chunks   repository.ChunkRepository
	convos   repository.ConversationRepository
	msgs     repository.MessageRepository
	settings repository.SettingsRepository
	embedder ai.Embedder
	llm      ai.LLMProvider
}

// NewChatService constructs a ChatService.
func NewChatService(
	bots repository.BotRepository,
	docs repository.DocumentRepository,
	chunks repository.ChunkRepository,
	convos repository.ConversationRepository,
	msgs repository.MessageRepository,
	settings repository.SettingsRepository,
	embedder ai.Embedder,
	llm ai.LLMProvider,
) *ChatService {
	return &ChatService{
		bots:     bots,
		docs:     docs,
		chunks:   chunks,
		convos:   convos,
		msgs:     msgs,
		settings: settings,
		embedder: embedder,
		llm:      llm,
	}
}

// ChatInput is a public chat turn. Slug is required; Channel defaults to public.
type ChatInput struct {
	Slug           string
	Message        string
	ConversationID *string
	Channel        string
}

// LookupSlug resolves a bot by slug without answering a question.
func (s *ChatService) LookupSlug(ctx context.Context, slug string) (*models.Bot, error) {
	bot, err := s.bots.GetBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBotNotFound
		}
		return nil, fmt.Errorf("lookup bot: %w", err)
	}
	return bot, nil
}

// Chat runs one RAG turn and streams via emit.
func (s *ChatService) Chat(ctx context.Context, in ChatInput, emit ChatEmitter) error {
	msg := strings.TrimSpace(in.Message)
	if msg == "" {
		return fmt.Errorf("%w: message is required", ErrValidation)
	}
	if utf8.RuneCountInString(msg) > maxChatMessageLen {
		return fmt.Errorf("%w: message is too long (max %d characters)", ErrValidation, maxChatMessageLen)
	}

	channel := strings.TrimSpace(in.Channel)
	if channel == "" {
		channel = models.ChannelPublic
	}
	if channel != models.ChannelPublic && channel != models.ChannelPlayground {
		return fmt.Errorf("%w: channel must be public or playground", ErrValidation)
	}

	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		return ErrBotNotFound
	}

	bot, err := s.LookupSlug(ctx, slug)
	if err != nil {
		return err
	}

	cfg, err := s.settings.GetByUserID(ctx, bot.UserID)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	started := time.Now()

	convo, err := s.resolveConversation(ctx, bot.UserID, in.ConversationID, msg, channel)
	if err != nil {
		return err
	}

	history, err := s.loadHistory(ctx, convo.ID)
	if err != nil {
		return err
	}

	if !bot.Active {
		return s.finishLocal(ctx, convo, emit, started, msg, inactiveMessage, nil, true)
	}

	ready, err := s.docs.CountReadyForUser(ctx, bot.UserID)
	if err != nil {
		return fmt.Errorf("count ready documents: %w", err)
	}
	if ready == 0 {
		return s.finishLocal(ctx, convo, emit, started, msg, emptyKBMessage, nil, false)
	}

	sources, hits, mismatch, err := s.retrieve(ctx, bot.UserID, cfg, retrievalQuery(msg, history))
	if err != nil {
		return err
	}
	if mismatch {
		return s.finishLocal(ctx, convo, emit, started, msg, mismatchMessage, nil, false)
	}
	if len(hits) == 0 {
		return s.finishLocal(ctx, convo, emit, started, msg, noHitMessage, sources, false)
	}

	if err := emit.Sources(sources); err != nil {
		return err
	}

	system, user := ai.BuildRAGPrompt(msg, hits)
	var answer strings.Builder
	usage, err := s.llm.ChatStream(ctx, ai.ChatRequest{
		System:      system,
		History:     history,
		User:        user,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
	}, func(token string) error {
		answer.WriteString(token)
		return emit.Token(token)
	})
	if err != nil {
		_, _ = s.persistTurn(ctx, convo.ID, msg, llmFailMessage, nil, 0, time.Since(started).Milliseconds())
		return fmt.Errorf("llm: %w", err)
	}

	latency := time.Since(started).Milliseconds()
	total := usage.TotalTokens
	if total == 0 {
		total = usage.PromptTokens + usage.CompletionTokens
	}
	botMsg, err := s.persistTurn(ctx, convo.ID, msg, answer.String(), sources, total, latency)
	if err != nil {
		return err
	}
	return emit.Done(convo.PublicID, botMsg.ID, total, latency)
}

func (s *ChatService) finishLocal(
	ctx context.Context,
	convo *models.Conversation,
	emit ChatEmitter,
	started time.Time,
	userText, botText string,
	sources []models.Source,
	inactive bool,
) error {
	if sources == nil {
		sources = []models.Source{}
	}
	if inactive {
		if err := emit.Inactive(botText); err != nil {
			return err
		}
	} else {
		if err := emit.Sources(sources); err != nil {
			return err
		}
		if err := emit.Token(botText); err != nil {
			return err
		}
	}
	latency := time.Since(started).Milliseconds()
	botMsg, err := s.persistTurn(ctx, convo.ID, userText, botText, sources, 0, latency)
	if err != nil {
		return err
	}
	return emit.Done(convo.PublicID, botMsg.ID, 0, latency)
}

func (s *ChatService) persistTurn(ctx context.Context, conversationID int64, userText, botText string, sources []models.Source, tokens int, latencyMS int64) (*models.Message, error) {
	if _, err := s.msgs.Create(ctx, &models.Message{
		ConversationID: conversationID,
		Role:           models.RoleUser,
		Content:        userText,
	}); err != nil {
		return nil, fmt.Errorf("save user message: %w", err)
	}
	lat := int(latencyMS)
	bot := &models.Message{
		ConversationID: conversationID,
		Role:           models.RoleBot,
		Content:        botText,
		Sources:        sources,
		LatencyMS:      &lat,
	}
	if tokens > 0 {
		bot.TokenUsage = &tokens
	}
	saved, err := s.msgs.Create(ctx, bot)
	if err != nil {
		return nil, fmt.Errorf("save bot message: %w", err)
	}
	return saved, nil
}

func (s *ChatService) resolveConversation(ctx context.Context, userID int64, publicID *string, firstMessage, channel string) (*models.Conversation, error) {
	if publicID != nil {
		id := strings.TrimSpace(*publicID)
		if id != "" {
			c, err := s.convos.GetByPublicID(ctx, id, userID)
			if err != nil {
				if errors.Is(err, repository.ErrNotFound) {
					return nil, fmt.Errorf("%w: conversation not found", ErrValidation)
				}
				return nil, fmt.Errorf("get conversation: %w", err)
			}
			return c, nil
		}
	}
	title := truncateRunes(firstMessage, titleMaxRunes)
	c, err := s.convos.Create(ctx, userID, title, channel)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return c, nil
}

func (s *ChatService) loadHistory(ctx context.Context, conversationID int64) ([]ai.ChatTurn, error) {
	list, err := s.msgs.ListByConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}
	if len(list) > historyMaxMsgs {
		list = list[len(list)-historyMaxMsgs:]
	}
	out := make([]ai.ChatTurn, 0, len(list))
	for _, m := range list {
		role := "user"
		if m.Role == models.RoleBot {
			role = "assistant"
		}
		out = append(out, ai.ChatTurn{
			Role:    role,
			Content: truncateRunes(m.Content, historyMaxRunes),
		})
	}
	return out, nil
}

func (s *ChatService) retrieve(ctx context.Context, userID int64, cfg *models.Settings, query string) ([]models.Source, []ai.ScoredItem, bool, error) {
	vecs, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, nil, false, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) != 1 {
		return nil, nil, false, fmt.Errorf("embed query: expected 1 vector")
	}
	qdim := len(vecs[0])

	chunks, err := s.chunks.ListReadyWithEmbeddingsForUser(ctx, userID)
	if err != nil {
		return nil, nil, false, fmt.Errorf("load chunks: %w", err)
	}

	skipped := 0
	items := make([]ai.VectorItem, 0, len(chunks))
	for _, c := range chunks {
		dim := c.EmbedDim
		if dim == 0 {
			dim = len(c.Embedding)
		}
		if dim != qdim || len(c.Embedding) != qdim {
			skipped++
			continue
		}
		items = append(items, ai.VectorItem{
			ID:         c.ID,
			DocumentID: c.DocumentID,
			Filename:   c.Filename,
			Content:    c.Content,
			Embedding:  c.Embedding,
		})
	}

	if len(chunks) > 0 && len(items) == 0 && skipped > 0 {
		return nil, nil, true, nil
	}

	k := cfg.TopK
	if k < 1 {
		k = 5
	}
	hits := ai.TopK(vecs[0], items, k, cfg.MinScore)

	sources := make([]models.Source, len(hits))
	for i, h := range hits {
		sources[i] = models.Source{
			DocID:    h.DocumentID,
			Filename: h.Filename,
			Snippet:  truncateRunes(h.Content, snippetMaxRunes),
			Score:    roundScore(h.Score),
		}
	}
	return sources, hits, false, nil
}

func retrievalQuery(current string, history []ai.ChatTurn) string {
	var lastUser string
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			lastUser = strings.TrimSpace(history[i].Content)
			break
		}
	}
	if lastUser == "" || lastUser == current {
		return current
	}
	return lastUser + "\n" + current
}

func roundScore(s float64) float64 {
	return float64(int(s*100+0.5)) / 100
}
