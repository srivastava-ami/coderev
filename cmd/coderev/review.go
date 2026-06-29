package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/github"
	"github.com/srivastava-ami/coderev/internal/llm"
)

const reviewFilePerms = 0o644

type reviewFlags struct {
	diffRef string
	postPR  bool
	prNum   int
	repo    string
	output  string
}

type deliverOpts struct {
	output string
	postPR bool
	prNum  int
	repo   string
	tc     analysis.ToolConfig
}

func newReviewCmd() *cobra.Command {
	var f reviewFlags
	cmd := &cobra.Command{
		Use:   "review [directory]",
		Short: "Run an LLM-powered code review on a git diff",
		Long: `Assemble a context-aware review prompt from the git diff, code graph, and
static findings, then send it to the configured LLM provider. Output is
advisory only — it never affects the scan gate (coderev . owns pass/fail).`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview(cmd, args, &f)
		},
	}
	cmd.Flags().StringVar(&f.diffRef, "diff", "HEAD~1", "git ref to diff against")
	cmd.Flags().BoolVar(&f.postPR, "post-pr", false, "post review as upserted PR comment via GitHub")
	cmd.Flags().IntVar(&f.prNum, "pr", 0, "PR number (0 = auto-detect via gh CLI)")
	cmd.Flags().StringVar(&f.repo, "repo", "", "owner/repo slug (auto-detect from git remote)")
	cmd.Flags().StringVar(&f.output, "output", "", "write review to file instead of stdout")
	return cmd
}

func runReview(cmd *cobra.Command, args []string, f *reviewFlags) error {
	target := "."
	if len(args) > 0 {
		target = args[0]
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolving target: %w", err)
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	tc, err := loadReviewConfig(absTarget)
	if err != nil {
		return err
	}
	if !tc.LLM.Enabled {
		fmt.Println("LLM not configured — run: coderev config llm --enable --provider cli")
		return nil
	}
	review, err := buildReview(ctx, tc, f.diffRef, absTarget)
	if err != nil {
		return err
	}
	return deliverReview(ctx, review, resolveDelivery(f, absTarget, tc))
}

func loadReviewConfig(target string) (analysis.ToolConfig, error) {
	tcPath, _ := config.DiscoverToolConfig(target)
	tc, err := config.LoadToolConfig(tcPath)
	if err != nil {
		return analysis.ToolConfig{}, fmt.Errorf("loading tool config: %w", err)
	}
	return tc, nil
}

func buildReview(ctx context.Context, tc analysis.ToolConfig, diffRef, target string) (string, error) {
	hunks, err := gitDiffHunks(ctx, diffRef, target)
	if err != nil {
		return "", err
	}
	neighbors := loadGraphNeighbors(tc, target, changedFileSet(hunks))
	prompt := llm.AssemblePrompt(llm.ReviewContext{
		BaseRef:   diffRef,
		Hunks:     hunks,
		Neighbors: neighbors,
	})
	provider, err := llm.New(tc.LLM)
	if err != nil {
		return "", fmt.Errorf("creating LLM provider: %w", err)
	}
	review, _, err := provider.Complete(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("LLM completion: %w", err)
	}
	return review, nil
}

func resolveDelivery(f *reviewFlags, target string, tc analysis.ToolConfig) deliverOpts {
	opts := deliverOpts{output: f.output, postPR: f.postPR, prNum: f.prNum, repo: f.repo, tc: tc}
	if f.postPR {
		if opts.repo == "" {
			opts.repo = autoDetectRepo(target)
		}
		if opts.prNum == 0 {
			opts.prNum = autoDetectPR(target)
		}
	}
	return opts
}

func deliverReview(ctx context.Context, review string, opts deliverOpts) error {
	if opts.output != "" {
		return os.WriteFile(opts.output, []byte(review), reviewFilePerms)
	}
	if opts.postPR {
		if opts.repo == "" || opts.prNum == 0 {
			return fmt.Errorf("cannot post PR comment: repo=%q pr=%d", opts.repo, opts.prNum)
		}
		client, err := github.New(opts.tc.Github.BaseURL)
		if err != nil {
			return fmt.Errorf("github client: %w", err)
		}
		return client.UpsertCommentContext(ctx, github.PRTarget{Repo: opts.repo, PR: opts.prNum}, review)
	}
	fmt.Print(review)
	return nil
}

func gitDiffHunks(ctx context.Context, ref, target string) ([]llm.DiffHunk, error) {
	out, err := exec.CommandContext(ctx, "git", "-C", target, "diff", "--unified=5", ref).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return llm.ParseDiff(out)
}

func changedFileSet(hunks []llm.DiffHunk) []string {
	seen := make(map[string]bool)
	var result []string
	for _, h := range hunks {
		if !seen[h.File] {
			seen[h.File] = true
			result = append(result, h.File)
		}
	}
	return result
}

func loadGraphNeighbors(cfg analysis.ToolConfig, target string, files []string) []llm.GraphNeighbor {
	outDir := cfg.Graph.OutputDir
	if outDir == "" {
		outDir = ".coderev/graph"
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(target, outDir)
	}
	data, err := os.ReadFile(filepath.Join(outDir, "graph.json"))
	if err != nil {
		return nil
	}
	neighbors, _ := llm.GraphNeighborhood(data, files, 2)
	return neighbors
}

// autoDetectRepo parses the git remote URL to extract "owner/repo" without
// hardcoding a specific host — works for any HTTPS or SSH remote.
func autoDetectRepo(target string) string {
	out, err := exec.Command("git", "-C", target, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	raw := strings.TrimSuffix(strings.TrimSpace(string(out)), ".git")
	// SSH format: user@host:owner/repo (colon before slash, no // in prefix)
	if i := strings.Index(raw, ":"); i > 0 && !strings.Contains(raw[:i], "/") {
		return raw[i+1:]
	}
	// HTTPS format: https://host/owner/repo
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return strings.TrimPrefix(u.Path, "/")
	}
	return ""
}

func autoDetectPR(target string) int {
	cmd := exec.Command("gh", "pr", "view", "--json", "number", "-q", ".number")
	cmd.Dir = target
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}
