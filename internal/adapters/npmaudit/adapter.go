package npmaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type Adapter struct {
	binary string
}

func New(binary string) *Adapter {
	if binary == "" {
		binary = "npm"
	}
	return &Adapter{binary: binary}
}

func (a *Adapter) Name() string { return "npmaudit" }

func (a *Adapter) IsAvailable() bool {
	_, err := exec.LookPath(a.binary)
	return err == nil
}

func (a *Adapter) Capabilities() []string {
	return []string{"security.dependencies"}
}

type auditReport struct {
	Vulnerabilities map[string]vuln `json:"vulnerabilities"`
}

type vuln struct {
	Name     string   `json:"name"`
	Severity string   `json:"severity"`
	Via      []any    `json:"via"`
	Nodes    []string `json:"nodes"`
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	pkgDir := req.Target
	if !fileExists(filepath.Join(pkgDir, "package.json")) {
		return nil, nil // not a Node.js project
	}
	data, err := a.execNpmAudit(ctx, pkgDir)
	if err != nil {
		return nil, err
	}
	return parseAuditOutput(data, pkgDir)
}

func (a *Adapter) execNpmAudit(ctx context.Context, pkgDir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, a.binary, "audit", "--json")
	cmd.Dir = pkgDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // npm audit exits non-zero when vulnerabilities are found
	if stdout.Len() == 0 {
		return nil, fmt.Errorf("npmaudit: no output — stderr: %s", stderr.String())
	}
	return stdout.Bytes(), nil
}

func parseAuditOutput(data []byte, pkgDir string) ([]analysis.Finding, error) {
	var report auditReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("npmaudit: parsing output: %w", err)
	}
	var findings []analysis.Finding
	for _, v := range report.Vulnerabilities {
		sev := mapSeverity(v.Severity)
		if sev == analysis.SeverityInfo {
			continue
		}
		findings = append(findings, analysis.Finding{
			Rule:        "security.dependencies",
			Pillar:      "security",
			Severity:    sev,
			File:        filepath.Join(pkgDir, "package.json"),
			Message:     fmt.Sprintf("vulnerable dependency: %s (%s)", v.Name, v.Severity),
			Remediation: "Run `npm audit fix` or upgrade the affected package manually.",
			Source:      "npmaudit",
		})
	}
	return findings, nil
}

func mapSeverity(s string) analysis.Severity {
	switch s {
	case "critical", "high":
		return analysis.SeverityBlocker
	case "moderate":
		return analysis.SeverityMajor
	case "low":
		return analysis.SeverityAdvisory
	default:
		return analysis.SeverityInfo
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
