package semgrep

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/srivastava-ami/coderev/internal/adapters/cmdutil"
	"github.com/srivastava-ami/coderev/internal/analysis"
)

//go:embed rules/*.yaml
var embeddedRules embed.FS

type Adapter struct{ binary string }

func New(binary string) *Adapter {
	if binary == "" {
		binary = "semgrep"
	}
	return &Adapter{binary: binary}
}

func (a *Adapter) Name() string      { return "semgrep" }
func (a *Adapter) IsAvailable() bool { return cmdutil.BinaryAvailable(a.binary) }
func (a *Adapter) Capabilities() []string {
	return []string{"security.pattern.*"}
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	data, err := a.execSemgrep(ctx, req)
	if err != nil {
		return nil, err
	}
	return parseSemgrepOutput(data)
}

func (a *Adapter) execSemgrep(ctx context.Context, req analysis.RunRequest) ([]byte, error) {
	rulesDir, err := extractRulesToTemp(embeddedRules)
	if err != nil {
		return nil, fmt.Errorf("semgrep: extracting embedded rules: %w", err)
	}
	defer os.RemoveAll(rulesDir)
	args := []string{"--config", rulesDir, "--json", "--quiet", "--no-rewrite-rule-ids", req.Target}
	args = append(args, req.ExtraArgs...)
	return cmdutil.RunTool(ctx, a.binary, "semgrep", args)
}

func extractRulesToTemp(rules embed.FS) (string, error) {
	dir, err := os.MkdirTemp("", "coderev-semgrep-*")
	if err != nil {
		return "", err
	}
	entries, err := fs.ReadDir(rules, "rules")
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := fs.ReadFile(rules, "rules/"+e.Name())
		if err != nil {
			os.RemoveAll(dir)
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o644); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
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

func parseSemgrepOutput(data []byte) ([]analysis.Finding, error) {
	var out semgrepOutput
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("semgrep: parsing output: %w", err)
	}
	findings := make([]analysis.Finding, 0, len(out.Results))
	for _, r := range out.Results {
		findings = append(findings, analysis.Finding{
			Rule:     r.CheckID,
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
