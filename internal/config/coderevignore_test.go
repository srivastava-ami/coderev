package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIgnoreList_Matches_glob(t *testing.T) {
	il := IgnoreList{patterns: []string{"*.md"}}
	for _, p := range []string{"README.md", "a/b/c.md", "docs/foo.md"} {
		if !il.Matches(p) {
			t.Errorf("expected match for %q", p)
		}
	}
	if il.Matches("main.go") {
		t.Error("unexpected match for main.go")
	}
}

func TestIgnoreList_Matches_docs(t *testing.T) {
	il := IgnoreList{patterns: []string{"docs/"}}
	for _, p := range []string{"docs/foo.go", "pkg/docs/bar.go"} {
		if !il.Matches(p) {
			t.Errorf("expected match for %q", p)
		}
	}
	if il.Matches("documentation.go") {
		t.Error("unexpected match for documentation.go")
	}
}

func TestIgnoreList_Matches_prefix(t *testing.T) {
	il := IgnoreList{patterns: []string{"CHANGELOG*"}}
	for _, p := range []string{"CHANGELOG", "CHANGELOG.md", "CHANGELOG-2024.txt"} {
		if !il.Matches(p) {
			t.Errorf("expected match for %q", p)
		}
	}
	if il.Matches("log.go") {
		t.Error("unexpected match for log.go")
	}
}

func TestIgnoreList_Empty(t *testing.T) {
	il := IgnoreList{}
	if il.Matches("anything.md") {
		t.Error("empty IgnoreList should never match")
	}
}

func TestParseIgnoreList_SkipsComments(t *testing.T) {
	content := "# comment\n\n*.md\n# another\n*.txt\n"
	patterns := parseIgnoreList(content)
	if len(patterns) != 2 {
		t.Fatalf("got %d patterns, want 2: %v", len(patterns), patterns)
	}
	if patterns[0] != "*.md" || patterns[1] != "*.txt" {
		t.Errorf("unexpected patterns: %v", patterns)
	}
}

func TestWriteDefaultIgnoreList_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".coderev", ".coderevignore")

	if err := WriteDefaultIgnoreList(path); err != nil {
		t.Fatal(err)
	}
	data1, _ := os.ReadFile(path)

	if err := WriteDefaultIgnoreList(path); err != nil {
		t.Fatal(err)
	}
	data2, _ := os.ReadFile(path)

	if string(data1) != string(data2) {
		t.Error("second write changed file content")
	}
}
