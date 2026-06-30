package depcve

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func writeJSONFile(t *testing.T, path string, v interface{}) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterContract(t *testing.T) {
	a := New("")
	if a.Name() != "depcve" {
		t.Errorf("Name() = %q, want depcve", a.Name())
	}
	if !a.IsAvailable() {
		t.Error("IsAvailable() must be true for the native adapter")
	}
	caps := a.Capabilities()
	if len(caps) != 1 || caps[0] != "security.dependencies" {
		t.Errorf("Capabilities() = %v, want [security.dependencies]", caps)
	}
	var _ analysis.ToolAdapter = New("")
}

func TestDetectsCVENpm(t *testing.T) {
	dir := t.TempDir()

	lockfile := filepath.Join(dir, "package-lock.json")
	writeJSONFile(t, lockfile, map[string]interface{}{
		"dependencies": map[string]interface{}{
			"lodash": map[string]interface{}{
				"version": "4.17.19",
			},
		},
	})

	snapDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		t.Fatal(err)
	}
	seedVulns := []Vuln{
		{
			ID:      "CVE-2021-23337",
			Summary: "Lodash before 4.17.21 vulnerable to Command Injection",
			Aliases: []string{"GHSA-35jh-r3h4-6jhm"},
			Affected: []struct {
				Package struct {
					Name      string `json:"name"`
					Ecosystem string `json:"ecosystem"`
				} `json:"package"`
				Ranges []struct {
					Type   string `json:"type"`
					Events []struct {
						Introduced string `json:"introduced,omitempty"`
						Fixed      string `json:"fixed,omitempty"`
					} `json:"events"`
				} `json:"ranges"`
			}{
				{
					Package: struct {
						Name      string `json:"name"`
						Ecosystem string `json:"ecosystem"`
					}{Name: "lodash", Ecosystem: "npm"},
					Ranges: []struct {
						Type   string `json:"type"`
						Events []struct {
							Introduced string `json:"introduced,omitempty"`
							Fixed      string `json:"fixed,omitempty"`
						} `json:"events"`
					}{
						{
							Type: "ECOSYSTEM",
							Events: []struct {
								Introduced string `json:"introduced,omitempty"`
								Fixed      string `json:"fixed,omitempty"`
							}{
								{Introduced: "0"},
								{Fixed: "4.17.21"},
							},
						},
					},
				},
			},
		},
	}
	if err := writeGzippedJSON(filepath.Join(snapDir, "osv-snapshot.json.gz"), seedVulns); err != nil {
		t.Fatal(err)
	}

	a := New("")
	req := analysis.RunRequest{Target: dir}
	findings, err := a.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Rule != "security.dependencies" {
		t.Errorf("Rule = %q, want security.dependencies", f.Rule)
	}
	if f.Pillar != "security" {
		t.Errorf("Pillar = %q, want security", f.Pillar)
	}
	if f.Severity != analysis.SeverityBlocker {
		t.Errorf("Severity = %q, want blocker", f.Severity)
	}
	if f.Source != "depcve" {
		t.Errorf("Source = %q, want depcve", f.Source)
	}
	if !strings.Contains(f.Message, "lodash") || !strings.Contains(f.Message, "CVE-2021-23337") {
		t.Errorf("Message = %q, want mention of lodash and CVE-2021-23337", f.Message)
	}
	if f.File != lockfile {
		t.Errorf("File = %q, want %q", f.File, lockfile)
	}
}

func TestSkipsGracefullyWithoutSnapshot(t *testing.T) {
	dir := t.TempDir()
	a := New("")
	findings, err := a.Run(context.Background(), analysis.RunRequest{Target: dir})
	if err != nil {
		t.Fatalf("Run must not error when no snapshot available: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("want 0 findings without snapshot, got %d", len(findings))
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"4.17.19", "4.17.21", -1},
		{"4.17.21", "4.17.19", 1},
		{"4.17.19", "4.17.19", 0},
		{"v4.17.19", "4.17.19", 0},
		{"0", "4.17.19", -1},
	}
	for _, tc := range tests {
		got := compareVersions(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestVersionInRanges(t *testing.T) {
	events := []struct {
		Introduced string `json:"introduced,omitempty"`
		Fixed      string `json:"fixed,omitempty"`
	}{
		{Introduced: "0"},
		{Fixed: "4.17.21"},
	}
	if !versionInRanges("4.17.19", events) {
		t.Error("4.17.19 should be vulnerable in [0, 4.17.21)")
	}
	if versionInRanges("4.17.21", events) {
		t.Error("4.17.21 should NOT be vulnerable (fixed)")
	}
	if versionInRanges("5.0.0", events) {
		t.Error("5.0.0 should NOT be vulnerable (past fixed)")
	}
}
