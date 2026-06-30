package npmaudit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func writePkgJSON(t *testing.T, dir string, content string) string {
	t.Helper()
	path := filepath.Join(dir, "package.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return path
}

func TestDevDepCappedAtAdvisory(t *testing.T) {
	dir := t.TempDir()
	writePkgJSON(t, dir, `{
  "dependencies": { "lodash": "^4.17.21" },
  "devDependencies": { "jest": "^29.0.0" }
}`)

	auditJSON, _ := json.Marshal(auditReport{
		Vulnerabilities: map[string]vuln{
			"lodash": {Name: "lodash", Severity: "critical"},
			"jest":   {Name: "jest", Severity: "high"},
		},
	})

	findings, err := parseAuditOutput(auditJSON, dir)
	if err != nil {
		t.Fatalf("parseAuditOutput: %v", err)
	}

	for _, f := range findings {
		switch {
		case f.Message == "vulnerable dependency: lodash (critical)":
			if f.Severity != analysis.SeverityBlocker {
				t.Errorf("production dep lodash should be blocker, got %s", f.Severity)
			}
		case f.Message == "vulnerable dependency: jest (high)":
			if f.Severity != analysis.SeverityAdvisory {
				t.Errorf("devDep jest (high) should be capped at advisory, got %s", f.Severity)
			}
		}
	}
}

func TestLineNumbersEmitted(t *testing.T) {
	dir := t.TempDir()
	writePkgJSON(t, dir, `{
  "dependencies": {
    "express": "^4.18.0"
  }
}`)

	auditJSON, _ := json.Marshal(auditReport{
		Vulnerabilities: map[string]vuln{
			"express": {Name: "express", Severity: "moderate"},
		},
	})

	findings, err := parseAuditOutput(auditJSON, dir)
	if err != nil {
		t.Fatalf("parseAuditOutput: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	if findings[0].Line == 0 {
		t.Error("expected non-zero line number for express vulnerability")
	}
}
