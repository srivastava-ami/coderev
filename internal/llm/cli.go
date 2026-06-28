package llm

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CLIProvider struct {
	command string
}

func (p *CLIProvider) Complete(ctx context.Context, prompt string) (string, error) {
	fields := strings.Fields(p.command)
	if len(fields) == 0 {
		return "", fmt.Errorf("llm: empty cli_command")
	}
	args := make([]string, 0, len(fields))
	for _, f := range fields[1:] {
		if f == "{prompt}" {
			args = append(args, prompt)
		} else {
			args = append(args, f)
		}
	}
	cmd := exec.CommandContext(ctx, fields[0], args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("llm: %q exited with error: %w: %s", fields[0], err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
