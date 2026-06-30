package imports

import (
	"strings"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func TestCycle_Go_ExternalTestPackage(t *testing.T) {
	// External test files (package foo_test) importing package foo must NOT
	// create edges among sibling _test.go files. Before the fix, resolveGo
	// resolved "example.com/m" to every .go file in the module root, including
	// sibling _test.go files, creating bogus circular dependencies.
	req := buildReq(t, []fileSpec{
		{"foo.go", analysis.LangGo, "package foo\n"},
		{"a_test.go", analysis.LangGo, "package foo_test\nimport \"example.com/m\"\n"},
		{"b_test.go", analysis.LangGo, "package foo_test\nimport \"example.com/m\"\n"},
		{"c_test.go", analysis.LangGo, "package foo_test\nimport \"example.com/m\"\n"},
	}, map[string]string{"go.mod": "module example.com/m\n\ngo 1.26\n"})

	g := BuildGraph(req)

	// Collect all _test.go node IDs.
	var testIDs []string
	for id := range g.Nodes {
		if strings.HasSuffix(id, "_test.go") {
			testIDs = append(testIDs, id)
		}
	}
	if len(testIDs) != 3 {
		t.Fatalf("expected 3 test nodes, got %d", len(testIDs))
	}

	// Verify no edge exists between any two test files.
	for from, succ := range g.Edges {
		if !strings.HasSuffix(from, "_test.go") {
			continue
		}
		for to := range succ {
			if strings.HasSuffix(to, "_test.go") {
				t.Errorf("test file %q has edge to test file %q — bogus cycle", from, to)
			}
		}
	}

	// Each test file must have exactly one outgoing edge: to foo.go (the only
	// non-test .go file that resolveGo should return).
	for _, id := range testIDs {
		succ := g.Edges[id]
		if len(succ) != 1 {
			t.Errorf("expected exactly 1 outgoing edge from %q, got %d: %v", id, len(succ), succ)
		}
	}
}
