package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/graph"
	"github.com/srivastava-ami/coderev/internal/llm"
)

const promptFile = ".coderev/prompt.md"
const reviewFile = ".coderev/review.md"
const coderevDirPerms = 0o755
const coderevFilePerms = 0o644

// buildGraphInline always rebuilds the code graph and returns the output dir.
func buildGraphInline(target string, tc analysis.ToolConfig) string {
	dir := tc.Graph.OutputDir
	if dir == "" {
		dir = defaultGraphDir
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(target, dir)
	}
	g, err := graph.Build(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: graph build failed: %v\n", err)
		return ""
	}
	graph.ComputeMetrics(g)
	if err := graph.ExportAll(g, dir); err != nil {
		fmt.Fprintf(os.Stderr, "warning: graph export failed: %v\n", err)
		return ""
	}
	fmt.Fprintf(os.Stderr, "  graph: %d nodes → %s/graph.json\n", len(g.Nodes), dir)
	return dir
}

// writePromptFile assembles the LLM review prompt and writes it to .coderev/prompt.md.
func writePromptFile(target string, findings []analysis.Finding, graphDir string) error {
	rc := llm.ReviewContext{BaseRef: flagDiff, Findings: findings}
	if flagDiff != "" {
		hunks, err := gitDiffHunks(context.Background(), flagDiff, target)
		if err == nil && len(hunks) > 0 {
			rc.Hunks = hunks
			fakeCfg := analysis.ToolConfig{}
			fakeCfg.Graph.OutputDir = graphDir
			rc.Neighbors = loadGraphNeighbors(fakeCfg, target, changedFileSet(hunks))
		}
	}
	if len(rc.Neighbors) == 0 && graphDir != "" {
		if data, err := os.ReadFile(filepath.Join(graphDir, "graph.json")); err == nil {
			rc.Neighbors, _ = llm.AllGraphNodes(data)
		}
	}
	prompt := llm.AssemblePrompt(rc)
	outPath := filepath.Join(target, promptFile)
	if err := os.MkdirAll(filepath.Dir(outPath), coderevDirPerms); err != nil {
		return err
	}
	if err := os.WriteFile(outPath, []byte(prompt), coderevFilePerms); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "  prompt: %s\n", promptFile)
	return nil
}

// maybeSendToLLM sends the assembled prompt to the configured LLM provider and
// writes the review to .coderev/review.md. LLM failures are warnings only.
func maybeSendToLLM(ctx context.Context, target string, tc analysis.ToolConfig) error {
	if !tc.LLM.Enabled {
		fmt.Fprintln(os.Stderr, "  review: LLM not configured — run: coderev config llm --enable --provider cli --command \"claude -p {prompt}\"")
		return nil
	}
	data, err := os.ReadFile(filepath.Join(target, promptFile))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: reading prompt file: %v\n", err)
		return nil
	}
	provider, err := llm.New(tc.LLM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: LLM provider: %v\n", err)
		return nil
	}
	estTokens := len(data) / 4
	stop := startSpinner(fmt.Sprintf("  review: asking AI (~%s input tokens)", fmtTokens(estTokens)))
	review, usage, err := provider.Complete(ctx, string(data))
	stop()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: LLM completion: %v\n", err)
		return nil
	}
	outPath := filepath.Join(target, reviewFile)
	if err := os.WriteFile(outPath, []byte(review), coderevFilePerms); err != nil {
		fmt.Fprintf(os.Stderr, "warning: writing review file: %v\n", err)
		return nil
	}
	fmt.Fprintf(os.Stderr, "  review: %s  (in: %s · out: %s tokens)\n",
		reviewFile, fmtTokens(usage.InputTokens), fmtTokens(usage.OutputTokens))
	return nil
}

func fmtTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d", n)
}

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
