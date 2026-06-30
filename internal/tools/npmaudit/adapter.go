package npmaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/srivastava-ami/coderev/internal/tools"
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

func (a *Adapter) IsAvailable() bool { return tools.BinaryAvailable(a.binary) }

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

// pkgMeta holds devDependency membership and per-package line numbers parsed
// from package.json. Used to cap severity for dev deps and emit precise line numbers.
type pkgMeta struct {
	devDeps map[string]bool
	lineFor map[string]int
}

func loadPkgMeta(pkgPath string) *pkgMeta {
	meta := &pkgMeta{
		devDeps: make(map[string]bool),
		lineFor: make(map[string]int),
	}
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return meta
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return meta
	}
	for name := range pkg.DevDependencies {
		meta.devDeps[name] = true
	}
	// Collect all declared dep names so we can find their line numbers.
	allDeps := make(map[string]bool, len(pkg.Dependencies)+len(pkg.DevDependencies))
	for name := range pkg.Dependencies {
		allDeps[name] = true
	}
	for name := range pkg.DevDependencies {
		allDeps[name] = true
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for name := range allDeps {
			if _, seen := meta.lineFor[name]; seen {
				continue
			}
			if strings.HasPrefix(trimmed, `"`+name+`"`) {
				meta.lineFor[name] = i + 1
			}
		}
	}
	return meta
}

func parseAuditOutput(data []byte, pkgDir string) ([]analysis.Finding, error) {
	var report auditReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("npmaudit: parsing output: %w", err)
	}
	pkgPath := filepath.Join(pkgDir, "package.json")
	meta := loadPkgMeta(pkgPath)

	var findings []analysis.Finding
	for _, v := range report.Vulnerabilities {
		sev := mapSeverity(v.Severity)
		if sev == analysis.SeverityInfo {
			continue
		}
		// devDependencies only affect the development environment — cap at advisory.
		if meta.devDeps[v.Name] && (sev == analysis.SeverityBlocker || sev == analysis.SeverityMajor) {
			sev = analysis.SeverityAdvisory
		}
		findings = append(findings, analysis.Finding{
			Rule:        "security.dependencies",
			Pillar:      "security",
			Severity:    sev,
			File:        pkgPath,
			Line:        meta.lineFor[v.Name],
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
