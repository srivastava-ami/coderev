package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/graph"
	"github.com/srivastava-ami/coderev/internal/llm"
	"github.com/srivastava-ami/coderev/internal/report"
)

const coderevIgnoreFile = ".coderev/.coderevignore"

func loadIgnoreList(target string) config.IgnoreList {
	ignorePath := filepath.Join(target, coderevIgnoreFile)
	_ = config.WriteDefaultIgnoreList(ignorePath)
	il, err := config.LoadIgnoreList(ignorePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: loading .coderevignore: %v\n", err)
		return config.IgnoreList{}
	}
	return il
}

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

func buildReviewContext(target string, findings []analysis.Finding, graphDir string) llm.ReviewContext {
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
	return applyIgnoreList(target, rc)
}

func applyIgnoreList(target string, rc llm.ReviewContext) llm.ReviewContext {
	il := loadIgnoreList(target)
	before := len(rc.Hunks) + len(rc.Findings) + len(rc.Neighbors)

	filtered := rc.Hunks[:0]
	for _, h := range rc.Hunks {
		if !il.Matches(h.File) {
			filtered = append(filtered, h)
		}
	}
	rc.Hunks = filtered

	filteredF := rc.Findings[:0]
	for _, f := range rc.Findings {
		if !il.Matches(f.File) {
			filteredF = append(filteredF, f)
		}
	}
	rc.Findings = filteredF

	filteredN := rc.Neighbors[:0]
	for _, n := range rc.Neighbors {
		if !il.Matches(n.File) {
			filteredN = append(filteredN, n)
		}
	}
	rc.Neighbors = filteredN

	after := len(rc.Hunks) + len(rc.Findings) + len(rc.Neighbors)
	if before-after > 0 {
		fmt.Fprintf(os.Stderr, "  ignore: %d item(s) filtered from LLM context\n", before-after)
	}
	return rc
}

func writePromptFile(target string, rc llm.ReviewContext) error {
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

type llmReviewReq struct {
	target string
	tc     analysis.ToolConfig
	rc     llm.ReviewContext
}

func maybeSendToLLM(ctx context.Context, req llmReviewReq) error {
	target, tc, rc := req.target, req.tc, req.rc
	if len(rc.Hunks) == 0 {
		fmt.Fprintln(os.Stderr, "  review: ⚠  no diff context — pass --diff <base-ref> for code-anchored review (higher hallucination risk without it)")
	}
	if !tc.LLM.Enabled {
		fmt.Fprintln(os.Stderr, "  review: LLM not configured — run: coderev config llm --enable --provider cli --command \"claude -p {prompt}\"")
		return nil
	}
	provider, err := llm.New(tc.LLM)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: LLM provider: %v\n", err)
		return nil
	}
	chunks := llm.ChunkByFile(rc)
	var review string
	var totalUsage llm.TokenUsage
	if len(chunks) <= 1 {
		prompt := llm.AssemblePrompt(rc)
		estTokens := len(prompt) / 4
		stop := startSpinner(fmt.Sprintf("  review: asking AI (~%s input tokens)", fmtTokens(estTokens)))
		review, totalUsage, err = provider.Complete(ctx, prompt)
		stop()
	} else {
		review, totalUsage, err = llm.ReviewChunked(ctx, provider, chunks, func(p llm.ChunkProgress) {
			fmt.Fprintf(os.Stderr, "\r  review: chunk %d/%d — %s (~%s tokens)%-10s",
				p.N, p.Total, p.File, fmtTokens(p.Est), "")
		})
		fmt.Fprintf(os.Stderr, "\r%-70s\r", "")
	}
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
		reviewFile, fmtTokens(totalUsage.InputTokens), fmtTokens(totalUsage.OutputTokens))
	return nil
}

func runFullGraphReview(ctx context.Context, req llmReviewReq, graphDir string) error {
	if graphDir == "" {
		fmt.Fprintln(os.Stderr, "  full-review: no graph available — run coderev . first")
		return nil
	}
	chunks, err := buildFullReviewChunks(req, graphDir)
	if err != nil || len(chunks) == 0 {
		return err
	}
	tc := req.tc
	if !tc.LLM.Enabled {
		fmt.Fprintln(os.Stderr, "  full-review: LLM not configured — run: coderev config llm --enable --provider cli --command \"claude -p {prompt}\"")
		return nil
	}
	provider, pErr := llm.New(tc.LLM)
	if pErr != nil {
		fmt.Fprintf(os.Stderr, "warning: LLM provider: %v\n", pErr)
		return nil
	}
	fmt.Fprintf(os.Stderr, "  full-review: %d file chunk(s) from graph\n", len(chunks))
	review, usage, rErr := llm.ReviewChunked(ctx, provider, chunks, func(p llm.ChunkProgress) {
		fmt.Fprintf(os.Stderr, "\r  full-review: chunk %d/%d — %s (~%s tokens)%-10s",
			p.N, p.Total, p.File, fmtTokens(p.Est), "")
	})
	fmt.Fprintf(os.Stderr, "\r%-70s\r", "")
	if rErr != nil {
		fmt.Fprintf(os.Stderr, "warning: full-review LLM completion: %v\n", rErr)
		return nil
	}
	outPath := filepath.Join(req.target, reviewFile)
	if wErr := os.WriteFile(outPath, []byte(review), coderevFilePerms); wErr != nil {
		fmt.Fprintf(os.Stderr, "warning: writing review file: %v\n", wErr)
		return nil
	}
	fmt.Fprintf(os.Stderr, "  full-review: %s  (in: %s · out: %s tokens)\n",
		reviewFile, fmtTokens(usage.InputTokens), fmtTokens(usage.OutputTokens))
	return nil
}

func buildFullReviewChunks(req llmReviewReq, graphDir string) ([]llm.ReviewChunk, error) {
	data, err := os.ReadFile(filepath.Join(graphDir, "graph.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: full-review: reading graph.json: %v\n", err)
		return nil, nil
	}
	byFile, err := llm.GraphNodesByFile(data)
	if err != nil || len(byFile) == 0 {
		fmt.Fprintln(os.Stderr, "  full-review: graph empty or unreadable")
		return nil, nil
	}
	il := loadIgnoreList(req.target)
	var chunks []llm.ReviewChunk
	for file, neighbors := range byFile {
		if il.Matches(file) {
			continue
		}
		rc := llm.ReviewContext{Neighbors: neighbors, Findings: filterFindingsByFile(req.rc.Findings, file)}
		chunks = append(chunks, llm.ReviewChunk{File: file, Ctx: rc})
	}
	if len(chunks) == 0 {
		fmt.Fprintln(os.Stderr, "  full-review: all files filtered by .coderevignore")
	}
	return chunks, nil
}

func filterFindingsByFile(findings []analysis.Finding, file string) []analysis.Finding {
	var out []analysis.Finding
	for _, f := range findings {
		if f.File == file {
			out = append(out, f)
		}
	}
	return out
}

func refreshHTMLWithReview(r report.Report, target string) error {
	data, err := os.ReadFile(filepath.Join(target, reviewFile))
	if err != nil {
		return nil
	}
	r.AIReview = string(data)
	htmlPath := filepath.Join(target, ".coderev", "report.html")
	if err := report.Generate(r, htmlPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: refreshing HTML report: %v\n", err)
		return nil
	}
	fmt.Fprintf(os.Stderr, "  report: %s updated with AI review\n", ".coderev/report.html")
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
