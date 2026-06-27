package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuild_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	g, err := Build(dir)
	if err != nil {
		t.Fatalf("Build(empty): %v", err)
	}
	if len(g.Nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(g.Nodes))
	}
}

func TestBuild_SingleGoFile(t *testing.T) {
	dir := t.TempDir()
	src := "package foo\n\nimport \"fmt\"\n\nfunc Hello() string {\n\treturn fmt.Sprintf(\"hello\")\n}\n\ntype Config struct {\n\tName string\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "foo.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(g.Nodes) == 0 {
		t.Fatal("expected at least 1 node")
	}

	var fileNode bool
	for _, n := range g.Nodes {
		if n.Kind == KindFile {
			fileNode = true
			if n.Label != "foo.go" {
				t.Errorf("file node label = %q, want foo.go", n.Label)
			}
		}
	}
	if !fileNode {
		t.Fatal("expected a file node")
	}

	var foundHello bool
	for _, n := range g.Nodes {
		if n.Kind == KindFunction && n.Label == "Hello" {
			foundHello = true
		}
	}
	if !foundHello {
		t.Fatal("expected function node 'Hello'")
	}

	var foundConfig bool
	for _, n := range g.Nodes {
		if n.Kind == KindType && n.Label == "Config" {
			foundConfig = true
		}
	}
	if !foundConfig {
		t.Fatal("expected type node 'Config'")
	}

	ComputeMetrics(g)

	fileID := cleanPath(filepath.Join(dir, "foo.go"))
	if g.FanOut[fileID] < 1 {
		t.Errorf("file fan-out = %d, want >= 1", g.FanOut[fileID])
	}
}

func TestBuild_ImportsEdge(t *testing.T) {
	dir := t.TempDir()

	aSrc := "package a\n\nimport \"example.com/foo/b\"\n\nfunc UseB() { b.B() }\n"
	bSrc := "package b\n\nfunc B() string { return \"b\" }\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "b"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "a.go"), []byte(aSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b", "b.go"), []byte(bSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var hasImport bool
	for _, e := range g.Edges {
		if e.Relation == "imports" {
			hasImport = true
			break
		}
	}
	if !hasImport {
		t.Fatal("expected at least one imports edge between a.go and b.go")
	}
}

func TestBuild_CallsEdge(t *testing.T) {
	dir := t.TempDir()
	src := "package foo\n\nfunc Caller() string { return Callee() }\nfunc Callee() string { return \"ok\" }\n"
	if err := os.WriteFile(filepath.Join(dir, "foo.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := Build(dir)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	var callsFound int
	for _, e := range g.Edges {
		if e.Relation == "calls" && strings.HasSuffix(e.Source, "Caller") && strings.HasSuffix(e.Target, "Callee") {
			callsFound++
		}
	}
	if callsFound == 0 {
		t.Fatal("expected calls edge from Caller to Callee")
	}
}
