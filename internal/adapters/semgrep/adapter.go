package semgrep

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/srivastava-ami/coderev/internal/adapters/cmdutil"
	"github.com/srivastava-ami/coderev/internal/analysis"
)

type Adapter struct {
	binary string
}

func New(binary string) *Adapter {
	if binary == "" {
		binary = "semgrep"
	}
	return &Adapter{binary: binary}
}

func (a *Adapter) Name() string { return "semgrep" }

func (a *Adapter) IsAvailable() bool {
	_, err := exec.LookPath(a.binary)
	return err == nil
}

func (a *Adapter) Capabilities() []string {
	return []string{"security.injection.*", "security.auth.*", "security.cryptography"}
}

type semgrepOutput struct {
	Results []semgrepResult `json:"results"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type semgrepResult struct {
	CheckID string `json:"check_id"`
	Path    string `json:"path"`
	Start   struct {
		Line int `json:"line"`
		Col  int `json:"col"`
	} `json:"start"`
	Extra struct {
		Message  string `json:"message"`
		Severity string `json:"severity"`
		Lines    string `json:"lines"`
		Metadata struct {
			Category string `json:"category"`
		} `json:"metadata"`
	} `json:"extra"`
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	data, err := a.execSemgrep(ctx, req)
	if err != nil {
		return nil, err
	}
	return parseSemgrepOutput(data)
}

func (a *Adapter) execSemgrep(ctx context.Context, req analysis.RunRequest) ([]byte, error) {
	args := []string{"--config", "p/owasp-top-ten", "--config", "p/secrets", "--json", "--quiet", req.Target}
	args = append(args, req.ExtraArgs...)
	return cmdutil.RunTool(ctx, a.binary, "semgrep", args)
}

func parseSemgrepOutput(data []byte) ([]analysis.Finding, error) {
	var out semgrepOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("semgrep: parsing output: %w", err)
	}
	findings := make([]analysis.Finding, 0, len(out.Results))
	for _, r := range out.Results {
		findings = append(findings, analysis.Finding{
			Rule:     "security.semgrep." + r.CheckID,
			Pillar:   "security",
			Severity: mapSeverity(r.Extra.Severity),
			File:     r.Path,
			Line:     r.Start.Line,
			Column:   r.Start.Col,
			Message:  r.Extra.Message,
			Snippet:  r.Extra.Lines,
			Source:   "semgrep",
		})
	}
	return findings, nil
}

func mapSeverity(s string) analysis.Severity {
	switch s {
	case "ERROR", "error":
		return analysis.SeverityBlocker
	case "WARNING", "warning":
		return analysis.SeverityMajor
	default:
		return analysis.SeverityAdvisory
	}
}
