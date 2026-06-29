package llm

import (
	"bytes"
	"context"
	"fmt"
	"os"
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
	f, err := os.CreateTemp("", "coderev-prompt-*.md")
	if err != nil {
		return "", fmt.Errorf("llm: creating temp prompt file: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(prompt); err != nil {
		f.Close()
		return "", fmt.Errorf("llm: writing prompt: %w", err)
	}
	f.Close()

	args := buildArgs(fields[1:], prompt, f.Name())
	cmd := exec.CommandContext(ctx, fields[0], args...)
	if !hasPromptPlaceholder(fields[1:]) {
		data, err := os.ReadFile(f.Name())
		if err == nil {
			cmd.Stdin = bytes.NewReader(data)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("llm: %q exited with error: %w: %s", fields[0], err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func buildArgs(fields []string, prompt, promptFile string) []string {
	args := make([]string, 0, len(fields))
	for _, f := range fields {
		switch f {
		case "{prompt}":
			args = append(args, prompt)
		case "{prompt_file}":
			args = append(args, promptFile)
		default:
			args = append(args, f)
		}
	}
	return args
}

func hasPromptPlaceholder(fields []string) bool {
	for _, f := range fields {
		if f == "{prompt}" || f == "{prompt_file}" {
			return true
		}
	}
	return false
}
