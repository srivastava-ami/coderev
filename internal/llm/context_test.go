package llm

import (
	"strings"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func TestParseDiff_basic(t *testing.T) {
	raw := []byte("diff --git a/foo/bar.go b/foo/bar.go\n--- a/foo/bar.go\n+++ b/foo/bar.go\n@@ -1,3 +1,4 @@\n package foo\n+// added\n func Bar() {}\n")
	hunks, err := ParseDiff(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(hunks) != 1 {
		t.Fatalf("want 1 hunk, got %d", len(hunks))
	}
	if hunks[0].File != "foo/bar.go" {
		t.Errorf("wrong file: %s", hunks[0].File)
	}
	if hunks[0].Header == "" {
		t.Error("missing header")
	}
}

func TestParseDiff_empty(t *testing.T) {
	hunks, err := ParseDiff([]byte{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hunks) != 0 {
		t.Errorf("want 0 hunks, got %d", len(hunks))
	}
}

func TestGraphNeighborhood_hops(t *testing.T) {
	graphJSON := []byte(`{"directed":false,"nodes":[{"id":"a.go","label":"a","source_file":"a.go"},{"id":"b.go","label":"b","source_file":"b.go"},{"id":"c.go","label":"c","source_file":"c.go"}],"links":[{"source":"a.go","target":"b.go","relation":"calls"},{"source":"b.go","target":"c.go","relation":"calls"}]}`)
	nb, err := GraphNeighborhood(graphJSON, []string{"a.go"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]bool{}
	for _, n := range nb {
		ids[n.ID] = true
	}
	if !ids["a.go"] {
		t.Error("seed node a.go missing")
	}
	if !ids["b.go"] {
		t.Error("1-hop b.go missing")
	}
	if ids["c.go"] {
		t.Error("2-hop c.go must not appear at hops=1")
	}
}

func TestGraphNeighborhood_invalid(t *testing.T) {
	_, err := GraphNeighborhood([]byte("not json"), []string{"x.go"}, 1)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestAssemblePrompt_contains_sections(t *testing.T) {
	ctx := ReviewContext{
		BaseRef: "main",
		Hunks: []DiffHunk{{File: "foo.go", Header: "@@ -1 +1 @@", Content: "+x\n"}},
		Findings: []analysis.Finding{{
			Rule: "complexity.cyclomatic", File: "foo.go", Line: 1, Message: "too complex",
		}},
		Neighbors: []GraphNeighbor{{ID: "foo.go:Bar", File: "foo.go", Label: "Bar"}},
	}
	p := AssemblePrompt(ctx)
	for _, want := range []string{"<graph_context>", "<findings>", "<diff base=", "</diff>"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestAssemblePrompt_empty(t *testing.T) {
	p := AssemblePrompt(ReviewContext{})
	if p == "" {
		t.Error("empty prompt even with no sections")
	}
	if strings.Contains(p, "<diff") {
		t.Error("must not emit diff section when no hunks")
	}
}
