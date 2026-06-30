package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/llm"
	"github.com/srivastava-ami/coderev/internal/report"
)

type llmReviewReq struct {
	target string
	tc     analysis.ToolConfig
	rc     llm.ReviewContext
}

// maybeSendToLLM optionally dispatches diff-based review to the configured LLM.
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

// runFullGraphReview reviews every file in the code graph via LLM.
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

// buildFullReviewChunks organizes the code graph into LLM review chunks (one per file).
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

// writePromptFile writes the assembled LLM prompt to disk for inspection.
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

// refreshHTMLWithReview updates the HTML report with the AI review content.
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
