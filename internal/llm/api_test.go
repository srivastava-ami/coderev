package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type testRequestBody struct {
	Model    string `json:"model"`
	Messages []struct {
		Content string `json:"content"`
	} `json:"messages"`
}

func TestAPIProvider(t *testing.T) {
	const testKey = "sk-test-key-12345"
	t.Setenv("TEST_LLM_API_KEY", testKey)

	var gotKey string
	var reqBody testRequestBody
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		if v := r.Header.Get("anthropic-version"); v != "2023-06-01" {
			t.Errorf("expected anthropic-version header, got %q", v)
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "response text"},
			},
		})
	}))
	defer srv.Close()

	cfg := analysis.LLMConfig{
		Enabled:    true,
		Provider:   "api",
		APIBaseURL: srv.URL,
		APIKeyEnv:  "TEST_LLM_API_KEY",
		Model:      "claude-3-haiku",
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := p.Complete(context.Background(), "test prompt")
	if err != nil {
		t.Fatal(err)
	}
	if gotKey != testKey {
		t.Errorf("expected x-api-key %q, got %q", testKey, gotKey)
	}
	if result != "response text" {
		t.Errorf("expected %q, got %q", "response text", result)
	}
	if reqBody.Model != "claude-3-haiku" {
		t.Errorf("expected model %q, got %q", "claude-3-haiku", reqBody.Model)
	}
	if len(reqBody.Messages) != 1 || reqBody.Messages[0].Content != "test prompt" {
		t.Errorf("unexpected messages: %+v", reqBody.Messages)
	}
}

func TestAPIProvider_MissingKey(t *testing.T) {
	cfg := analysis.LLMConfig{
		Enabled:    true,
		Provider:   "api",
		APIBaseURL: "http://localhost:0",
		APIKeyEnv:  "NONEXISTENT_TEST_KEY",
		Model:      "claude-3-haiku",
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestAPIProvider_NonOKStatus(t *testing.T) {
	t.Setenv("TEST_LLM_STATUS", "sk-test")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := analysis.LLMConfig{
		Enabled:    true,
		Provider:   "api",
		APIBaseURL: srv.URL,
		APIKeyEnv:  "TEST_LLM_STATUS",
		Model:      "claude-3-haiku",
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}
