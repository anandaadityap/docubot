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
	inactiveMessage   = "Bot sedang tidak aktif. Silakan coba lagi nanti."
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
	Done(conversationID, messageID int64, totalTokens int, latencyMS int64) error
	Inactive(message string) error
}

// ChatService orchestrates retrieval + LLM streaming + persistence.
type ChatService struct {
	users    repository.UserRepository
	chunks   repository.ChunkRepository
	convos   repository.ConversationRepository
	msgs     repository.MessageRepository
	settings repository.SettingsRepository
	embedder ai.Embedder
	llm      ai.LLMProvider
}

// NewChatService constructs a ChatService.
func NewChatService(
	users repository.UserRepository,
	chunks repository.ChunkRepository,
	convos repository.ConversationRepository,
	msgs repository.MessageRepository,
	settings repository.SettingsRepository,
	embedder ai.Embedder,
	llm ai.LLMProvider,
) *ChatService {
	return &ChatService{
		users:    users,
		chunks:   chunks,
		convos:   convos,
		msgs:     msgs,
		settings: settings,
		embedder: embedder,
		llm:      llm,
	}
}

// ChatInput is a public chat turn.
type ChatInput struct {
	Message        string
	ConversationID *int64
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

	owner, err := s.users.First(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotConfigured
		}
		return fmt.Errorf("resolve owner: %w", err)
	}

	cfg, err := s.settings.GetByUserID(ctx, owner.ID)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	started := time.Now()

	convo, err := s.resolveConversation(ctx, owner.ID, in.ConversationID, msg)
	if err != nil {
		return err
	}

	if _, err := s.msgs.Create(ctx, &models.Message{
		ConversationID: convo.ID,
		Role:           models.RoleUser,
		Content:        msg,
	}); err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	if !cfg.BotActive {
		botMsg, err := s.msgs.Create(ctx, &models.Message{
			ConversationID: convo.ID,
			Role:           models.RoleBot,
			Content:        inactiveMessage,
		})
		if err != nil {
			return fmt.Errorf("save inactive message: %w", err)
		}
		if err := emit.Inactive(inactiveMessage); err != nil {
			return err
		}
		return emit.Done(convo.ID, botMsg.ID, 0, time.Since(started).Milliseconds())
	}

	sources, hits, err := s.retrieve(ctx, owner.ID, cfg, msg)
	if err != nil {
		return err
	}
	if err := emit.Sources(sources); err != nil {
		return err
	}

	system, user := ai.BuildRAGPrompt(msg, hits)
	var answer strings.Builder
	usage, err := s.llm.ChatStream(ctx, ai.ChatRequest{
		System:      system,
		User:        user,
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
	}, func(token string) error {
		answer.WriteString(token)
		return emit.Token(token)
	})
	if err != nil {
		return fmt.Errorf("llm: %w", err)
	}

	latency := int(time.Since(started).Milliseconds())
	total := usage.TotalTokens
	if total == 0 {
		total = usage.PromptTokens + usage.CompletionTokens
	}
	botMsg, err := s.msgs.Create(ctx, &models.Message{
		ConversationID: convo.ID,
		Role:           models.RoleBot,
		Content:        answer.String(),
		Sources:        sources,
		LatencyMS:      &latency,
		TokenUsage:     &total,
	})
	if err != nil {
		return fmt.Errorf("save bot message: %w", err)
	}
	return emit.Done(convo.ID, botMsg.ID, total, int64(latency))
}

func (s *ChatService) resolveConversation(ctx context.Context, userID int64, id *int64, firstMessage string) (*models.Conversation, error) {
	if id != nil && *id > 0 {
		c, err := s.convos.GetByIDForUser(ctx, *id, userID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, fmt.Errorf("%w: conversation not found", ErrValidation)
			}
			return nil, fmt.Errorf("get conversation: %w", err)
		}
		return c, nil
	}
	title := truncateRunes(firstMessage, titleMaxRunes)
	c, err := s.convos.Create(ctx, userID, title)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return c, nil
}

func (s *ChatService) retrieve(ctx context.Context, userID int64, cfg *models.Settings, query string) ([]models.Source, []ai.ScoredItem, error) {
	vecs, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) != 1 {
		return nil, nil, fmt.Errorf("embed query: expected 1 vector")
	}

	chunks, err := s.chunks.ListReadyWithEmbeddingsForUser(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("load chunks: %w", err)
	}

	items := make([]ai.VectorItem, len(chunks))
	for i, c := range chunks {
		items[i] = ai.VectorItem{
			ID:         c.ID,
			DocumentID: c.DocumentID,
			Filename:   c.Filename,
			Content:    c.Content,
			Embedding:  c.Embedding,
		}
	}

	k := cfg.TopK
	if k < 1 {
		k = 5
	}
	minScore := cfg.MinScore
	hits := ai.TopK(vecs[0], items, k, minScore)

	sources := make([]models.Source, len(hits))
	for i, h := range hits {
		sources[i] = models.Source{
			DocID:    h.DocumentID,
			Filename: h.Filename,
			Snippet:  truncateRunes(h.Content, snippetMaxRunes),
			Score:    roundScore(h.Score),
		}
	}
	return sources, hits, nil
}

func roundScore(s float64) float64 {
	return float64(int(s*100+0.5)) / 100
}
