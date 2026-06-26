package imports

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// fileSpec is a tiny fixture file: a path relative to the target plus contents.
type fileSpec struct {
	rel     string
	lang    analysis.Language
	content string
}

// buildReq materialises fixtures under a temp dir and returns a RunRequest.
// Files are written to disk (so go.mod/tsconfig sibling lookups work) and also
// carried in-memory as FileInfo, mirroring how the runner supplies them.
func buildReq(t *testing.T, files []fileSpec, extra map[string]string) analysis.RunRequest {
	t.Helper()
	dir := t.TempDir()

	for name, content := range extra {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	var infos []analysis.FileInfo
	for _, f := range files {
		full := filepath.Join(dir, f.rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(f.content), 0o644); err != nil {
			t.Fatalf("write %s: %v", f.rel, err)
		}
		infos = append(infos, analysis.FileInfo{
			Path:     full,
			Language: f.lang,
			Content:  []byte(f.content),
		})
	}
	return analysis.RunRequest{Target: dir, Files: infos}
}

func runAdapter(t *testing.T, req analysis.RunRequest) []analysis.Finding {
	t.Helper()
	findings, err := New().Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return findings
}

func TestAdapter_StaticContract(t *testing.T) {
	a := New()
	if !a.IsAvailable() {
		t.Fatal("native adapter must always be available")
	}
	if a.Name() != "imports" {
		t.Fatalf("Name = %q", a.Name())
	}
	caps := a.Capabilities()
	want := map[string]bool{"file_structure.circular_deps": false, "nx_conventions.boundaries": false}
	for _, c := range caps {
		want[c] = true
	}
	for c, ok := range want {
		if !ok {
			t.Errorf("missing capability %q", c)
		}
	}
}

func TestCycle_TypeScript(t *testing.T) {
	req := buildReq(t, []fileSpec{
		{"a.ts", analysis.LangTypeScript, `import { b } from './b';\nexport const a = () => b();`},
		{"b.ts", analysis.LangTypeScript, `import { a } from './a';\nexport const b = () => a();`},
	}, nil)
	assertOneCycle(t, runAdapter(t, req))
}

func TestCycle_JavaScript(t *testing.T) {
	req := buildReq(t, []fileSpec{
		{"a.js", analysis.LangJavaScript, `const { b } = require('./b');\nmodule.exports.a = () => b();`},
		{"b.js", analysis.LangJavaScript, `const { a } = require('./a');\nmodule.exports.b = () => a();`},
	}, nil)
	assertOneCycle(t, runAdapter(t, req))
}

func TestCycle_Go(t *testing.T) {
	req := buildReq(t, []fileSpec{
		{"a/a.go", analysis.LangGo, "package a\nimport \"example.com/m/b\"\nvar _ = b.B"},
		{"b/b.go", analysis.LangGo, "package b\nimport \"example.com/m/a\"\nvar _ = a.A"},
	}, map[string]string{"go.mod": "module example.com/m\n\ngo 1.26\n"})
	assertOneCycle(t, runAdapter(t, req))
}

func TestAcyclic_NoFindings(t *testing.T) {
	req := buildReq(t, []fileSpec{
		{"a.ts", analysis.LangTypeScript, `import { b } from './b';\nexport const a = () => b();`},
		{"b.ts", analysis.LangTypeScript, `import { c } from './c';\nexport const b = () => c();`},
		{"c.ts", analysis.LangTypeScript, `export const c = () => 42;`},
	}, nil)
	if f := runAdapter(t, req); len(f) != 0 {
		t.Fatalf("acyclic graph produced %d findings: %v", len(f), f)
	}
}

func TestAlias_TsconfigPaths(t *testing.T) {
	tsconfig := `{
  "compilerOptions": {
    // baseUrl + path alias resolution
    "baseUrl": ".",
    "paths": { "@app/*": ["src/*"] }
  }
}`
	req := buildReq(t, []fileSpec{
		{"src/a.ts", analysis.LangTypeScript, `import { b } from '@app/b';\nexport const a = () => b();`},
		{"src/b.ts", analysis.LangTypeScript, `import { a } from '@app/a';\nexport const b = () => a();`},
	}, map[string]string{"tsconfig.json": tsconfig})

	g := BuildGraph(req)
	if len(g.Cycles()) != 1 {
		t.Fatalf("alias imports should form 1 cycle, got %d (edges=%v)", len(g.Cycles()), g.Edges)
	}
}

func TestPackageImport_Excluded(t *testing.T) {
	// External package imports must not appear as graph edges.
	req := buildReq(t, []fileSpec{
		{"a.ts", analysis.LangTypeScript, `import React from 'react';\nimport { b } from './b';\nexport const a = () => b();`},
		{"b.ts", analysis.LangTypeScript, `export const b = () => 1;`},
	}, nil)
	g := BuildGraph(req)
	if len(g.Cycles()) != 0 {
		t.Fatalf("expected no cycle, got %v", g.Cycles())
	}
	// a -> b is the only internal edge; react is dropped.
	total := 0
	for _, succ := range g.Edges {
		total += len(succ)
	}
	if total != 1 {
		t.Fatalf("expected exactly 1 internal edge, got %d (%v)", total, g.Edges)
	}
}

func TestBuildGraph_Reusable(t *testing.T) {
	// Exported graph accessor exposes nodes + edges for downstream consumers.
	req := buildReq(t, []fileSpec{
		{"a.ts", analysis.LangTypeScript, `import './b';`},
		{"b.ts", analysis.LangTypeScript, `export const b = 1;`},
	}, nil)
	g := BuildGraph(req)
	if len(g.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.Nodes))
	}
	found := false
	for from, succ := range g.Edges {
		if strings.HasSuffix(from, "a.ts") {
			for to := range succ {
				if strings.HasSuffix(to, "b.ts") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected a.ts -> b.ts edge, edges=%v", g.Edges)
	}
}

func assertOneCycle(t *testing.T, findings []analysis.Finding) {
	t.Helper()
	if len(findings) != 1 {
		t.Fatalf("expected 1 circular-dependency finding, got %d: %v", len(findings), findings)
	}
	f := findings[0]
	if f.Rule != "file_structure.circular_deps" {
		t.Errorf("Rule = %q", f.Rule)
	}
	if f.Severity != analysis.SeverityBlocker {
		t.Errorf("Severity = %q, want blocker", f.Severity)
	}
	if f.Source != "imports" {
		t.Errorf("Source = %q", f.Source)
	}
	if !strings.Contains(f.Message, "circular dependency") {
		t.Errorf("Message = %q", f.Message)
	}
}
