package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/toolmgr"
)

// ── hook templates ────────────────────────────────────────────────────────────

const preCommitHook = `#!/usr/bin/env bash
# installed by: coderev install-hooks
set -euo pipefail
CODEREV=$(command -v coderev 2>/dev/null || true)
if [[ -z "$CODEREV" ]]; then
  CODEREV="$(git rev-parse --show-toplevel)/bin/coderev"
  [[ -x "$CODEREV" ]] || { echo "coderev not found — skipping pre-commit scan"; exit 0; }
fi
ROOT="$(git rev-parse --show-toplevel)"
echo "coderev · pre-commit scan (changed files)…"
if git rev-parse HEAD &>/dev/null; then
  "$CODEREV" --diff HEAD "$ROOT"
else
  "$CODEREV" "$ROOT"
fi
`

const prePushHook = `#!/usr/bin/env bash
# installed by: coderev install-hooks
set -euo pipefail
CODEREV=$(command -v coderev 2>/dev/null || true)
if [[ -z "$CODEREV" ]]; then
  CODEREV="$(git rev-parse --show-toplevel)/bin/coderev"
  [[ -x "$CODEREV" ]] || { echo "coderev not found — skipping pre-push scan"; exit 0; }
fi
echo "coderev · pre-push full scan…"
"$CODEREV" "$(git rev-parse --show-toplevel)"
`

// postCommitHook refreshes the native code graph after each commit. It runs in
// the background so it never slows the commit, and skips silently if coderev is
// unavailable. Mirrors the convenience of an auto-rebuild without coupling to
// the separate graphify tool.
const postCommitHook = `#!/usr/bin/env bash
# installed by: coderev install-hooks
set -euo pipefail
CODEREV=$(command -v coderev 2>/dev/null || true)
if [[ -z "$CODEREV" ]]; then
  CODEREV="$(git rev-parse --show-toplevel)/bin/coderev"
  [[ -x "$CODEREV" ]] || exit 0
fi
ROOT="$(git rev-parse --show-toplevel)"
echo "coderev · post-commit graph rebuild (background)…"
( "$CODEREV" graph "$ROOT" >/dev/null 2>&1 & )
`

// ── commands ──────────────────────────────────────────────────────────────────

var cmdSetup = &cobra.Command{
	Use:   "setup",
	Short: "Install scanner dependencies and git hooks (full onboarding)",
	Long: `Install all external scanner dependencies (gitleaks, semgrep, madge)
and the pre-commit / pre-push git hooks for the current repository.

Run once after cloning to get full coverage:
  coderev setup`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runInstallDeps(cmd, args); err != nil {
			return err
		}
		return runInstallHooks(cmd, args)
	},
}

var cmdInstallHooks = &cobra.Command{
	Use:   "install-hooks",
	Short: "Install pre-commit and pre-push git hooks into .git/hooks/",
	RunE:  runInstallHooks,
}

var cmdInstallDeps = &cobra.Command{
	Use:   "install-deps",
	Short: "Install scanner dependencies: gitleaks, semgrep, madge, and Node/npm npm audit support",
	RunE:  runInstallDeps,
}

// ── hook installation ─────────────────────────────────────────────────────────

func runInstallHooks(_ *cobra.Command, _ []string) error {
	gitDir, err := gitDir()
	if err != nil {
		return fmt.Errorf("not inside a git repository: %w", err)
	}
	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	fmt.Println("installing coderev git hooks…")
	if err := writeHook(filepath.Join(hooksDir, "pre-commit"), preCommitHook); err != nil {
		return err
	}
	if err := writeHook(filepath.Join(hooksDir, "pre-push"), prePushHook); err != nil {
		return err
	}
	if err := writeHook(filepath.Join(hooksDir, "post-commit"), postCommitHook); err != nil {
		return err
	}
	fmt.Println("\ndone — hooks fire automatically on every commit and push.")
	fmt.Println("to skip once: git commit --no-verify")
	return nil
}

func writeHook(path, content string) error {
	if existing, err := os.ReadFile(path); err == nil {
		if !strings.Contains(string(existing), "coderev") {
			fmt.Printf("  ⚠  %s already exists (not by coderev) — skipping\n", filepath.Base(path))
			fmt.Printf("     add the coderev check manually or remove the existing hook first\n")
			return nil
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	fmt.Printf("  ✓  %s\n", path)
	return nil
}

func gitDir() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	if err != nil {
		return "", err
	}
	dir := strings.TrimSpace(string(out))
	if !filepath.IsAbs(dir) {
		cwd, _ := os.Getwd()
		dir = filepath.Join(cwd, dir)
	}
	return dir, nil
}

// ── dependency installation ───────────────────────────────────────────────────

func runInstallDeps(_ *cobra.Command, _ []string) error {
	fmt.Println("── installing scanner dependencies ──────────────────────────────────────")
	tc, err := config.LoadToolConfig("")
	if err != nil {
		return fmt.Errorf("loading tool config: %w", err)
	}
	if err := toolmgr.EnsureAll(toolSources(tc)); err != nil {
		fmt.Printf("  ⚠  some tools could not be installed: %v\n", err)
	}
	fmt.Println("\n── done ─────────────────────────────────────────────────────────────────")
	return nil
}
