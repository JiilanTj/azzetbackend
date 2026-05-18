package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	defaultBaseURL  = "https://api.openai.com/v1"
	defaultTimeout  = 30 * time.Second
	defaultMaxTokens = 4096
)

type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

type Config struct {
	APIKey     string
	Model      string
	BaseURL    string
	Timeout    time.Duration
	MaxTokens  int
}

func New(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = defaultMaxTokens
	}

	return &Client{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func NewFromEnv(apiKey, model string) *Client {
	return New(Config{
		APIKey: apiKey,
		Model:  model,
	})
}

func (c *Client) HealthCheck(ctx context.Context) error {
	req := ChatRequest{
		Model: c.model,
		Messages: []Message{
			{Role: "user", Content: "Say hello in one word"},
		},
		MaxTokens: 10,
	}

	resp, err := c.Chat(ctx, req)
	if err != nil {
		return err
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("empty response from OpenAI")
	}

	slog.Info("openai connected", "model", c.model)
	return nil
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Model = c.model

	if req.MaxTokens == 0 {
		req.MaxTokens = defaultMaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	slog.Debug("openai chat response",
		"model", chatResp.Model,
		"tokens_prompt", chatResp.Usage.PromptTokens,
		"tokens_completion", chatResp.Usage.CompletionTokens,
	)

	return &chatResp, nil
}

func (c *Client) ChatJSON(ctx context.Context, systemPrompt, userPrompt string, dest interface{}) error {
	req := ChatRequest{
		Messages: []Message{
			{Role: "system", Content: systemPrompt + "\n\nRespond ONLY with valid JSON. No markdown, no code blocks, no extra text."},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	resp, err := c.Chat(ctx, req)
	if err != nil {
		return err
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("empty response from OpenAI")
	}

	content := resp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), dest); err != nil {
		slog.Error("failed to parse JSON from AI response", "content", content, "error", err)
		return fmt.Errorf("invalid JSON from AI: %w", err)
	}

	return nil
}
