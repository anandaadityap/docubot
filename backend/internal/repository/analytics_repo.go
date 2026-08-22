package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DailyStat is chats created on a calendar day.
type DailyStat struct {
	Date  string
	Chats int
}

// TopQuestion is a grouped user question.
type TopQuestion struct {
	Question string
	Count    int
}

// AnalyticsRepository computes dashboard aggregates.
type AnalyticsRepository interface {
	Overview(ctx context.Context, userID int64, since time.Time) (totalConv, totalMsg, totalBot int, avgLatency float64, daily []DailyStat, err error)
	TopQuestions(ctx context.Context, userID int64, limit int) ([]TopQuestion, error)
}

// AnalyticsRepo is the SQLite implementation.
type AnalyticsRepo struct {
	db *sql.DB
}

// NewAnalyticsRepo returns a SQLite-backed AnalyticsRepository.
func NewAnalyticsRepo(db *sql.DB) *AnalyticsRepo {
	return &AnalyticsRepo{db: db}
}

// Overview returns totals, average bot latency, and daily conversation counts since `since`.
func (r *AnalyticsRepo) Overview(ctx context.Context, userID int64, since time.Time) (int, int, int, float64, []DailyStat, error) {
	var totalConv, totalMsg, totalBot int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM conversations WHERE user_id = ?`, userID,
	).Scan(&totalConv); err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("count conversations: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM messages m
		INNER JOIN conversations c ON c.id = m.conversation_id
		WHERE c.user_id = ?`, userID,
	).Scan(&totalMsg); err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("count messages: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM messages m
		INNER JOIN conversations c ON c.id = m.conversation_id
		WHERE c.user_id = ? AND m.role = 'bot'`, userID,
	).Scan(&totalBot); err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("count bot messages: %w", err)
	}

	var avg sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, `
		SELECT AVG(m.latency_ms) FROM messages m
		INNER JOIN conversations c ON c.id = m.conversation_id
		WHERE c.user_id = ? AND m.role = 'bot' AND m.latency_ms IS NOT NULL`, userID,
	).Scan(&avg); err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("avg latency: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT date(created_at) AS d, COUNT(*)
		FROM conversations
		WHERE user_id = ? AND created_at >= ?
		GROUP BY d
		ORDER BY d ASC`,
		userID, since.UTC().Format("2006-01-02"),
	)
	if err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("daily chats: %w", err)
	}
	defer rows.Close()

	var daily []DailyStat
	for rows.Next() {
		var st DailyStat
		if err := rows.Scan(&st.Date, &st.Chats); err != nil {
			return 0, 0, 0, 0, nil, fmt.Errorf("scan daily: %w", err)
		}
		daily = append(daily, st)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("daily rows: %w", err)
	}
	if daily == nil {
		daily = []DailyStat{}
	}

	avgVal := 0.0
	if avg.Valid {
		avgVal = avg.Float64
	}
	return totalConv, totalMsg, totalBot, avgVal, daily, nil
}

// TopQuestions returns the most frequent user messages.
func (r *AnalyticsRepo) TopQuestions(ctx context.Context, userID int64, limit int) ([]TopQuestion, error) {
	if limit < 1 {
		limit = 10
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT MIN(m.content) AS question, COUNT(*) AS cnt
		FROM messages m
		INNER JOIN conversations c ON c.id = m.conversation_id
		WHERE c.user_id = ? AND m.role = 'user'
		GROUP BY LOWER(TRIM(m.content))
		ORDER BY cnt DESC
		LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("top questions: %w", err)
	}
	defer rows.Close()

	var out []TopQuestion
	for rows.Next() {
		var q TopQuestion
		if err := rows.Scan(&q.Question, &q.Count); err != nil {
			return nil, fmt.Errorf("scan top question: %w", err)
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("top questions rows: %w", err)
	}
	if out == nil {
		out = []TopQuestion{}
	}
	return out, nil
}
