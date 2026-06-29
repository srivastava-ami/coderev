package llm

import (
	"context"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func TestCLIProvider_Echo(t *testing.T) {
	cfg := analysis.LLMConfig{
		Enabled:    true,
		Provider:   "cli",
		CLICommand: "/bin/echo {prompt}",
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := p.Complete(context.Background(), "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if result != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", result)
	}
}

func TestCLIProvider_SpecialChars(t *testing.T) {
	cfg := analysis.LLMConfig{
		Enabled:    true,
		Provider:   "cli",
		CLICommand: "/bin/echo {prompt}",
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	prompt := "hello $(whoami) && rm -rf /"
	result, _, err := p.Complete(context.Background(), prompt)
	if err != nil {
		t.Fatal(err)
	}
	if result != prompt {
		t.Errorf("expected %q, got %q", prompt, result)
	}
}

func TestCLIProvider_EmptyCommand(t *testing.T) {
	cfg := analysis.LLMConfig{
		Enabled:    true,
		Provider:   "cli",
		CLICommand: "",
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for empty cli_command")
	}
}
