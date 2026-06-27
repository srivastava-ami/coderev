// Package depcve provides a native ToolAdapter that detects known CVEs in
// third-party dependencies by matching lockfile entries against an offline
// OSV vulnerability snapshot. It is the default provider for
// security.dependencies, replacing the npm-audit-bound npmaudit adapter with
// a pure-Go, offline, multi-ecosystem alternative.
package depcve

import (
	"context"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

const adapterName = "depcve"

type Adapter struct {
	snapshotURL string
}

func New(snapshotURL string) *Adapter {
	return &Adapter{snapshotURL: snapshotURL}
}

func (a *Adapter) Name() string { return adapterName }

func (a *Adapter) IsAvailable() bool { return true }

func (a *Adapter) Capabilities() []string {
	return []string{"security.dependencies"}
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	vulns := loadSnapshot(req.Target, a.snapshotURL)
	if len(vulns) == 0 {
		return nil, nil
	}

	deps, err := parseLockfiles(req.Target)
	if err != nil || len(deps) == 0 {
		return nil, nil
	}

	var findings []analysis.Finding
	for _, dep := range deps {
		for _, vuln := range vulns {
			ids := matchVuln(dep, vuln)
			if len(ids) == 0 {
				continue
			}
			msg := dep.Name + "@" + dep.Version + " " + ids[0]
			findings = append(findings, analysis.Finding{
				Rule:        "security.dependencies",
				Pillar:      "security",
				Severity:    analysis.SeverityBlocker,
				File:        dep.File,
				Message:     msg,
				Remediation: "Upgrade to the latest patched version of the dependency. Run 'npm audit', 'go mod tidy', or check the CVE advisory for the fixed version.",
				Source:      adapterName,
				Tags:        ids,
				Standards:   ids,
			})
		}
	}
	return findings, nil
}
