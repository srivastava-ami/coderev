package cmdutil

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// RunTool executes an external binary and returns its stdout.
// If the process fails AND produced no output, the error includes stderr.
// This pattern is shared by all external-binary adapters.
func RunTool(ctx context.Context, binary, name string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		return nil, fmt.Errorf("%s: %w — stderr: %s", name, err, stderr.String())
	}
	return stdout.Bytes(), nil
}
