package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func mustJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	body, _ := io.ReadAll(r.Body)
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	return m
}

func TestOpenAICompatible(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_ = mustJSON(t, r)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello there"}}]}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Kind: OpenAI, BaseURL: srv.URL, Model: "gpt-x", APIKey: "sk-test"})
	out, err := c.Complete(context.Background(), "sys", "hi")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "hello there" {
		t.Errorf("content = %q", out)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestOpenAICompatibleNoKeyOmitsAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Kind: Ollama, BaseURL: srv.URL, Model: "llama"})
	if _, err := c.Complete(context.Background(), "s", "u"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("expected no Authorization header, got %q", gotAuth)
	}
}

func TestAnthropic(t *testing.T) {
	var gotKey, gotVersion, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		gotPath = r.URL.Path
		body := mustJSON(t, r)
		if body["system"] != "sys" {
			t.Errorf("system = %v", body["system"])
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"claude "},{"type":"text","text":"reply"}]}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Kind: Anthropic, BaseURL: srv.URL, Model: "claude-x", APIKey: "ak"})
	out, err := c.Complete(context.Background(), "sys", "hi")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "claude reply" {
		t.Errorf("content = %q", out)
	}
	if gotKey != "ak" || gotVersion != "2023-06-01" {
		t.Errorf("headers key=%q version=%q", gotKey, gotVersion)
	}
	if gotPath != "/messages" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestGemini(t *testing.T) {
	var gotKey, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-goog-api-key")
		gotPath = r.URL.Path
		_ = mustJSON(t, r)
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"gemini reply"}]}}]}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Kind: Gemini, BaseURL: srv.URL, Model: "gemini-x", APIKey: "gk"})
	out, err := c.Complete(context.Background(), "sys", "hi")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "gemini reply" {
		t.Errorf("content = %q", out)
	}
	if gotKey != "gk" {
		t.Errorf("goog key = %q", gotKey)
	}
	if gotPath != "/models/gemini-x:generateContent" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestDisabled(t *testing.T) {
	c := New(Config{Enabled: false, Kind: OpenAI, Model: "x", APIKey: "k"})
	if _, err := c.Complete(context.Background(), "s", "u"); err != ErrDisabled {
		t.Errorf("err = %v, want ErrDisabled", err)
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want error
	}{
		{"unknown kind", Config{Kind: "bogus", Model: "m", APIKey: "k"}, ErrUnknownKind},
		{"no model", Config{Kind: OpenAI, APIKey: "k"}, ErrNoModel},
		{"custom needs base url", Config{Kind: Custom, Model: "m"}, ErrNoBaseURL},
		{"openai needs key", Config{Kind: OpenAI, Model: "m"}, ErrNoAPIKey},
		{"ollama needs no key", Config{Kind: Ollama, Model: "m"}, nil},
		{"custom ok with base url", Config{Kind: Custom, Model: "m", BaseURL: "http://x"}, nil},
	}
	for _, tc := range cases {
		if got := tc.cfg.Validate(); got != tc.want {
			t.Errorf("%s: Validate = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestErrorStatusPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"bad key"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Kind: OpenAI, BaseURL: srv.URL, Model: "m", APIKey: "k"})
	_, err := c.Complete(context.Background(), "s", "u")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("err = %v, want a 401", err)
	}
}

func TestExplainConflictSendsDetails(t *testing.T) {
	var userMsg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := mustJSON(t, r)
		msgs := body["messages"].([]any)
		userMsg = msgs[len(msgs)-1].(map[string]any)["content"].(string)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"keep the newer one"}}]}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Kind: OpenAI, BaseURL: srv.URL, Model: "m", APIKey: "k"})
	out, err := c.ExplainConflict(context.Background(), ConflictDetails{
		Folder:         "photos",
		Path:           "trip/img.jpg",
		LocalDevice:    "AAAAAAAAAAAAAAAA",
		RemoteDevice:   "BBBBBBBBBBBBBBBB",
		LocalModified:  time.Unix(1_700_000_000, 0),
		RemoteModified: time.Unix(1_700_100_000, 0),
	})
	if err != nil {
		t.Fatalf("ExplainConflict: %v", err)
	}
	if out != "keep the newer one" {
		t.Errorf("out = %q", out)
	}
	if !strings.Contains(userMsg, "trip/img.jpg") || !strings.Contains(userMsg, "photos") {
		t.Errorf("prompt missing details: %q", userMsg)
	}
}

func TestAnalyzeLogsTruncates(t *testing.T) {
	var userMsg string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := mustJSON(t, r)
		msgs := body["messages"].([]any)
		userMsg = msgs[len(msgs)-1].(map[string]any)["content"].(string)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"looks healthy"}}]}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Kind: OpenAI, BaseURL: srv.URL, Model: "m", APIKey: "k"})
	big := strings.Repeat("x", 20000)
	if _, err := c.AnalyzeLogs(context.Background(), big); err != nil {
		t.Fatalf("AnalyzeLogs: %v", err)
	}
	if len(userMsg) > 12100 {
		t.Errorf("logs not truncated: %d chars", len(userMsg))
	}
}

func TestTestConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	c := New(Config{Enabled: true, Kind: OpenAI, BaseURL: srv.URL, Model: "m", APIKey: "k"})
	if err := c.TestConnection(context.Background()); err != nil {
		t.Errorf("TestConnection: %v", err)
	}
}
