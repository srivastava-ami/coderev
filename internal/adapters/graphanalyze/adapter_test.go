package graphanalyze

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

const testModule = "mytest"

func TestAdapter_Metadata(t *testing.T) {
	a := New(analysis.GraphAnalyzeConfig{})
	if a.Name() != "graphanalyze" {
		t.Errorf("Name() = %q, want graphanalyze", a.Name())
	}
	if !a.IsAvailable() {
		t.Error("IsAvailable() = false, want true")
	}
	want := []string{"architecture.coupling", "architecture.hotspot"}
	caps := a.Capabilities()
	if len(caps) != len(want) {
		t.Fatalf("Capabilities() = %v, want %v", caps, want)
	}
	for i := range want {
		if caps[i] != want[i] {
			t.Errorf("Capabilities()[%d] = %q, want %q", i, caps[i], want[i])
		}
	}
}

func TestAdapter_Run_AboveThreshold(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir)
	writeMain(t, dir, 25)
	for i := range 25 {
		writeSubPkg(t, dir, i)
	}

	cfg := analysis.GraphAnalyzeConfig{
		Enabled:   true,
		FanInMax:  20,
		FanOutMax: 20,
		Rules:     []string{"architecture.coupling", "architecture.hotspot"},
	}
	a := New(cfg)
	findings, err := a.Run(context.Background(), analysis.RunRequest{Target: dir})
	if err != nil {
		t.Fatal(err)
	}

	var coupling, hotspot int
	for _, f := range findings {
		if f.Source != "graphanalyze" {
			t.Errorf("Source = %q, want graphanalyze", f.Source)
		}
		if f.Pillar != "architecture" {
			t.Errorf("Pillar = %q, want architecture", f.Pillar)
		}
		if f.Severity != analysis.SeverityAdvisory {
			t.Errorf("Severity = %v, want advisory", f.Severity)
		}
		if f.File == "" {
			t.Error("File is empty")
		}
		if f.Remediation == "" {
			t.Error("Remediation is empty")
		}
		switch f.Rule {
		case "architecture.coupling":
			coupling++
			if !strings.Contains(f.Message, "fan-out") {
				t.Errorf("coupling message missing fan-out: %s", f.Message)
			}
		case "architecture.hotspot":
			hotspot++
			if !strings.Contains(f.Message, "hotspot") {
				t.Errorf("hotspot message missing hotspot: %s", f.Message)
			}
		}
	}
	if coupling == 0 {
		t.Error("expected architecture.coupling finding")
	}
	if hotspot == 0 {
		t.Error("expected architecture.hotspot finding")
	}
}

func TestAdapter_Run_BelowThreshold(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir)
	writeMain(t, dir, 5)
	for i := range 5 {
		writeSubPkg(t, dir, i)
	}

	cfg := analysis.GraphAnalyzeConfig{
		Enabled:   true,
		FanInMax:  20,
		FanOutMax: 20,
		Rules:     []string{"architecture.coupling", "architecture.hotspot"},
	}
	a := New(cfg)
	findings, err := a.Run(context.Background(), analysis.RunRequest{Target: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range findings {
		if f.Rule == "architecture.coupling" || f.Rule == "architecture.hotspot" {
			t.Errorf("unexpected finding: %s: %s", f.Rule, f.Message)
		}
	}
}

func TestAdapter_Run_GraphBuildError(t *testing.T) {
	cfg := analysis.GraphAnalyzeConfig{
		Enabled: true,
		Rules:   []string{"architecture.coupling"},
	}
	a := New(cfg)
	findings, err := a.Run(context.Background(), analysis.RunRequest{Target: "/nonexistent/path/xyz789"})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for bad target, got %d", len(findings))
	}
}

func TestAdapter_Run_EmptyTarget(t *testing.T) {
	dir := t.TempDir()
	cfg := analysis.GraphAnalyzeConfig{
		Enabled:   true,
		FanInMax:  20,
		FanOutMax: 20,
		Rules:     []string{"architecture.coupling", "architecture.hotspot"},
	}
	a := New(cfg)
	findings, err := a.Run(context.Background(), analysis.RunRequest{Target: dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for empty target, got %d", len(findings))
	}
}

// Helpers

func writeGoMod(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+testModule+"\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeMain(t *testing.T, dir string, n int) {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("package main\n\n")
	for i := range n {
		fmt.Fprintf(&sb, "import \"%s/pkg%d\"\n", testModule, i)
	}
	sb.WriteString("\nfunc main() {}\n")
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeSubPkg(t *testing.T, dir string, i int) {
	t.Helper()
	pkgDir := filepath.Join(dir, fmt.Sprintf("pkg%d", i))
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	src := fmt.Sprintf("package pkg%d\n\nimport \"%s\"\n", i, testModule)
	if err := os.WriteFile(filepath.Join(pkgDir, fmt.Sprintf("pkg%d.go", i)), []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
}
