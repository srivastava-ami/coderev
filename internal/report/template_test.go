package report

import (
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/architecture"
)

func TestGenerateHTMLReportValid(t *testing.T) {
	// Create minimal report data
	rep := Report{
		Meta: Meta{
			RepoName:         "test-repo",
			StandardsVersion: "1.0",
			Generated:        "2026-06-30",
		},
		Summary: Summary{
			TotalFiles:    10,
			TotalFindings: 5,
			OverallStatus: "PASS",
			BySeverity: map[string]int{
				"blocker":  0,
				"major":    3,
				"advisory": 2,
			},
			ByPillar: map[string]int{
				"stability": 3,
			},
		},
		Findings: []analysis.Finding{
			{
				Rule:        "test.rule",
				Pillar:      "stability",
				Severity:    analysis.SeverityMajor,
				File:        "test.go",
				Line:        10,
				Message:     "test finding",
				Remediation: "fix it",
			},
		},
		Pillars: []PillarResult{
			{
				Name:   "stability",
				Status: "FAIL",
				Score:  0.8,
				Findings: []analysis.Finding{
					{Rule: "test.rule", Message: "test"},
				},
			},
		},
		Architecture: architecture.Summary{},
		Generated:    time.Now(),
	}

	// Write to temp file
	tmpfile, err := os.CreateTemp("", "report-*.html")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	err = Generate(rep, tmpfile.Name())
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Read generated HTML
	htmlBytes, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to read generated HTML: %v", err)
	}

	html := string(htmlBytes)
	if len(html) == 0 {
		t.Error("generated HTML is empty")
	}

	// Validate essential structure
	checks := []struct {
		name    string
		pattern string
	}{
		{"const R =", `const R = \{`},
		{"renderHTML function", `function renderHTML\(`},
		{"DOMContentLoaded", `DOMContentLoaded`},
		{"renderWarnings function", `function renderWarnings\(\)`},
		{"renderViolations function", `function renderViolations\(\)`},
		{"renderPillars function", `function renderPillars\(\)`},
		{"repo name", `test-repo`},
		{"overall status", `PASS`},
	}

	for _, check := range checks {
		if !regexp.MustCompile(check.pattern).MatchString(html) {
			t.Errorf("missing expected pattern: %s (%s)", check.name, check.pattern)
		}
	}

	// Validate arrow function syntax in renderPillars and renderViolations
	// Should have: pillars.map(p => { (NOT p => {) )
	arrowFuncs := []struct {
		name    string
		pattern string
	}{
		{"renderPillars arrow", `pillars\.map\(p => \{\s*const pct`},
		{"renderViolations arrow", `filtered\.map\(\(f, i\) => \{\s*const shortFile`},
	}

	for _, check := range arrowFuncs {
		if !regexp.MustCompile(check.pattern).MatchString(html) {
			t.Errorf("arrow function syntax may be incorrect: %s", check.name)
		}
	}

	// Check for broken arrow function syntax (misplaced closing paren after {)
	brokenArrows := []string{
		`=> \{\)`,  // Should be => {, not => {)
	}

	for _, pattern := range brokenArrows {
		if regexp.MustCompile(pattern).MatchString(html) {
			t.Errorf("detected broken arrow function syntax: %s", pattern)
		}
	}
}
