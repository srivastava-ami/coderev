package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/architecture"
	"github.com/srivastava-ami/coderev/internal/baseline"
	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/output/ghpr"
	"github.com/srivastava-ami/coderev/internal/plugin"
	"github.com/srivastava-ami/coderev/internal/report"
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
	flagPluginDir      string
)

func main() {
	root := &cobra.Command{
		Use:   "coderev [directory]",
		Short: "Local code review against code_review_standards.toml",
		Long: `coderev analyses a codebase against the rules defined in
code_review_standards.toml and produces an interactive Markdown report
(default) or a self-contained HTML report — no LLMs, no cloud, no network.

If [directory] is omitted the current directory is used.
Standards and tool-config files are auto-discovered (target dir → cwd → ~/.config/coderev/).`,
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		RunE:    run,
		Example: `  coderev                                 # scan cwd, write coderev-report.md
  coderev /path/to/repo                   # scan specific repo
  coderev --format html                   # HTML report instead of Markdown
  coderev --output ./reports/review.md    # custom output path`,
	}

	root.Flags().StringVar(&flagStandards, "standards", "", "path to code_review_standards.toml (auto-discovered, falls back to built-in defaults)")
	root.Flags().StringVar(&flagOutput, "output", "", "output file path (default depends on --format)")
	root.Flags().StringVar(&flagConfig, "config", "", "path to tool_config.toml (auto-discovered if omitted)")
	root.Flags().StringVar(&flagFormat, "format", "markdown", "output format: markdown (default), html, sarif")
	root.Flags().StringVar(&flagRepo, "repo", "", "owner/repo slug — auto-detected from git remote (e.g. acme/my-repo)")
	root.Flags().IntVar(&flagPR, "pr", 0, "PR number — auto-detected from gh CLI if omitted")
	root.Flags().BoolVar(&flagAnnotatePR, "annotate-pr", false, "post findings as inline GitHub PR comments (requires gh CLI)")
	root.Flags().StringVar(&flagDiff, "diff", "", "incremental mode: only scan files changed since this git ref (e.g. HEAD~1, main)")
	root.Flags().BoolVar(&flagUpdateBaseline, "update-baseline", false, "save current findings as new baseline in .coderev/baseline.json")
	root.Flags().StringVar(&flagPluginDir, "plugin-dir", "", "custom plugin directory (default: ~/.config/coderev/plugins)")

	root.AddCommand(cmdSetup, cmdInstallHooks, cmdInstallDeps, cmdPlugin)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	target, err := resolveTarget(args)
	if err != nil {
		return err
	}
	stds, stdLabel, err := resolveStandards(target)
	if err != nil {
		return err
	}
	tc, err := config.LoadToolConfig(resolveToolConfigFile(target))
	if err != nil {
		return fmt.Errorf("loading tool config: %w", err)
	}

	ads := buildAdapters(stds, tc)
	printStartup(target, stdLabel, ads)

	pluginDir := flagPluginDir
	if pluginDir == "" {
		var err error
		pluginDir, err = getPluginDir()
		if err != nil {
			return fmt.Errorf("resolving plugin directory: %w", err)
		}
	}
	plugins, err := plugin.DiscoverPlugins(pluginDir)
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}
	if len(plugins) > 0 {
		fmt.Printf("          plugins: %d found\n", len(plugins))
		for _, p := range plugins {
			fmt.Printf("            • %s %s (%s)\n", p.Manifest.Name, p.Manifest.Version, p.Manifest.Description)
		}
	}

	result, err := runAnalysis(target, stds, tc, ads)
	if err != nil {
		return err
	}

	r, outputPath, err := buildAndWrite(target, stdLabel, stds, result)
	if err != nil {
		return err
	}
	printSummary(r.Summary, result.Warnings, outputPath)
	return postAnnotate(r, target)
}

func runAnalysis(target string, stds config.Standards, tc config.ToolConfig, ads []analysis.ToolAdapter) (analysis.RunResult, error) {
	runner := analysis.NewRunner(stds, tc, ads)
	if flagDiff != "" {
		runner = runner.WithDiff(flagDiff)
	}
	result, err := runner.Run(context.Background(), target)
	if err != nil {
		return analysis.RunResult{}, fmt.Errorf("running analysis: %w", err)
	}
	return result, nil
}

func buildAndWrite(target, stdFile string, stds config.Standards, result analysis.RunResult) (report.Report, string, error) {
	base, _ := baseline.Load(target)
	delta := baseline.Compute(base, result.Findings)
	r := report.Build(report.BuildRequest{
		Target:    target,
		Standards: stds,
		StdFile:   stdFile,
		Files:     result.Files,
		Findings:  result.Findings,
		Warnings:  result.Warnings,
		Arch:      architecture.Detect(target),
		Delta:     &delta,
	})
	if flagUpdateBaseline {
		if err := baseline.Save(target, result.Findings); err != nil {
			fmt.Printf("warning: saving baseline failed: %v\n", err)
		} else {
			fmt.Println("baseline: saved current findings as new baseline")
		}
	}
	outputPath := resolveOutputPath(flagOutput, flagFormat)
	if err := generateReport(r, outputPath, flagFormat, flagRepo); err != nil {
		return report.Report{}, "", fmt.Errorf("generating report: %w", err)
	}
	return r, outputPath, nil
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

func generateReport(r report.Report, outputPath, format, repoURI string) error {
	switch format {
	case "html":
		return report.Generate(r, outputPath)
	case "sarif":
		return report.GenerateSARIF(r, outputPath, repoURI)
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

// resolveStandards loads standards from: --standards flag → target dir →
// ~/.config/coderev/ → built-in defaults embedded in the binary.
// Returns the standards and a human-readable label for the startup line.
func resolveStandards(target string) (config.Standards, string, error) {
	if flagStandards != "" {
		s, err := config.Load(flagStandards)
		return s, flagStandards, err
	}
	if p, ok := config.DiscoverStandards(target); ok {
		s, err := config.Load(p)
		return s, p, err
	}
	s, err := config.LoadDefaults()
	return s, "built-in defaults", err
}

func resolveToolConfigFile(target string) string {
	if flagConfig != "" {
		return flagConfig
	}
	p, _ := config.DiscoverToolConfig(target)
	return p
}

func resolveOutputPath(flag, format string) string {
	f := flag
	if f == "" {
		switch format {
		case "html":
			f = "coderev-report.html"
		case "sarif":
			f = "coderev-report.sarif"
		default:
			f = "coderev-report.md"
		}
	}
	if filepath.IsAbs(f) {
		return f
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, f)
}

