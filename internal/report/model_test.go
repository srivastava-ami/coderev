package report

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/architecture"
	"github.com/srivastava-ami/coderev/internal/config"
)

func sampleFindings() []analysis.Finding {
	return []analysis.Finding{
		{Rule: "complexity.cyclomatic", Pillar: "complexity", Severity: analysis.SeverityBlocker, File: "/a/b.ts", Line: 10, Message: "too complex"},
		{Rule: "type_safety.no_any", Pillar: "type_safety", Severity: analysis.SeverityBlocker, File: "/a/b.ts", Line: 20, Message: "any type"},
		{Rule: "documentation.todo_format", Pillar: "documentation", Severity: analysis.SeverityMajor, File: "/a/c.ts", Line: 5, Message: "TODO without ticket"},
		{Rule: "observability.logging", Pillar: "observability", Severity: analysis.SeverityAdvisory, File: "/a/d.ts", Line: 3, Message: "console.log"},
	}
}

func sampleFiles() []analysis.FileInfo {
	return []analysis.FileInfo{
		{Path: "/a/b.ts", Language: analysis.LangTypeScript, Lines: 100},
		{Path: "/a/c.ts", Language: analysis.LangTypeScript, Lines: 50},
		{Path: "/a/d.ts", Language: analysis.LangTypeScript, Lines: 30},
	}
}

func req(findings []analysis.Finding, files []analysis.FileInfo, warnings []analysis.AdapterWarning) BuildRequest {
	return BuildRequest{
		Target:    "/a",
		Standards: config.Standards{},
		StdFile:   "standards.toml",
		Files:     files,
		Findings:  findings,
		Warnings:  warnings,
		Arch:      architecture.Summary{},
	}
}

func TestBuildSetsOverallStatusFAIL(t *testing.T) {
	r := Build(req(sampleFindings(), sampleFiles(), nil))
	if r.Summary.OverallStatus != "FAIL" {
		t.Errorf("OverallStatus = %q, want FAIL (blockers present)", r.Summary.OverallStatus)
	}
}

func TestBuildSetsOverallStatusPASS(t *testing.T) {
	advisoryOnly := []analysis.Finding{
		{Rule: "observability.logging", Pillar: "observability", Severity: analysis.SeverityAdvisory, File: "/a/b.ts", Line: 1},
	}
	r := Build(req(advisoryOnly, sampleFiles(), nil))
	if r.Summary.OverallStatus != "PASS" {
		t.Errorf("OverallStatus = %q, want PASS (only advisory findings)", r.Summary.OverallStatus)
	}
}

func TestBuildCountsBySeverity(t *testing.T) {
	r := Build(req(sampleFindings(), sampleFiles(), nil))
	if r.Summary.BySeverity["blocker"] != 2 {
		t.Errorf("blocker count = %d, want 2", r.Summary.BySeverity["blocker"])
	}
	if r.Summary.BySeverity["major"] != 1 {
		t.Errorf("major count = %d, want 1", r.Summary.BySeverity["major"])
	}
	if r.Summary.BySeverity["advisory"] != 1 {
		t.Errorf("advisory count = %d, want 1", r.Summary.BySeverity["advisory"])
	}
}

func TestBuildTotalFindingsCount(t *testing.T) {
	r := Build(req(sampleFindings(), sampleFiles(), nil))
	if r.Summary.TotalFindings != 4 {
		t.Errorf("TotalFindings = %d, want 4", r.Summary.TotalFindings)
	}
}

func TestBuildTotalFilesCount(t *testing.T) {
	r := Build(req(sampleFindings(), sampleFiles(), nil))
	if r.Summary.TotalFiles != 3 {
		t.Errorf("TotalFiles = %d, want 3", r.Summary.TotalFiles)
	}
}

func TestBuildPillarsGrouped(t *testing.T) {
	r := Build(req(sampleFindings(), sampleFiles(), nil))
	pillarNames := map[string]bool{}
	for _, p := range r.Pillars {
		pillarNames[p.Name] = true
	}
	for _, expected := range []string{"complexity", "type_safety", "documentation", "observability"} {
		if !pillarNames[expected] {
			t.Errorf("expected pillar %q in result, not found", expected)
		}
	}
}

func TestBuildFileResultsSortedByFindingCount(t *testing.T) {
	r := Build(req(sampleFindings(), sampleFiles(), nil))
	if len(r.Files) == 0 {
		t.Fatal("expected file results, got none")
	}
	if len(r.Files[0].Findings) < len(r.Files[len(r.Files)-1].Findings) {
		t.Error("file results should be sorted descending by finding count")
	}
}

func TestBuildMetaRepoName(t *testing.T) {
	r := Build(BuildRequest{Target: "/home/user/my-project", StdFile: "standards.toml"})
	if r.Meta.RepoName != "my-project" {
		t.Errorf("RepoName = %q, want my-project", r.Meta.RepoName)
	}
}

func TestBuildWithWarnings(t *testing.T) {
	warnings := []analysis.AdapterWarning{{Adapter: "semgrep", Reason: "binary not found"}}
	r := Build(req(nil, sampleFiles(), warnings))
	if len(r.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(r.Warnings))
	}
	if r.Warnings[0].Adapter != "semgrep" {
		t.Errorf("warning adapter = %q, want semgrep", r.Warnings[0].Adapter)
	}
}

func TestBuildEmptyFindings(t *testing.T) {
	r := Build(req(nil, sampleFiles(), nil))
	if r.Summary.OverallStatus != "PASS" {
		t.Errorf("empty findings should be PASS, got %s", r.Summary.OverallStatus)
	}
	if r.Summary.TotalFindings != 0 {
		t.Errorf("expected 0 total findings, got %d", r.Summary.TotalFindings)
	}
}
