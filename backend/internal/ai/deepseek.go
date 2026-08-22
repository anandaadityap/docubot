package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIChat is an OpenAI-compatible chat completions client (DeepSeek, OpenAI, etc.).
type OpenAIChat struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewOpenAIChat constructs a streaming chat client.
func NewOpenAIChat(apiKey, baseURL, model string) *OpenAIChat {
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = "deepseek-chat"
	}
	return &OpenAIChat{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

type chatCompletionRequest struct {
	Model         string        `json:"model"`
	Messages      []chatMessage `json:"messages"`
	Temperature   float64       `json:"temperature"`
	MaxTokens     int           `json:"max_tokens"`
	Stream        bool          `json:"stream"`
	StreamOptions *streamOpts   `json:"stream_options,omitempty"`
}

type streamOpts struct {
	IncludeUsage bool `json:"include_usage"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ChatStream implements LLMProvider using SSE from /chat/completions.
func (c *OpenAIChat) ChatStream(ctx context.Context, req ChatRequest, onToken func(token string) error) (TokenUsage, error) {
	if c.apiKey == "" {
		return TokenUsage{}, fmt.Errorf("llm api key missing")
	}
	maxTok := req.MaxTokens
	if maxTok < 1 {
		maxTok = 500
	}
	if maxTok > 500 {
		maxTok = 500
	}

	body, err := json.Marshal(chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.User},
		},
		Temperature:   req.Temperature,
		MaxTokens:     maxTok,
		Stream:        true,
		StreamOptions: &streamOpts{IncludeUsage: true},
	})
	if err != nil {
		return TokenUsage{}, fmt.Errorf("marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return TokenUsage{}, fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return TokenUsage{}, fmt.Errorf("chat request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return TokenUsage{}, fmt.Errorf("llm API status %d: %s", resp.StatusCode, truncateStr(string(raw), 300))
	}

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var usage TokenUsage
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return usage, err
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return usage, fmt.Errorf("llm API error: %s", chunk.Error.Message)
		}
		if chunk.Usage != nil {
			usage = TokenUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}
		for _, ch := range chunk.Choices {
			if ch.Delta.Content == "" {
				continue
			}
			if err := onToken(ch.Delta.Content); err != nil {
				return usage, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return usage, fmt.Errorf("read llm stream: %w", err)
	}
	return usage, nil
}

var _ LLMProvider = (*OpenAIChat)(nil)
