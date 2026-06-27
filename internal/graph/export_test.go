package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExportHTMLOfflineAndDeterministic locks the two guarantees the generated
// graph.html must always hold: it is fully self-contained (no network/CDN/library
// reference) and byte-for-byte deterministic for a given graph.
func TestExportHTMLOfflineAndDeterministic(t *testing.T) {
	g := sampleGraph()

	d1 := t.TempDir()
	d2 := t.TempDir()
	if err := ExportGraphHTML(g, d1); err != nil {
		t.Fatal(err)
	}
	if err := ExportGraphHTML(g, d2); err != nil {
		t.Fatal(err)
	}
	h1, err := os.ReadFile(filepath.Join(d1, "graph.html"))
	if err != nil {
		t.Fatal(err)
	}
	h2, err := os.ReadFile(filepath.Join(d2, "graph.html"))
	if err != nil {
		t.Fatal(err)
	}

	if string(h1) != string(h2) {
		t.Error("graph.html is not deterministic across runs")
	}

	// No external dependency of any kind may leak into the artifact.
	for _, bad := range []string{"http://", "https://", "d3js", "cdn", "unpkg", "jsdelivr", "<script src", "<link "} {
		if strings.Contains(string(h1), bad) {
			t.Errorf("graph.html contains forbidden external reference %q — must be fully offline", bad)
		}
	}
}
