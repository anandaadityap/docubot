package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/repository"
	"github.com/supernand/docubot/backend/internal/util"
)

var (
	// ErrBotNotFound is returned when a public slug does not exist.
	ErrBotNotFound = errors.New("bot not found")
	// ErrSlugTaken is returned when updating to a slug owned by another user.
	ErrSlugTaken = errors.New("slug taken")
)

// BotService loads and updates bot identity (slug, name, welcome, active).
type BotService struct {
	bots     repository.BotRepository
	settings repository.SettingsRepository
	docs     repository.DocumentRepository
}

// NewBotService constructs a BotService.
func NewBotService(bots repository.BotRepository, settings repository.SettingsRepository, docs repository.DocumentRepository) *BotService {
	return &BotService{bots: bots, settings: settings, docs: docs}
}

// PublicProfile is GET /api/v1/bots/:slug. Missing slug is ErrBotNotFound (not another bot).
func (s *BotService) PublicProfile(ctx context.Context, slug string) (*models.PublicBot, error) {
	bot, err := s.bots.GetBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBotNotFound
		}
		return nil, err
	}
	ready := 0
	if s.docs != nil {
		ready, err = s.docs.CountReadyForUser(ctx, bot.UserID)
		if err != nil {
			return nil, err
		}
	}
	return &models.PublicBot{
		Slug:           bot.Slug,
		BotName:        bot.Name,
		WelcomeMessage: bot.WelcomeMessage,
		BotActive:      bot.Active,
		Configured:     true,
		HasReadyKB:     ready > 0,
	}, nil
}

// Demo returns the oldest bot for the landing "Coba demo" button.
// No users: configured=false with HTTP 200 (not 404).
func (s *BotService) Demo(ctx context.Context) (*models.DemoBot, error) {
	bot, err := s.bots.GetOldest(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return &models.DemoBot{Configured: false}, nil
		}
		return nil, err
	}
	ready := 0
	if s.docs != nil {
		ready, err = s.docs.CountReadyForUser(ctx, bot.UserID)
		if err != nil {
			return nil, err
		}
	}
	return &models.DemoBot{
		Slug:       bot.Slug,
		BotName:    bot.Name,
		HasReadyKB: ready > 0,
		Configured: true,
	}, nil
}

// GetByUser returns the admin bot payload for JWT userID.
func (s *BotService) GetByUser(ctx context.Context, userID int64) (*models.AdminBot, error) {
	bot, err := s.bots.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBotNotFound
		}
		return nil, err
	}
	return adminBot(bot), nil
}

// UpdateBotInput is PUT /api/v1/admin/bot.
type UpdateBotInput struct {
	Slug           string
	Name           string
	WelcomeMessage string
	Active         bool
}

// Update validates slug/name/welcome, writes bots, and mirrors identity onto settings.
func (s *BotService) Update(ctx context.Context, userID int64, in UpdateBotInput) (*models.AdminBot, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if utf8.RuneCountInString(name) > 80 {
		return nil, fmt.Errorf("%w: name is too long", ErrValidation)
	}
	welcome := strings.TrimSpace(in.WelcomeMessage)
	if welcome == "" {
		return nil, fmt.Errorf("%w: welcome_message is required", ErrValidation)
	}
	if utf8.RuneCountInString(welcome) > 500 {
		return nil, fmt.Errorf("%w: welcome_message is too long", ErrValidation)
	}

	slug := util.Slugify(in.Slug)
	if !util.ValidSlug(slug) {
		return nil, fmt.Errorf("%w: slug must be 3–48 characters (a-z, 0-9, hyphen) and not reserved", ErrValidation)
	}

	cur, err := s.bots.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrBotNotFound
		}
		return nil, err
	}

	taken, err := s.bots.SlugTaken(ctx, slug, userID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrSlugTaken
	}

	cur.Slug = slug
	cur.Name = name
	cur.WelcomeMessage = welcome
	cur.Active = in.Active
	if err := s.bots.Update(ctx, cur); err != nil {
		if errors.Is(err, repository.ErrSlugTaken) {
			return nil, ErrSlugTaken
		}
		return nil, err
	}

	if err := s.mirrorSettingsIdentity(ctx, userID, name, welcome, in.Active); err != nil {
		return nil, err
	}

	updated, err := s.bots.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return adminBot(updated), nil
}

func (s *BotService) mirrorSettingsIdentity(ctx context.Context, userID int64, name, welcome string, active bool) error {
	cfg, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil
		}
		return err
	}
	cfg.BotName = name
	cfg.WelcomeMessage = welcome
	cfg.BotActive = active
	return s.settings.Update(ctx, cfg)
}

func adminBot(b *models.Bot) *models.AdminBot {
	return &models.AdminBot{
		Slug:           b.Slug,
		Name:           b.Name,
		WelcomeMessage: b.WelcomeMessage,
		Active:         b.Active,
		PublicPath:     "/b/" + b.Slug,
		EmbedPath:      "/b/" + b.Slug + "?embed=1",
	}
}
