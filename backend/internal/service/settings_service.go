package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/repository"
)

// SettingsService loads and updates bot settings.
type SettingsService struct {
	users    repository.UserRepository
	settings repository.SettingsRepository
}

// NewSettingsService constructs a SettingsService.
func NewSettingsService(users repository.UserRepository, settings repository.SettingsRepository) *SettingsService {
	return &SettingsService{users: users, settings: settings}
}

// GetPublic returns the owner bot profile for the public chat page.
func (s *SettingsService) GetPublic(ctx context.Context) (*models.PublicBot, error) {
	owner, err := s.users.First(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &models.PublicBot{
				BotName:        "DocuBot",
				WelcomeMessage: "Halo! Ada yang bisa saya bantu?",
				BotActive:      false,
				Configured:     false,
			}, nil
		}
		return nil, err
	}
	cfg, err := s.settings.GetByUserID(ctx, owner.ID)
	if err != nil {
		return nil, err
	}
	return &models.PublicBot{
		BotName:        cfg.BotName,
		WelcomeMessage: cfg.WelcomeMessage,
		BotActive:      cfg.BotActive,
		Configured:     true,
	}, nil
}

// Get returns admin settings for userID.
func (s *SettingsService) Get(ctx context.Context, userID int64) (*models.Settings, error) {
	cfg, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return cfg, nil
}

// UpdateInput is a partial-safe full replace of settings.
type UpdateInput struct {
	BotName        string
	WelcomeMessage string
	BotActive      bool
	Temperature    float64
	MaxTokens      int
	TopK           int
	MinScore       float64
}

// Update validates and saves settings.
func (s *SettingsService) Update(ctx context.Context, userID int64, in UpdateInput) (*models.Settings, error) {
	name := strings.TrimSpace(in.BotName)
	if name == "" {
		return nil, fmt.Errorf("%w: bot_name is required", ErrValidation)
	}
	if utf8.RuneCountInString(name) > 80 {
		return nil, fmt.Errorf("%w: bot_name is too long", ErrValidation)
	}
	welcome := strings.TrimSpace(in.WelcomeMessage)
	if welcome == "" {
		return nil, fmt.Errorf("%w: welcome_message is required", ErrValidation)
	}
	if utf8.RuneCountInString(welcome) > 500 {
		return nil, fmt.Errorf("%w: welcome_message is too long", ErrValidation)
	}
	if in.Temperature < 0 || in.Temperature > 2 {
		return nil, fmt.Errorf("%w: temperature must be between 0 and 2", ErrValidation)
	}
	if in.MaxTokens < 1 || in.MaxTokens > 500 {
		return nil, fmt.Errorf("%w: max_tokens must be between 1 and 500", ErrValidation)
	}
	if in.TopK < 1 || in.TopK > 20 {
		return nil, fmt.Errorf("%w: top_k must be between 1 and 20", ErrValidation)
	}
	if in.MinScore < 0 || in.MinScore > 1 {
		return nil, fmt.Errorf("%w: min_score must be between 0 and 1", ErrValidation)
	}

	cur, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	cur.BotName = name
	cur.WelcomeMessage = welcome
	cur.BotActive = in.BotActive
	cur.Temperature = in.Temperature
	cur.MaxTokens = in.MaxTokens
	cur.TopK = in.TopK
	cur.MinScore = in.MinScore
	if err := s.settings.Update(ctx, cur); err != nil {
		return nil, err
	}
	return s.settings.GetByUserID(ctx, userID)
}
