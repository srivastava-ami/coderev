package madge

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/srivastava-ami/coderev/internal/adapters/cmdutil"
	"github.com/srivastava-ami/coderev/internal/analysis"
)

type Adapter struct {
	binary string
}

func New(binary string) *Adapter {
	if binary == "" {
		binary = "madge"
	}
	return &Adapter{binary: binary}
}

func (a *Adapter) Name() string { return "madge" }

func (a *Adapter) IsAvailable() bool {
	if _, err := os.Stat(a.binary); err == nil {
		return true
	}
	_, err := exec.LookPath(a.binary)
	return err == nil
}

func (a *Adapter) Capabilities() []string {
	return []string{"file_structure.circular_deps", "nx_conventions.boundaries"}
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	data, err := a.execMadge(ctx, req)
	if err != nil {
		return nil, err
	}
	return parseMadgeOutput(data)
}

func (a *Adapter) execMadge(ctx context.Context, req analysis.RunRequest) ([]byte, error) {
	args := []string{"--circular", "--json", req.Target}
	args = append(args, req.ExtraArgs...)
	return cmdutil.RunTool(ctx, a.binary, "madge", args)
}

func parseMadgeOutput(data []byte) ([]analysis.Finding, error) {
	var cycles [][]string
	if err := json.Unmarshal(data, &cycles); err != nil {
		return nil, fmt.Errorf("madge: parsing output: %w", err)
	}
	findings := make([]analysis.Finding, 0, len(cycles))
	for _, cycle := range cycles {
		if len(cycle) == 0 {
			continue
		}
		findings = append(findings, analysis.Finding{
			Rule:        "file_structure.circular_deps",
			Pillar:      "file_structure",
			Severity:    analysis.SeverityBlocker,
			File:        cycle[0],
			Message:     fmt.Sprintf("circular dependency: %s", strings.Join(cycle, " → ")),
			Remediation: "Introduce an abstraction layer or restructure the dependency graph.",
			Source:      "madge",
		})
	}
	return findings, nil
}
