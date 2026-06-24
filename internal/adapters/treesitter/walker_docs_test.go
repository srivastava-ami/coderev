package treesitter

import (
	"context"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// ── Documentation ─────────────────────────────────────────────────────────────

func TestCommentedOutCodeViolation(t *testing.T) {
	src := `
function foo(): void {
  // const x = getUser();
  // return x;
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "documentation.no_comment_tombstones") {
		t.Error("expected documentation.no_comment_tombstones for commented-out code")
	}
}

func TestTODOWithoutTicketViolation(t *testing.T) {
	src := `
function foo(): void {
  // TODO: fix this later
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "documentation.todo_format") {
		t.Error("expected documentation.todo_format for TODO without ticket")
	}
}

func TestTODOWithTicketNoViolation(t *testing.T) {
	src := `
function foo(): void {
  // TODO(#1234): fix this after upgrade
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "documentation.todo_format") {
		t.Error("should not flag TODO(#1234) which has a ticket reference")
	}
}

// ── File structure ────────────────────────────────────────────────────────────

func TestFileLengthViolation(t *testing.T) {
	src := ""
	for i := 0; i < 260; i++ {
		src += "const _x = 1;\n"
	}
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "file_structure.file_length") {
		t.Error("expected file_structure.file_length violation for 260-line file")
	}
}

// ── Go language support ───────────────────────────────────────────────────────

func TestGoFunctionComplexity(t *testing.T) {
	src := `package main

func classify(a, b, c int) string {
	if a > 0 {
		if b > 0 {
			return "ab"
		} else if c > 0 {
			return "ac"
		}
	} else if b < 0 {
		return "neg"
	}
	if a == 0 || c == 0 {
		return "zero"
	}
	return "other"
}
`
	findings := findingsForSrc(t, src, analysis.LangGo)
	if !hasRule(findings, "complexity.cyclomatic") {
		t.Error("expected complexity.cyclomatic violation in Go function")
	}
}

// ── Adapter availability ──────────────────────────────────────────────────────

func TestAdapterIsAlwaysAvailable(t *testing.T) {
	a := New(defaultStds())
	if !a.IsAvailable() {
		t.Error("treesitter adapter must always be available (pure Go)")
	}
}

func TestAdapterCapabilitiesNonEmpty(t *testing.T) {
	a := New(defaultStds())
	if len(a.Capabilities()) == 0 {
		t.Error("adapter must declare at least one capability")
	}
}

func TestAdapterRunEmptyFileList(t *testing.T) {
	a := New(defaultStds())
	findings, err := a.Run(context.Background(), analysis.RunRequest{Target: ".", Files: nil})
	if err != nil {
		t.Fatalf("Run with empty file list: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty file list, got %d", len(findings))
	}
}
