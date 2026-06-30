package gitleaks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/srivastava-ami/coderev/internal/tools"
	"github.com/srivastava-ami/coderev/internal/analysis"
)

type Adapter struct {
	binary string
}

func New(binary string) *Adapter {
	if binary == "" {
		binary = "gitleaks"
	}
	return &Adapter{binary: binary}
}

func (a *Adapter) Name() string { return "gitleaks" }

func (a *Adapter) IsAvailable() bool { return tools.BinaryAvailable(a.binary) }

func (a *Adapter) Capabilities() []string {
	return []string{"security.secrets"}
}

type gitleaksResult struct {
	Description string `json:"Description"`
	File        string `json:"File"`
	Line        int    `json:"StartLine"`
	Match       string `json:"Match"`
	RuleID      string `json:"RuleID"`
	Secret      string `json:"Secret"`
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	data, err := a.execGitleaks(ctx, req)
	if err != nil {
		return nil, err
	}
	return parseGitleaksOutput(data)
}

func (a *Adapter) execGitleaks(ctx context.Context, req analysis.RunRequest) ([]byte, error) {
	f, err := os.CreateTemp("", "coderev-gitleaks-*.json")
	if err != nil {
		return nil, fmt.Errorf("gitleaks: creating temp file: %w", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	args := []string{"detect", "--source", req.Target, "--report-format", "json", "--report-path", f.Name(), "--no-git", "--exit-code", "0"}
	args = append(args, req.ExtraArgs...)
	if _, err := tools.RunTool(ctx, a.binary, "gitleaks", args); err != nil {
		return nil, err
	}
	return os.ReadFile(f.Name())
}

func parseGitleaksOutput(data []byte) ([]analysis.Finding, error) {
	var results []gitleaksResult
	if err := json.Unmarshal(data, &results); err != nil {
		if len(data) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("gitleaks: parsing output: %w", err)
	}
	findings := make([]analysis.Finding, 0, len(results))
	for _, r := range results {
		findings = append(findings, analysis.Finding{
			Rule:        "security.secrets." + r.RuleID,
			Pillar:      "security",
			Severity:    analysis.SeverityBlocker,
			File:        r.File,
			Line:        r.Line,
			Message:     fmt.Sprintf("secret detected (%s): %s", r.Description, maskSecret(r.Match)),
			Remediation: "Remove the secret from source. Rotate it immediately. Store in a secrets manager.",
			Source:      "gitleaks",
		})
	}
	return findings, nil
}

const (
	maskRevealChars = 3                   // leading/trailing chars kept visible in a masked secret
	maskMinLength   = maskRevealChars * 2 // shortest secret long enough to partially reveal
)

func maskSecret(s string) string {
	if len(s) <= maskMinLength {
		return "***"
	}
	return s[:maskRevealChars] + "***" + s[len(s)-maskRevealChars:]
}
