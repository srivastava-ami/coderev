package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/architecture"
	"github.com/srivastava-ami/coderev/internal/baseline"
	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/output"
	"github.com/srivastava-ami/coderev/internal/output/ghpr"
	"github.com/srivastava-ami/coderev/internal/plugin"
	"github.com/srivastava-ami/coderev/internal/quality"
	"github.com/srivastava-ami/coderev/internal/report"
)

var gitDiff gitDiffService

func graphJSONPath(target string, tc analysis.ToolConfig) string {
	dir := tc.Graph.OutputDir
	if dir == "" {
		dir = defaultGraphDir
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(target, dir)
	}
	return filepath.Join(dir, "graph.json")
}

func discoverAndPrintPlugins() {
	pluginDir := flagPluginDir
	if pluginDir == "" {
		var err error
		pluginDir, err = getPluginDir()
		if err != nil {
			return
		}
	}
	plugins, err := plugin.DiscoverPlugins(pluginDir)
	if err != nil || len(plugins) == 0 {
		return
	}
	fmt.Printf("          plugins: %d found\n", len(plugins))
	for _, p := range plugins {
		fmt.Printf("            • %s %s (%s)\n", p.Manifest.Name, p.Manifest.Version, p.Manifest.Description)
	}
}

func resolveGate(findings []analysis.Finding) (quality.GateResult, error) {
	if flagGate == "" && !flagJSON {
		return quality.GateResult{}, nil
	}
	gc := config.DefaultGateConfig()
	if flagGate != "" {
		loaded, err := config.LoadGate(flagGate)
		if err != nil {
			return quality.GateResult{}, fmt.Errorf("loading gate config: %w", err)
		}
		gc = *loaded
	}
	return quality.Evaluate(findings, gc), nil
}

func writeJSONOutput(result analysis.RunResult, gateResult quality.GateResult) error {
	if err := output.WriteJSON(result, os.Stdout, gateResult); err != nil {
		return fmt.Errorf("writing JSON output: %w", err)
	}
	if !gateResult.Passed {
		return fmt.Errorf("quality gate: FAILED — %s", gateResult.Message)
	}
	return nil
}

func printGateResult(gateResult quality.GateResult) error {
	if flagGate == "" {
		return nil
	}
	if gateResult.Passed {
		fmt.Println("quality gate: PASSED")
		return nil
	}
	fmt.Printf("quality gate: FAILED — %s\n", gateResult.Message)
	return fmt.Errorf("%s", gateResult.Message)
}

func runAnalysis(target string, stds analysis.Standards, tc analysis.ToolConfig, ads []analysis.ToolAdapter) (analysis.RunResult, error) {
	runner := analysis.NewRunner(stds, tc, ads)
	if flagDiff != "" {
		runner = runner.WithDiff(flagDiff, gitDiff)
	}
	result, err := runner.Run(context.Background(), target)
	if err != nil {
		return analysis.RunResult{}, fmt.Errorf("running analysis: %w", err)
	}
	return result, nil
}

func buildAndWrite(s runSetup, result analysis.RunResult, graphDir string) (report.Report, string, error) {
	target := s.target
	base, _ := baseline.Load(target)
	delta := baseline.Compute(base, result.Findings)
	existingReview, _ := os.ReadFile(filepath.Join(target, ".coderev", "review.md"))
	var graphJSON string
	if graphDir != "" {
		if data, err := os.ReadFile(filepath.Join(graphDir, "graph.json")); err == nil {
			graphJSON = string(data)
		}
	}
	r := report.Build(report.BuildRequest{
		Target:    target,
		Standards: s.stds,
		StdFile:   s.stdLabel,
		Files:     result.Files,
		Findings:  result.Findings,
		Warnings:  result.Warnings,
		Arch:      architecture.DetectWithGraph(target, graphJSONPath(target, s.tc)),
		Delta:     &delta,
		AIReview:  string(existingReview),
		GraphJSON: graphJSON,
	})
	if flagUpdateBaseline {
		if err := baseline.Save(target, result.Findings); err != nil {
			fmt.Printf("warning: saving baseline failed: %v\n", err)
		} else {
			fmt.Println("baseline: saved current findings as new baseline")
		}
	}
	primaryPath, err := generateAllReports(r, flagOutput, flagFormat, flagRepo, s.tc.SARIF)
	if err != nil {
		return report.Report{}, "", fmt.Errorf("generating report: %w", err)
	}
	return r, primaryPath, nil
}

func postAnnotate(r report.Report, target string) error {
	if !flagAnnotatePR {
		return nil
	}

	repoSlug := flagRepo
	if repoSlug == "" {
		var err error
		repoSlug, err = ghpr.RepoSlug(target)
		if err != nil {
			return fmt.Errorf("--annotate-pr: cannot detect GitHub repo from git remote: %w\nPass --repo owner/repo explicitly", err)
		}
	}

	prNumber := flagPR
	if prNumber == 0 {
		var err error
		prNumber, err = ghpr.OpenPR(target)
		if err != nil {
			return fmt.Errorf("--annotate-pr: cannot detect open PR: %w\nPass --pr <number> explicitly", err)
		}
	}

	if err := ghpr.Annotate(ghpr.AnnotateRequest{Report: r, RepoSlug: repoSlug, PRNumber: prNumber, Target: target}); err != nil {
		fmt.Printf("warning: PR annotation failed: %v\n", err)
	}
	return nil
}

type prReviewReq struct {
	target   string
	tc       analysis.ToolConfig
	repoSlug string
	prNumber int
}

func postReviewToPR(req prReviewReq) {
	body, err := os.ReadFile(filepath.Join(req.target, ".coderev", "review.md"))
	if err != nil {
		return
	}
	repoSlug := req.repoSlug
	if repoSlug == "" {
		repoSlug, err = ghpr.RepoSlug(req.target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: posting AI review to PR: detecting repo: %v\n", err)
			return
		}
	}
	prNumber := req.prNumber
	if prNumber == 0 {
		prNumber, err = ghpr.OpenPR(req.target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: posting AI review to PR: detecting PR: %v\n", err)
			return
		}
	}
	if err := ghpr.PostInlineComment(repoSlug, prNumber, req.target, string(body)); err != nil {
		fmt.Fprintf(os.Stderr, "warning: posting AI review inline to PR: %v\n", err)
		return
	}
}

// generateAllReports writes all three formats (md, html, sarif) into .coderev/.
// If --output or --format are set, only that format is written.
// Returns the path of the primary (markdown) report.
func generateAllReports(r report.Report, outputFlag, formatFlag, repoURI string, sarifCfg analysis.SARIFConfig) (string, error) {
	if outputFlag != "" || formatFlag != "" {
		p := resolveOutputPath(outputFlag, formatFlag)
		return p, generateReport(r, p, formatFlag, repoURI, sarifCfg)
	}
	mdPath := resolveOutputPath("", "markdown")
	if err := report.GenerateMarkdown(r, mdPath); err != nil {
		return "", err
	}
	htmlPath := resolveOutputPath("", "html")
	if err := report.Generate(r, htmlPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: HTML report failed: %v\n", err)
	}
	sarifPath := resolveOutputPath("", "sarif")
	if err := report.GenerateSARIF(r, sarifPath, repoURI, sarifCfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: SARIF report failed: %v\n", err)
	}
	return mdPath, nil
}

func generateReport(r report.Report, outputPath, format, repoURI string, sarifCfg analysis.SARIFConfig) error {
	switch format {
	case "html":
		return report.Generate(r, outputPath)
	case "sarif":
		return report.GenerateSARIF(r, outputPath, repoURI, sarifCfg)
	default:
		return report.GenerateMarkdown(r, outputPath)
	}
}

func resolveTarget(args []string) (string, error) {
	if len(args) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting current directory: %w", err)
		}
		return cwd, nil
	}
	abs, err := filepath.Abs(args[0])
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("target directory not found: %s", abs)
	}
	return abs, nil
}

func resolveStandards(target string) (analysis.Standards, string, error) {
	if flagStandards != "" {
		s, err := config.Load(flagStandards)
		return s, flagStandards, err
	}
	s, err := config.LoadDefaults()
	return s, "built-in defaults", err
}

func resolveToolConfigFile(target string) string {
	if flagConfig != "" {
		return flagConfig
	}
	p, found := config.DiscoverToolConfig(target)
	if !found {
		return ""
	}
	return p
}

func resolveOutputPath(flag, format string) string {
	f := flag
	if f == "" {
		switch format {
		case "html":
			f = filepath.Join(".coderev", "report.html")
		case "sarif":
			f = filepath.Join(".coderev", "report.sarif")
		default:
			f = filepath.Join(".coderev", "report.md")
		}
	}
	if filepath.IsAbs(f) {
		return f
	}
	cwd, _ := os.Getwd()
	p := filepath.Join(cwd, f)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return p
}
