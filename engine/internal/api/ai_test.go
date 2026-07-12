package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mockOpenAI(t *testing.T, reply string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + reply + `"}}]}`))
	}))
}

func TestAIConfigMasksKey(t *testing.T) {
	s := newTestServer(t)
	body := `{"enabled":true,"kind":"openai","base_url":"http://example.invalid","model":"gpt","api_key":"sk-secret"}`
	if rec := do(t, s, "PUT", "/ai", body, testToken); rec.Code != http.StatusOK {
		t.Fatalf("PUT /ai code = %d (%s)", rec.Code, rec.Body)
	}
	rec := do(t, s, "GET", "/ai", "", testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /ai code = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "sk-secret") {
		t.Error("GET /ai leaked the API key")
	}
	var v aiConfigView
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !v.HasKey || !v.Enabled || v.Model != "gpt" {
		t.Errorf("view = %+v", v)
	}
}

func TestAIKeyCarriesOver(t *testing.T) {
	s := newTestServer(t)
	do(t, s, "PUT", "/ai", `{"enabled":true,"kind":"openai","model":"gpt","api_key":"sk-first"}`, testToken)
	do(t, s, "PUT", "/ai", `{"enabled":true,"kind":"openai","model":"gpt-2","api_key":""}`, testToken)
	if got := s.aiConfig().APIKey; got != "sk-first" {
		t.Errorf("key not carried over: %q", got)
	}
	if got := s.aiConfig().Model; got != "gpt-2" {
		t.Errorf("model not updated: %q", got)
	}
}

func TestAITestAndExplain(t *testing.T) {
	srv := mockOpenAI(t, "keep the newer copy")
	defer srv.Close()
	s := newTestServer(t)
	cfg := `{"enabled":true,"kind":"openai","base_url":"` + srv.URL + `","model":"gpt","api_key":"k"}`
	if rec := do(t, s, "PUT", "/ai", cfg, testToken); rec.Code != http.StatusOK {
		t.Fatalf("PUT /ai code = %d (%s)", rec.Code, rec.Body)
	}
	if rec := do(t, s, "POST", "/ai/test", cfg, testToken); rec.Code != http.StatusOK {
		t.Fatalf("POST /ai/test code = %d (%s)", rec.Code, rec.Body)
	}
	rec := do(t, s, "POST", "/ai/explain-conflict", `{"folder":"photos","path":"a.jpg"}`, testToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("explain code = %d (%s)", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), "keep the newer copy") {
		t.Errorf("explain body = %s", rec.Body)
	}
}

func TestAIExplainDisabledReturns412(t *testing.T) {
	s := newTestServer(t)
	rec := do(t, s, "POST", "/ai/explain-conflict", `{"folder":"x","path":"y"}`, testToken)
	if rec.Code != http.StatusPreconditionFailed {
		t.Errorf("code = %d, want 412", rec.Code)
	}
}

func TestAIAnalyzeLogs(t *testing.T) {
	srv := mockOpenAI(t, "no errors found")
	defer srv.Close()
	s := newTestServer(t)
	do(t, s, "PUT", "/ai", `{"enabled":true,"kind":"openai","base_url":"`+srv.URL+`","model":"gpt","api_key":"k"}`, testToken)
	rec := do(t, s, "POST", "/ai/analyze-logs", `{"logs":"line1\nline2"}`, testToken)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "no errors found") {
		t.Errorf("analyze code=%d body=%s", rec.Code, rec.Body)
	}
}
