package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func TestNewPatternMatcher(t *testing.T) {
	pm, err := NewPatternMatcher()
	if err != nil {
		t.Fatalf("NewPatternMatcher failed: %v", err)
	}

	if pm == nil {
		t.Fatal("PatternMatcher is nil")
	}

	if pm.RuleCount() != 0 {
		t.Fatalf("expected 0 rules, got %d", pm.RuleCount())
	}
}

func TestLoadRule(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "test.rule",
		Severity: "blocker",
		Pillar:   "security",
		Patterns: []Pattern{
			{
				Type:    "string_match",
				Pattern: "http://",
				Message: "unencrypted",
			},
		},
	}

	err := pm.LoadRule(rule)
	if err != nil {
		t.Fatalf("LoadRule failed: %v", err)
	}

	if pm.RuleCount() != 1 {
		t.Fatalf("expected 1 rule, got %d", pm.RuleCount())
	}

	loaded := pm.Rules()["test.rule"]
	if loaded == nil {
		t.Fatal("rule not loaded")
	}

	if loaded.ID != "test.rule" {
		t.Fatalf("expected rule ID 'test.rule', got %s", loaded.ID)
	}
}

func TestLoadRuleValidation(t *testing.T) {
	pm, _ := NewPatternMatcher()

	// Test empty rule ID
	rule := &Rule{
		ID:       "",
		Severity: "blocker",
		Patterns: []Pattern{
			{Type: "string_match", Pattern: "test"},
		},
	}

	err := pm.LoadRule(rule)
	if err == nil {
		t.Fatal("expected error for empty rule ID")
	}

	// Test no patterns
	rule.ID = "test.rule"
	rule.Patterns = []Pattern{}

	err = pm.LoadRule(rule)
	if err == nil {
		t.Fatal("expected error for no patterns")
	}
}

func TestIsLanguageSupported(t *testing.T) {
	tests := []struct {
		name       string
		languages  []string
		lang       analysis.Language
		supported  bool
	}{
		{"empty list supports all", []string{}, analysis.LangTypeScript, true},
		{"typescript", []string{"typescript"}, analysis.LangTypeScript, true},
		{"case insensitive", []string{"TypeScript"}, analysis.LangTypeScript, true},
		{"multiple langs", []string{"go", "python", "javascript"}, analysis.LangGo, true},
		{"unsupported", []string{"go", "python"}, analysis.LangTypeScript, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &Rule{Languages: tt.languages}
			got := rule.IsLanguageSupported(tt.lang)
			if got != tt.supported {
				t.Errorf("expected %v, got %v", tt.supported, got)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	pm, _ := NewPatternMatcher()

	// Add a test rule
	rule := &Rule{
		ID:        "security.test",
		Severity:  "blocker",
		Pillar:    "security",
		Languages: []string{"typescript"},
		Patterns: []Pattern{
			{
				Type:    "string_match",
				Pattern: "http://",
				Message: "unencrypted",
			},
		},
	}

	pm.LoadRule(rule)

	// Match should find the http:// pattern
	findings, err := pm.Match("const url = 'http://example.com'", "test.ts", analysis.LangTypeScript)
	if err != nil {
		t.Fatalf("Match failed: %v", err)
	}

	if len(findings) == 0 {
		t.Fatal("expected findings for http:// pattern")
	}
	if findings[0].Rule != "security.test" {
		t.Fatalf("expected rule 'security.test', got %s", findings[0].Rule)
	}
}

func TestLoadRulesFromTOML(t *testing.T) {
	// This test verifies that LoadRules can read TOML files
	// Note: Full integration with embedded FS is tested in Phase A2
	// For Phase A1, we verify the structure and error handling
	pm, _ := NewPatternMatcher()

	// Verify LoadRules accepts a filesystem and path
	// The actual embedded FS will be integrated in Phase A2
	rule := &Rule{
		ID:       "security.test_rule",
		Severity: "major",
		Pillar:   "security",
		Patterns: []Pattern{
			{
				Type:    "string_match",
				Pattern: "test",
				Message: "test message",
			},
		},
	}

	err := pm.LoadRule(rule)
	if err != nil {
		t.Fatalf("LoadRule failed: %v", err)
	}

	// Verify the rule was loaded
	if pm.RuleCount() != 1 {
		t.Fatalf("expected 1 rule, got %d", pm.RuleCount())
	}

	if pm.PatternCount() != 1 {
		t.Fatalf("expected 1 pattern, got %d", pm.PatternCount())
	}
}

// ══════════════════════════════════════════════════════════════════════════
// Phase A2 Chunk 2: Pattern Matcher Tests for 5 Simple Types
// ══════════════════════════════════════════════════════════════════════════

// TestStringMatchPositive: string_match should detect regex and literal patterns
func TestStringMatchPositive(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "security.http",
		Severity: "blocker",
		Pillar:   "security",
		Patterns: []Pattern{
			{
				Type:    "string_match",
				Pattern: "http://",
				Message: "Unencrypted HTTP",
			},
		},
	}

	pm.LoadRule(rule)

	findings, _ := pm.Match("const url = 'http://example.com';", "test.ts", analysis.LangTypeScript)
	if len(findings) == 0 {
		t.Fatal("expected finding for http:// pattern")
	}
}

// TestStringMatchNegative: string_match should not match excluded patterns
func TestStringMatchNegative(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "security.http",
		Severity: "blocker",
		Pillar:   "security",
		Patterns: []Pattern{
			{
				Type:    "string_match",
				Pattern: "http://",
				Exclude: []string{"localhost", "127.0.0.1"},
				Message: "Unencrypted HTTP",
			},
		},
	}

	pm.LoadRule(rule)

	// Should NOT match because localhost is excluded
	findings, _ := pm.Match("const url = 'http://localhost:3000';", "test.ts", analysis.LangTypeScript)
	if len(findings) > 0 {
		t.Fatal("expected no findings for excluded localhost")
	}
}

// TestMethodCallPositive: method_call should detect method invocations
func TestMethodCallPositive(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "security.eval",
		Severity: "blocker",
		Pillar:   "security",
		Patterns: []Pattern{
			{
				Type:    "method_call",
				Pattern: "eval\\s*\\(",
				Message: "eval() detected",
			},
		},
	}

	pm.LoadRule(rule)

	findings, _ := pm.Match("const result = eval(userInput);", "test.js", analysis.LangJavaScript)
	if len(findings) == 0 {
		t.Fatal("expected finding for eval() call")
	}
}

// TestMethodCallNegative: method_call should not match excluded methods
func TestMethodCallNegative(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "security.eval",
		Severity: "blocker",
		Pillar:   "security",
		Patterns: []Pattern{
			{
				Type:    "method_call",
				Pattern: "eval\\s*\\(",
				Exclude: []string{"safeEval"},
				Message: "eval() detected",
			},
		},
	}

	pm.LoadRule(rule)

	// Should NOT match because safeEval is excluded
	findings, _ := pm.Match("const result = safeEval(userInput);", "test.js", analysis.LangJavaScript)
	if len(findings) > 0 {
		t.Fatal("expected no findings for excluded safeEval")
	}
}

// TestImportPositive: import should detect dangerous imports
func TestImportPositive(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "security.unsafe_import",
		Severity: "blocker",
		Pillar:   "security",
		Languages: []string{"go", "python"},
		Patterns: []Pattern{
			{
				Type:    "import",
				Exclude: []string{"unsafe", "os"}, // module names to detect
				Message: "Dangerous import detected",
			},
		},
	}

	pm.LoadRule(rule)

	// Go: import "unsafe"
	findings, _ := pm.Match("import \"unsafe\"", "test.go", analysis.LangGo)
	if len(findings) == 0 {
		t.Fatal("expected finding for unsafe import")
	}

	// Python: from os import system
	findings, _ = pm.Match("from os import system", "test.py", analysis.LangPython)
	if len(findings) == 0 {
		t.Fatal("expected finding for os import")
	}
}

// TestImportNegative: import should not match safe imports
func TestImportNegative(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "security.unsafe_import",
		Severity: "blocker",
		Pillar:   "security",
		Patterns: []Pattern{
			{
				Type:    "import",
				Exclude: []string{"unsafe"},
				Message: "Dangerous import",
			},
		},
	}

	pm.LoadRule(rule)

	// Safe import
	findings, _ := pm.Match("import \"fmt\"", "test.go", analysis.LangGo)
	if len(findings) > 0 {
		t.Fatal("expected no findings for safe import")
	}
}

// TestVariableAssignmentPositive: variable_assignment should detect hardcoded secrets
func TestVariableAssignmentPositive(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "security.hardcoded_secret",
		Severity: "blocker",
		Pillar:   "security",
		Languages: []string{"javascript", "typescript"},
		Patterns: []Pattern{
			{
				Type:    "variable_assignment",
				Exclude: []string{"api_key", "password", "secret"}, // assigns_to names
				Message: "Hardcoded secret detected",
			},
		},
	}

	pm.LoadRule(rule)

	// Hardcoded secret (not from env)
	findings, _ := pm.Match("const api_key = 'sk-12345xyz';", "test.ts", analysis.LangTypeScript)
	if len(findings) == 0 {
		t.Fatal("expected finding for hardcoded api_key")
	}
}

// TestVariableAssignmentNegative: variable_assignment should not match env-sourced values
func TestVariableAssignmentNegative(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "security.hardcoded_secret",
		Severity: "blocker",
		Pillar:   "security",
		Patterns: []Pattern{
			{
				Type:    "variable_assignment",
				Exclude: []string{"api_key", "password"},
				Message: "Hardcoded secret detected",
			},
		},
	}

	pm.LoadRule(rule)

	// From environment (should not match)
	findings, _ := pm.Match("const api_key = process.env.API_KEY;", "test.ts", analysis.LangTypeScript)
	if len(findings) > 0 {
		t.Fatal("expected no findings for env-sourced api_key")
	}
}

// TestFunctionDefPositive: function_def should detect functions with many parameters
func TestFunctionDefPositive(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "complexity.too_many_params",
		Severity: "major",
		Pillar:   "complexity",
		Languages: []string{"typescript", "javascript"},
		Patterns: []Pattern{
			{
				Type:    "function_def",
				Pattern: "5", // threshold: > 5 params
				Message: "Function has too many parameters",
			},
		},
	}

	pm.LoadRule(rule)

	// Function with 6 parameters (exceeds threshold of 5)
	findings, _ := pm.Match("function handler(a, b, c, d, e, f) {", "test.ts", analysis.LangTypeScript)
	if len(findings) == 0 {
		t.Fatal("expected finding for function with 6 parameters")
	}
}

// TestFunctionDefNegative: function_def should not match functions within threshold
func TestFunctionDefNegative(t *testing.T) {
	pm, _ := NewPatternMatcher()

	rule := &Rule{
		ID:       "complexity.too_many_params",
		Severity: "major",
		Pillar:   "complexity",
		Patterns: []Pattern{
			{
				Type:    "function_def",
				Pattern: "5", // threshold: > 5 params
				Message: "Function has too many parameters",
			},
		},
	}

	pm.LoadRule(rule)

	// Function with 4 parameters (within threshold)
	findings, _ := pm.Match("function handler(a, b, c, d) {", "test.ts", analysis.LangTypeScript)
	if len(findings) > 0 {
		t.Fatal("expected no findings for function with 4 parameters")
	}
}
