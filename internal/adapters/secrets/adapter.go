// Package secrets is the native, dependency-free secret scanner. It is the
// default provider for the security.secrets capability behind the
// analysis.ToolAdapter port. Unlike the gitleaks adapter it shells out to no
// external binary: detection is regex rules plus a Shannon-entropy heuristic
// over the working-tree files collected by the runner.
package secrets

import (
	"context"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Adapter implements analysis.ToolAdapter natively.
type Adapter struct{}

// New returns a ready-to-use native secret scanner.
func New() *Adapter { return &Adapter{} }

// Name identifies the adapter in warnings and finding provenance.
func (a *Adapter) Name() string { return "secrets" }

// IsAvailable is always true: the scanner is pure Go with no external binary,
// so the security.secrets capability is satisfied on every machine.
func (a *Adapter) IsAvailable() bool { return true }

// Capabilities declares the rule this adapter satisfies.
func (a *Adapter) Capabilities() []string { return []string{"security.secrets"} }

// Run scans every file's content for secrets. It honours context cancellation
// between files and never errors on individual content (a malformed line just
// does not match).
func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	var findings []analysis.Finding
	for _, fi := range req.Files {
		select {
		case <-ctx.Done():
			return findings, ctx.Err()
		default:
		}
		findings = append(findings, scanContent(fi.Path, fi.Content)...)
	}
	return findings, nil
}

// scanContent runs both the named pattern rules and the generic, keyword-gated
// entropy detector over each line of a file.
func scanContent(path string, content []byte) []analysis.Finding {
	var out []analysis.Finding
	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		lineNum := i + 1

		// High-confidence named patterns.
		matchedNamed := false
		for _, r := range patternRules {
			if m := r.re.FindString(line); m != "" {
				out = append(out, newFinding(path, lineNum, r.id, r.desc, m))
				matchedNamed = true
			}
		}
		// Skip the generic detector when a named rule already fired on this
		// line — it is the same secret, just less specifically described.
		if matchedNamed {
			continue
		}

		// Generic: a secret-ish assignment whose value is high entropy.
		if m := reAssignment.FindStringSubmatch(line); m != nil {
			key, val := m[1], m[2]
			if reSecretName.MatchString(key) && looksLikeSecret(val) {
				desc := "high-entropy value assigned to " + strings.TrimSpace(key)
				out = append(out, newFinding(path, lineNum, "generic-high-entropy", desc, val))
			}
		}
	}
	return out
}

// newFinding builds a blocker security.secrets finding with a masked sample.
// The rule id is namespaced under security.secrets so it dedupes/merges
// cleanly alongside the optional gitleaks adapter in the runner.
func newFinding(path string, line int, ruleID, desc, match string) analysis.Finding {
	return analysis.Finding{
		Rule:        "security.secrets." + ruleID,
		Pillar:      "security",
		Severity:    analysis.SeverityBlocker,
		File:        path,
		Line:        line,
		Message:     "potential secret detected (" + desc + "): " + maskSecret(match),
		Remediation: "Remove the secret from source and rotate it immediately. Load it from a secrets manager or an environment variable instead.",
		Source:      "secrets",
	}
}

// maskSecret reveals only the first and last few characters of a match so the
// report can locate the leak without re-printing the full credential.
func maskSecret(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "***" + s[len(s)-4:]
}
