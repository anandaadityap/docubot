package service

import (
	"context"
	"time"

	"github.com/supernand/docubot/backend/internal/repository"
)

// AnalyticsService computes dashboard metrics.
type AnalyticsService struct {
	repo repository.AnalyticsRepository
	now  func() time.Time
}

// NewAnalyticsService constructs an AnalyticsService.
func NewAnalyticsService(repo repository.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo, now: time.Now}
}

// Overview is GET /analytics/overview payload.
type Overview struct {
	TotalConversations int         `json:"total_conversations"`
	TotalMessages      int         `json:"total_messages"`
	TotalBotMessages   int         `json:"total_bot_messages"`
	AvgLatencyMS       int         `json:"avg_latency_ms"`
	Daily              []DailyStat `json:"daily"`
}

// DailyStat is chats per day.
type DailyStat struct {
	Date  string `json:"date"`
	Chats int    `json:"chats"`
}

// TopQuestion is a grouped user question.
type TopQuestion struct {
	Question string `json:"question"`
	Count    int    `json:"count"`
}

// OverviewLast14Days returns totals plus a 14-day series (missing days filled with 0).
func (s *AnalyticsService) OverviewLast14Days(ctx context.Context, userID int64) (*Overview, error) {
	now := s.now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	since := today.AddDate(0, 0, -13)

	totalConv, totalMsg, totalBot, avgLatency, daily, err := s.repo.Overview(ctx, userID, since)
	if err != nil {
		return nil, err
	}

	byDate := make(map[string]int, len(daily))
	for _, d := range daily {
		byDate[d.Date] = d.Chats
	}
	filled := make([]DailyStat, 0, 14)
	for i := 0; i < 14; i++ {
		day := since.AddDate(0, 0, i).Format("2006-01-02")
		filled = append(filled, DailyStat{Date: day, Chats: byDate[day]})
	}

	return &Overview{
		TotalConversations: totalConv,
		TotalMessages:      totalMsg,
		TotalBotMessages:   totalBot,
		AvgLatencyMS:       int(avgLatency + 0.5),
		Daily:              filled,
	}, nil
}

// TopQuestions returns the most common user questions.
func (s *AnalyticsService) TopQuestions(ctx context.Context, userID int64, limit int) ([]TopQuestion, error) {
	if limit < 1 || limit > 50 {
		limit = 10
	}
	rows, err := s.repo.TopQuestions(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]TopQuestion, len(rows))
	for i, r := range rows {
		out[i] = TopQuestion{Question: r.Question, Count: r.Count}
	}
	return out, nil
}
