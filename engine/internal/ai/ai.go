// Package ai is an optional, bring-your-own-key assistant for explaining
// conflicts and summarizing logs. It is self-contained: the sync engine never
// depends on it, and it reaches a provider only when the user configures one.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Kind string

const (
	OpenAI     Kind = "openai"
	Anthropic  Kind = "anthropic"
	Gemini     Kind = "gemini"
	OpenRouter Kind = "openrouter"
	Ollama     Kind = "ollama"
	LMStudio   Kind = "lmstudio"
	Custom     Kind = "custom"
)

type Config struct {
	Enabled bool   `json:"enabled"`
	Kind    Kind   `json:"kind"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`
	APIKey  string `json:"api_key"`
}

var (
	ErrDisabled     = errors.New("ai: assistant is disabled")
	ErrUnknownKind  = errors.New("ai: unknown provider")
	ErrNoModel      = errors.New("ai: no model configured")
	ErrNoBaseURL    = errors.New("ai: a custom provider needs a base URL")
	ErrEmptyReply   = errors.New("ai: provider returned no content")
	ErrNoAPIKey     = errors.New("ai: this provider needs an API key")
	defaultBaseURLs = map[Kind]string{
		OpenAI:     "https://api.openai.com/v1",
		OpenRouter: "https://openrouter.ai/api/v1",
		Ollama:     "http://localhost:11434/v1",
		LMStudio:   "http://localhost:1234/v1",
		Anthropic:  "https://api.anthropic.com/v1",
		Gemini:     "https://generativelanguage.googleapis.com/v1beta",
	}
)

func (c Config) Validate() error {
	if _, ok := knownKinds[c.Kind]; !ok {
		return ErrUnknownKind
	}
	if c.Model == "" {
		return ErrNoModel
	}
	if c.effectiveBaseURL() == "" {
		return ErrNoBaseURL
	}
	if (c.Kind == OpenAI || c.Kind == OpenRouter || c.Kind == Anthropic || c.Kind == Gemini) && c.APIKey == "" {
		return ErrNoAPIKey
	}
	return nil
}

func (c Config) effectiveBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return defaultBaseURLs[c.Kind]
}

var knownKinds = map[Kind]bool{
	OpenAI: true, Anthropic: true, Gemini: true,
	OpenRouter: true, Ollama: true, LMStudio: true, Custom: true,
}

type Client struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 60 * time.Second}}
}

func (c *Client) Complete(ctx context.Context, system, user string) (string, error) {
	if !c.cfg.Enabled {
		return "", ErrDisabled
	}
	if err := c.cfg.Validate(); err != nil {
		return "", err
	}
	switch c.cfg.Kind {
	case Anthropic:
		return c.completeAnthropic(ctx, system, user)
	case Gemini:
		return c.completeGemini(ctx, system, user)
	default:
		return c.completeOpenAI(ctx, system, user)
	}
}

func (c *Client) endpoint(path string) string {
	return strings.TrimRight(c.cfg.effectiveBaseURL(), "/") + path
}

func (c *Client) postJSON(ctx context.Context, url string, headers map[string]string, body any) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("ai: %s returned %d: %s", c.cfg.Kind, resp.StatusCode, snippet(data))
	}
	return data, nil
}

func (c *Client) completeOpenAI(ctx context.Context, system, user string) (string, error) {
	body := map[string]any{
		"model": c.cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	headers := map[string]string{}
	if c.cfg.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.cfg.APIKey
	}
	data, err := c.postJSON(ctx, c.endpoint("/chat/completions"), headers, body)
	if err != nil {
		return "", err
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", ErrEmptyReply
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

func (c *Client) completeAnthropic(ctx context.Context, system, user string) (string, error) {
	body := map[string]any{
		"model":      c.cfg.Model,
		"max_tokens": 1024,
		"system":     system,
		"messages": []map[string]string{
			{"role": "user", "content": user},
		},
	}
	headers := map[string]string{
		"x-api-key":         c.cfg.APIKey,
		"anthropic-version": "2023-06-01",
	}
	data, err := c.postJSON(ctx, c.endpoint("/messages"), headers, body)
	if err != nil {
		return "", err
	}
	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, part := range out.Content {
		b.WriteString(part.Text)
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", ErrEmptyReply
	}
	return text, nil
}

func (c *Client) completeGemini(ctx context.Context, system, user string) (string, error) {
	body := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{{"text": system}},
		},
		"contents": []map[string]any{
			{"role": "user", "parts": []map[string]string{{"text": user}}},
		},
	}
	headers := map[string]string{"x-goog-api-key": c.cfg.APIKey}
	url := c.endpoint("/models/" + c.cfg.Model + ":generateContent")
	data, err := c.postJSON(ctx, url, headers, body)
	if err != nil {
		return "", err
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 {
		return "", ErrEmptyReply
	}
	var b strings.Builder
	for _, part := range out.Candidates[0].Content.Parts {
		b.WriteString(part.Text)
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", ErrEmptyReply
	}
	return text, nil
}

func snippet(data []byte) string {
	s := strings.TrimSpace(string(data))
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
