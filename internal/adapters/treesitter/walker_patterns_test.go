package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// ── Type safety ───────────────────────────────────────────────────────────────

func TestAnyTypeViolation(t *testing.T) {
	src := `
function parse(data: any): string {
  return data as any;
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "type_safety.no_any") {
		t.Error("expected type_safety.no_any violation")
	}
}

func TestNoAnyViolationOnCleanCode(t *testing.T) {
	src := `
function parse(data: unknown): string {
  if (typeof data === "string") return data;
  return "";
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "type_safety.no_any") {
		t.Error("should not flag clean unknown-typed function")
	}
}

// ── Observability ─────────────────────────────────────────────────────────────

func TestConsolLogViolation(t *testing.T) {
	src := `
function save(data: string): void {
  console.log("saving", data);
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "observability.logging") {
		t.Error("expected observability.logging violation for console.log")
	}
}

func TestConsoleLogInCommentNoViolation(t *testing.T) {
	src := `
// console.log("this is just a comment")
function clean(): void {
  return;
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "observability.logging") {
		t.Error("should not flag console.log inside a comment")
	}
}

// ── Hardcoding ────────────────────────────────────────────────────────────────

func TestHardcodedURLViolation(t *testing.T) {
	src := `
const BASE = "https://api.production.example.com/v1";
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "hardcoding.urls_and_paths") {
		t.Error("expected hardcoding.urls_and_paths for literal production URL")
	}
}

func TestLocalhostURLNoViolation(t *testing.T) {
	src := `
const BASE = "http://localhost:3000";
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "hardcoding.urls_and_paths") {
		t.Error("should not flag localhost URL")
	}
}
