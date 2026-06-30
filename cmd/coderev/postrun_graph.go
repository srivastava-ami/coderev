package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/graph"
	"github.com/srivastava-ami/coderev/internal/llm"
)

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

// buildReviewContext assembles LLM context from findings, diff hunks, and graph neighbors.
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

// applyIgnoreList filters hunks, findings, and neighbors by .coderevignore.
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

// loadIgnoreList loads and returns the .coderevignore file.
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
