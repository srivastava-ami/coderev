package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/llm"
	"github.com/srivastava-ami/coderev/internal/toolmgr"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

var (
	flagStandards      string
	flagOutput         string
	flagConfig         string
	flagFormat         string
	flagRepo           string
	flagPR             int
	flagAnnotatePR     bool
	flagDiff           string
	flagUpdateBaseline bool
	flagJSON           bool
	flagGate           string
	flagPluginDir      string
	flagReview         bool
	flagFullReview     bool
)

func main() {
	root := &cobra.Command{
		Use:   "coderev [directory]",
		Short: "Local code review against built-in coding standards",
		Long: `coderev analyses a codebase against built-in coding standards and
produces an interactive Markdown report (default) or a self-contained HTML
report — no LLMs, no cloud, no network.

If [directory] is omitted the current directory is used.
Standards are built into the binary. Run coderev . with no configuration needed.`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		RunE:    run,
		Example: `  coderev                                 # scan cwd, write coderev-report.md
  coderev /path/to/repo                   # scan specific repo
  coderev --format html                   # HTML report instead of Markdown
  coderev --output ./reports/review.md    # custom output path`,
	}

	root.Flags().StringVar(&flagStandards, "standards", "", "path to code_review_standards.toml (escape hatch — usually not needed)")
	root.Flags().MarkHidden("standards")
	root.Flags().StringVar(&flagOutput, "output", "", "output file path (default depends on --format)")
	root.Flags().StringVar(&flagConfig, "config", "", "path to tool_config.toml (auto-discovered if omitted)")
	root.Flags().StringVar(&flagFormat, "format", "", "output format: markdown (default), html, sarif")
	root.Flags().StringVar(&flagRepo, "repo", "", "owner/repo slug — auto-detected from git remote (e.g. acme/my-repo)")
	root.Flags().IntVar(&flagPR, "pr", 0, "PR number — auto-detected from gh CLI if omitted")
	root.Flags().BoolVar(&flagAnnotatePR, "annotate-pr", false, "post findings as inline GitHub PR comments (requires gh CLI)")
	root.Flags().StringVar(&flagDiff, "diff", "", "incremental mode: only scan files changed since this git ref (e.g. HEAD~1, main)")
	root.Flags().BoolVar(&flagUpdateBaseline, "update-baseline", false, "save current findings as new baseline in .coderev/baseline.json")
	root.Flags().BoolVar(&flagJSON, "json", false, "output findings as JSON instead of markdown")
	root.Flags().StringVar(&flagGate, "gate", "", "path to .coderev-gate.toml for quality gate check")
	root.Flags().StringVar(&flagPluginDir, "plugin-dir", "", "custom plugin directory (default: ~/.config/coderev/plugins)")
	root.Flags().BoolVar(&flagReview, "review", false, "send assembled prompt to configured LLM and write .coderev/review.md")
	root.Flags().BoolVar(&flagFullReview, "full-review", false, "review every file in the code graph via LLM (graph context only, no diff required)")

	root.AddCommand(cmdSetup, cmdInstallHooks, cmdInstallDeps, cmdPlugin, cmdGraph, cmdConfig, cmdAsk, newReviewCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	s := setupRun(args)
	if s.err != nil {
		return s.err
	}
	// Download URLs come from the tool config (loaded above), not from hardcoded
	// literals — so EnsureAll runs after the config is resolved.
	if err := toolmgr.EnsureAll(toolSources(s.tc)); err != nil {
		fmt.Fprintf(os.Stderr, "⚠  some external tools could not be installed: %v\n", err)
	}
	ads := buildAdapters(s.stds, s.tc)
	printStartup(s.target, s.stdLabel, ads)
	discoverAndPrintPlugins()

	result, err := runAnalysis(s.target, s.stds, s.tc, ads)
	if err != nil {
		return err
	}

	if flagJSON {
		return jsonRun(result)
	}

	return stdRun(s, result)
}

// toolSources extracts the download-URL templates from the tool config for toolmgr.
func toolSources(tc analysis.ToolConfig) toolmgr.Sources {
	return toolmgr.Sources{
		GitleaksURL: tc.Adapters.Gitleaks.DownloadURL,
		SemgrepURL:  tc.Adapters.Semgrep.DownloadURL,
	}
}

type runSetup struct {
	target   string
	stds     analysis.Standards
	stdLabel string
	tc       analysis.ToolConfig
	err      error
}

func setupRun(args []string) runSetup {
	var s runSetup
	s.target, s.err = resolveTarget(args)
	if s.err != nil {
		return s
	}
	s.stds, s.stdLabel, s.err = resolveStandards(s.target)
	if s.err != nil {
		return s
	}
	s.tc, s.err = config.LoadToolConfig(resolveToolConfigFile(s.target))
	if s.err != nil {
		s.err = fmt.Errorf("loading tool config: %w", s.err)
	}
	return s
}

func jsonRun(result analysis.RunResult) error {
	gateResult, err := resolveGate(result.Findings)
	if err != nil {
		return err
	}
	return writeJSONOutput(result, gateResult)
}

func stdRun(s runSetup, result analysis.RunResult) error {
	gateResult, err := resolveGate(result.Findings)
	if err != nil {
		return err
	}
	r, outputPath, err := buildAndWrite(s, result)
	if err != nil {
		return err
	}
	printSummary(r.Summary, result.Warnings, outputPath)
	if err := printGateResult(gateResult); err != nil {
		return err
	}
	if err := postAnnotate(r, s.target); err != nil {
		return err
	}
	graphDir := buildGraphInline(s.target, s.tc)
	rc := buildReviewContext(s.target, result.Findings, graphDir)
	if err := writePromptFile(s.target, rc); err != nil {
		fmt.Fprintf(os.Stderr, "warning: writing prompt file: %v\n", err)
	}
	if flagFullReview {
		if err := runFullGraphReview(context.Background(), llmReviewReq{target: s.target, tc: s.tc, rc: llm.ReviewContext{Findings: result.Findings}}, graphDir); err != nil {
			return err
		}
	} else if flagReview {
		if err := maybeSendToLLM(context.Background(), llmReviewReq{target: s.target, tc: s.tc, rc: rc}); err != nil {
			return err
		}
	} else {
		return nil
	}
	return refreshHTMLWithReview(r, s.target)
}



