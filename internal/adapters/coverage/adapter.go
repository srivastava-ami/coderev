package coverage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/config"
)

const defaultThreshold = 80.0

// Adapter reads coverage reports (lcov.info or go coverage.out) and emits
// testing.coverage findings for files that fall below the configured threshold.
type Adapter struct {
	cfg config.CoverageConfig
}

func New(cfg config.CoverageConfig) *Adapter {
	return &Adapter{cfg: cfg}
}

func (a *Adapter) Name() string { return "coverage" }

func (a *Adapter) IsAvailable() bool {
	_, ok := a.findCoverageFile("")
	return ok
}

func (a *Adapter) Capabilities() []string { return []string{"testing.coverage"} }

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	covFile, ok := a.findCoverageFile(req.Target)
	if !ok {
		return nil, nil
	}

	var covMap map[string]fileCoverage
	var err error

	switch filepath.Base(covFile) {
	case "lcov.info":
		covMap, err = parseLcov(covFile)
	default:
		covMap, err = parseGoCover(covFile)
	}
	if err != nil {
		return nil, fmt.Errorf("coverage: parsing %s: %w", covFile, err)
	}

	threshold := a.cfg.Threshold
	if threshold == 0 {
		threshold = defaultThreshold
	}

	var findings []analysis.Finding
	for file, cov := range covMap {
		if cov.total == 0 {
			continue
		}
		pct := float64(cov.hit) / float64(cov.total) * 100
		if pct >= threshold {
			continue
		}
		findings = append(findings, analysis.Finding{
			Rule:        "testing.coverage",
			Pillar:      "testing",
			Severity:    analysis.SeverityMajor,
			File:        file,
			Line:        1,
			Source:      "coverage",
			Message:     fmt.Sprintf("%.1f%% line coverage (threshold %.0f%%) — %d/%d lines hit", pct, threshold, cov.hit, cov.total),
			Remediation: fmt.Sprintf("Add unit tests to reach %.0f%% coverage. Focus on untested branches first.", threshold),
		})
	}
	return findings, nil
}

// findCoverageFile searches for lcov.info or coverage.out under the target.
func (a *Adapter) findCoverageFile(target string) (string, bool) {
	if a.cfg.LcovPath != "" {
		if _, err := os.Stat(a.cfg.LcovPath); err == nil {
			return a.cfg.LcovPath, true
		}
	}
	if a.cfg.GoCoverPath != "" {
		if _, err := os.Stat(a.cfg.GoCoverPath); err == nil {
			return a.cfg.GoCoverPath, true
		}
	}
	candidates := []string{
		filepath.Join(target, "coverage", "lcov.info"),
		filepath.Join(target, "coverage.out"),
		"coverage/lcov.info",
		"coverage.out",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}
