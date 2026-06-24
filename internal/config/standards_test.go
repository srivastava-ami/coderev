package config

import (
	"os"
	"testing"
)

const minimalStandards = `
[meta]
version      = "2.0.0"
last_updated = "2026-06-23"

[complexity.cyclomatic]
max_value     = 8
advisory_at   = 5
hard_block_at = 12

[complexity.function_length]
max_lines    = 30
advisory_at  = 20

[complexity.parameter_count]
max_count = 3

[complexity.nesting]
max_depth = 2

[file_structure.file_length]
max_lines   = 250
advisory_at = 150

[file_structure.class_length]
max_lines = 120

[security.secrets]
rule = "no_hardcoded_secrets"
patterns = ["password\\s*=\\s*['\"][^'\"]+['\"]"]
`

func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "standards-*.toml")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestLoadParsesVersion(t *testing.T) {
	path := writeTempTOML(t, minimalStandards)
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Meta.Version != "2.0.0" {
		t.Errorf("version = %q, want 2.0.0", s.Meta.Version)
	}
}

func TestLoadParsesComplexityThresholds(t *testing.T) {
	path := writeTempTOML(t, minimalStandards)
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Complexity.Cyclomatic.MaxValue != 8 {
		t.Errorf("cyclomatic max_value = %d, want 8", s.Complexity.Cyclomatic.MaxValue)
	}
	if s.Complexity.Cyclomatic.HardBlockAt != 12 {
		t.Errorf("cyclomatic hard_block_at = %d, want 12", s.Complexity.Cyclomatic.HardBlockAt)
	}
	if s.Complexity.Function.MaxLines != 30 {
		t.Errorf("function max_lines = %d, want 30", s.Complexity.Function.MaxLines)
	}
	if s.Complexity.Parameters.MaxCount != 3 {
		t.Errorf("parameter max_count = %d, want 3", s.Complexity.Parameters.MaxCount)
	}
	if s.Complexity.Nesting.MaxDepth != 2 {
		t.Errorf("nesting max_depth = %d, want 2", s.Complexity.Nesting.MaxDepth)
	}
}

func TestLoadParsesFileStructure(t *testing.T) {
	path := writeTempTOML(t, minimalStandards)
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.FileStructure.FileLength.MaxLines != 250 {
		t.Errorf("file_length max_lines = %d, want 250", s.FileStructure.FileLength.MaxLines)
	}
}

func TestLoadParsesSecurityPatterns(t *testing.T) {
	path := writeTempTOML(t, minimalStandards)
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(s.Security.Secrets.Patterns) == 0 {
		t.Error("expected at least one secret pattern")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/does/not/exist.toml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	path := writeTempTOML(t, "this is not valid toml ::::")
	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid TOML, got nil")
	}
}

