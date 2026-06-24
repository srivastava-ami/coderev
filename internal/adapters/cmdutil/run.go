package cmdutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// BinaryAvailable reports whether name exists at the given path or on $PATH.
// Shared by all external-binary adapters for their IsAvailable() method.
func BinaryAvailable(binary string) bool {
	if _, err := os.Stat(binary); err == nil {
		return true
	}
	_, err := exec.LookPath(binary)
	return err == nil
}

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
