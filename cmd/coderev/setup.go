package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
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
	Short: "Install scanner dependencies: gitleaks (required), semgrep (required), madge (recommended)",
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

	hasBrew := commandExists("brew")
	hasPip3 := commandExists("pip3")
	hasPipx := commandExists("pipx")
	hasNPM := commandExists("npm")

	fmt.Println("\n  [gitleaks] secrets & credential scanning (required)")
	if err := installGitleaks(hasBrew); err != nil {
		fmt.Printf("  ✗  gitleaks: %v\n", err)
	}

	fmt.Println("\n  [semgrep] OWASP security pattern scanning (required)")
	if err := installSemgrep(hasBrew, hasPipx, hasPip3); err != nil {
		fmt.Printf("  ✗  semgrep: %v\n", err)
	}

	fmt.Println("\n  [madge] circular dependency detection for NX/TypeScript (recommended)")
	if err := installMadge(hasNPM); err != nil {
		fmt.Printf("  ⚠  madge: %v\n", err)
	}

	fmt.Println("\n── done ─────────────────────────────────────────────────────────────────")
	return nil
}

func installGitleaks(hasBrew bool) error {
	if commandExists("gitleaks") {
		v, _ := runOutput("gitleaks", "version")
		fmt.Printf("  ✓  already installed: %s\n", strings.TrimSpace(v))
		return nil
	}
	if hasBrew {
		return brewInstall("gitleaks")
	}
	return installGitleaksFromRelease()
}

func installSemgrep(hasBrew, hasPipx, hasPip3 bool) error {
	if commandExists("semgrep") {
		v, _ := runOutput("semgrep", "--version")
		fmt.Printf("  ✓  already installed: %s\n", strings.TrimSpace(v))
		return nil
	}
	if hasBrew {
		return brewInstall("semgrep")
	}
	if hasPipx {
		return runVisible("pipx", "install", "semgrep")
	}
	if hasPip3 {
		return runVisible("pip3", "install", "--user", "semgrep")
	}
	return fmt.Errorf("no installer found (brew / pipx / pip3)\n     install manually: https://semgrep.dev/docs/getting-started/")
}

func installMadge(hasNPM bool) error {
	if commandExists("madge") {
		v, _ := runOutput("madge", "--version")
		fmt.Printf("  ✓  already installed: %s\n", strings.TrimSpace(v))
		return nil
	}
	if hasNPM {
		return runVisible("npm", "install", "-g", "madge")
	}
	return fmt.Errorf("npm not found — install Node.js then run: npm install -g madge")
}

func installGitleaksFromRelease() error {
	os_ := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	// resolve latest version via GitHub API
	fetch := fetchCmd()
	versionJSON, err := exec.Command(fetch[0], append(fetch[1:],
		"https://api.github.com/repos/gitleaks/gitleaks/releases/latest")...).Output()
	if err != nil {
		return fmt.Errorf("cannot reach GitHub releases: %w", err)
	}
	version := extractJSONField(string(versionJSON), "tag_name")
	version = strings.TrimPrefix(version, "v")
	if version == "" {
		return fmt.Errorf("could not determine latest gitleaks version")
	}
	tarball := fmt.Sprintf("gitleaks_%s_%s_%s.tar.gz", version, os_, arch)
	url := fmt.Sprintf("https://github.com/gitleaks/gitleaks/releases/download/v%s/%s", version, tarball)

	fmt.Printf("  → downloading gitleaks %s from GitHub releases…\n", version)
	tmpDir, err := os.MkdirTemp("", "gitleaks-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// download and extract
	dlArgs := append(fetch[1:], url)
	dl := exec.Command(fetch[0], dlArgs...)
	dl.Dir = tmpDir
	if fetch[0] == "curl" {
		dl.Args = append([]string{"curl", "-fsSL", url, "|", "tar", "-xz", "-C", tmpDir}, []string{}...)
	}
	// use shell to pipe curl/wget into tar
	shell := fmt.Sprintf("%s %s | tar -xz -C %s gitleaks",
		strings.Join(append(fetch, url), " "), "", tmpDir)
	if err := exec.Command("sh", "-c", shell).Run(); err != nil {
		return fmt.Errorf("downloading gitleaks: %w", err)
	}
	dest := "/usr/local/bin/gitleaks"
	if err := copyFile(filepath.Join(tmpDir, "gitleaks"), dest, 0o755); err != nil {
		return fmt.Errorf("installing gitleaks to %s: %w", dest, err)
	}
	fmt.Printf("  ✓  gitleaks %s installed → %s\n", version, dest)
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func brewInstall(pkg string) error {
	fmt.Printf("  → brew install %s\n", pkg)
	return runVisible("brew", "install", pkg)
}

func runVisible(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return string(out), err
}

func fetchCmd() []string {
	if commandExists("curl") {
		return []string{"curl", "-fsSL"}
	}
	return []string{"wget", "-qO-"}
}

func extractJSONField(json, field string) string {
	key := fmt.Sprintf(`"%s":`, field)
	idx := strings.Index(json, key)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(json[idx+len(key):])
	rest = strings.TrimPrefix(rest, `"`)
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}
