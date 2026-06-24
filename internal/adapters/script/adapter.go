package script

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/srivastava-ami/coderev/internal/adapters/cmdutil"
	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Adapter is the extensibility hook: any external program that writes findings
// as NDJSON or JSON in our Finding schema can plug in via tool_config.toml
// without touching Go source.
//
// Protocol "ndjson": one JSON-encoded Finding per line.
// Protocol "json":   a JSON array of Finding objects.
type Adapter struct {
	name     string
	binary   string
	protocol string
	rules    []string
	extraArgs []string
}

func New(name, binary, protocol string, rules, extraArgs []string) *Adapter {
	if protocol == "" {
		protocol = "ndjson"
	}
	return &Adapter{name: name, binary: binary, protocol: protocol, rules: rules, extraArgs: extraArgs}
}

func (a *Adapter) Name() string { return a.name }

func (a *Adapter) IsAvailable() bool {
	_, err := exec.LookPath(a.binary)
	return err == nil
}

func (a *Adapter) Capabilities() []string { return a.rules }

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	args := substituteTarget(a.extraArgs, req.Target)
	data, err := cmdutil.RunTool(ctx, a.binary, "script:"+a.name, args)
	if err != nil {
		return nil, err
	}
	switch a.protocol {
	case "json":
		return parseJSON(data, a.name)
	default:
		return parseNDJSON(data, a.name)
	}
}

func parseNDJSON(data []byte, source string) ([]analysis.Finding, error) {
	var findings []analysis.Finding
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var f analysis.Finding
		if err := json.Unmarshal(line, &f); err != nil {
			return findings, fmt.Errorf("script adapter: invalid NDJSON line: %w", err)
		}
		f.Source = source
		findings = append(findings, f)
	}
	return findings, nil
}

func parseJSON(data []byte, source string) ([]analysis.Finding, error) {
	var findings []analysis.Finding
	if err := json.Unmarshal(data, &findings); err != nil {
		return nil, fmt.Errorf("script adapter: invalid JSON: %w", err)
	}
	for i := range findings {
		findings[i].Source = source
	}
	return findings, nil
}

func substituteTarget(args []string, target string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = strings.ReplaceAll(a, "{{target}}", target)
	}
	return out
}
