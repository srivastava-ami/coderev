package main

import (
	"fmt"
	"os"
	"time"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Constants for coderev paths and file permissions.
const coderevIgnoreFile = ".coderev/.coderevignore"
const promptFile = ".coderev/prompt.md"
const reviewFile = ".coderev/review.md"
const coderevDirPerms = 0o755
const coderevFilePerms = 0o644

// fmtTokens formats a token count with thousands separators.
func fmtTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d", n)
}

// startSpinner displays a rotating spinner with a label.
func startSpinner(label string) func() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	done := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprintf(os.Stderr, "\r%-60s\r", "")
				return
			case <-time.After(100 * time.Millisecond):
				fmt.Fprintf(os.Stderr, "\r%s %s", label, frames[i%len(frames)])
				i++
			}
		}
	}()
	return func() { close(done) }
}

// filterFindingsByFile filters findings by file path.
func filterFindingsByFile(findings []analysis.Finding, file string) []analysis.Finding {
	var out []analysis.Finding
	for _, f := range findings {
		if f.File == file {
			out = append(out, f)
		}
	}
	return out
}
