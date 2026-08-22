package service

import (
	"context"
	"errors"

	"github.com/supernand/docubot/backend/internal/models"
	"github.com/supernand/docubot/backend/internal/repository"
)

// ConversationService is the admin conversation viewer.
type ConversationService struct {
	convos repository.ConversationRepository
	msgs   repository.MessageRepository
}

// NewConversationService constructs a ConversationService.
func NewConversationService(convos repository.ConversationRepository, msgs repository.MessageRepository) *ConversationService {
	return &ConversationService{convos: convos, msgs: msgs}
}

// ListPage is a paginated conversation list.
type ListPage struct {
	Items []models.Conversation `json:"items"`
	Total int                   `json:"total"`
	Page  int                   `json:"page"`
}

// List returns conversations for the admin.
func (s *ConversationService) List(ctx context.Context, userID int64, page, limit int) (*ListPage, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	items, total, err := s.convos.ListByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}
	return &ListPage{Items: items, Total: total, Page: page}, nil
}

// ConversationDetail is a conversation plus messages.
type ConversationDetail struct {
	models.Conversation
	Messages []models.Message `json:"messages"`
}

// Get returns one conversation owned by userID.
func (s *ConversationService) Get(ctx context.Context, userID, id int64) (*ConversationDetail, error) {
	c, err := s.convos.GetByIDForUser(ctx, id, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	msgs, err := s.msgs.ListByConversation(ctx, id)
	if err != nil {
		return nil, err
	}
	return &ConversationDetail{Conversation: *c, Messages: msgs}, nil
}
