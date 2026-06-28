package llm

import (
	"strings"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func TestNew_OAuth(t *testing.T) {
	_, err := New(analysis.LLMConfig{Provider: "oauth"})
	if err == nil {
		t.Fatal("expected error for oauth provider")
	}
	if !strings.Contains(err.Error(), "claude -p {prompt}") {
		t.Errorf("oauth error should mention the guidance, got: %v", err)
	}
}

func TestNew_Unknown(t *testing.T) {
	_, err := New(analysis.LLMConfig{Provider: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestNew_APIMissingKeyEnv(t *testing.T) {
	_, err := New(analysis.LLMConfig{Provider: "api", APIKeyEnv: ""})
	if err == nil {
		t.Fatal("expected error for api provider without api_key_env")
	}
}
