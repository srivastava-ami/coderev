package toolmgr

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ToolDir returns ~/.coderev/tools/, creating it if needed.
func ToolDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	dir := filepath.Join(home, ".coderev", "tools")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return dir, nil
}

// ToolPath returns the full path to a tool binary under ToolDir().
func ToolPath(name string) (string, error) {
	dir, err := ToolDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

// Exists reports whether a named tool binary exists in ToolDir() or on $PATH.
func Exists(name string) bool {
	if p, err := ToolPath(name); err == nil {
		if _, err2 := os.Stat(p); err2 == nil {
			return true
		}
	}
	_, err := exec.LookPath(name)
	return err == nil
}

// EnsureAll downloads and installs all external scanner tools that are not
// yet available. Prints progress to stderr.
func EnsureAll() error {
	var errs []string
	for _, t := range tools {
		if Exists(t.Name) {
			continue
		}
		fmt.Fprintf(os.Stderr, "  → downloading %s…\n", t.Name)
		if err := t.Install(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", t.Name, err))
			fmt.Fprintf(os.Stderr, "  ✗  %s: %v\n", t.Name, err)
		} else {
			fmt.Fprintf(os.Stderr, "  ✓  %s installed\n", t.Name)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("tool installation errors:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}

// tool describes an external scanner binary that toolmgr can download.
type tool struct {
	Name    string
	Binary  string
	Install func() error
}

var semgrepReleaseURL = "https://github.com/semgrep/semgrep/releases/download/v%s/semgrep-v%s-ubuntu-16.04.tgz"

var tools = []tool{
	{
		Name:   "gitleaks",
		Binary: "gitleaks",
		Install: func() error {
			return ensureGitleaks()
		},
	},
	{
		Name:   "semgrep",
		Binary: "semgrep",
		Install: func() error {
			return ensureSemgrep()
		},
	},
	{
		Name:   "madge",
		Binary: "madge",
		Install: func() error {
			return ensureMadge()
		},
	},
}

func ensureGitleaks() error {
	ver := "8.18.2"
	url := gitleaksURL(ver)
	return downloadTGZ(url, "gitleaks", "gitleaks")
}

func gitleaksURL(ver string) string {
	os_ := runtime.GOOS
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return fmt.Sprintf(
		"https://github.com/gitleaks/gitleaks/releases/download/v%s/gitleaks_%s_%s_%s.tar.gz",
		ver, ver, os_, arch,
	)
}

func ensureSemgrep() error {
	// pipx works everywhere and isolates the install correctly
	if onPATH("pipx") {
		return runVisible("pipx", "install", "semgrep")
	}
	// macOS system Python is PEP 668 externally managed — pip3 will always fail;
	// use brew which has semgrep as a formula
	if runtime.GOOS == "darwin" {
		if onPATH("brew") {
			return runVisible("brew", "install", "semgrep")
		}
		return fmt.Errorf("install semgrep manually: brew install semgrep  OR  brew install pipx && pipx install semgrep")
	}
	// Linux: pip3 --user works, static binary as last resort
	if onPATH("pip3") {
		return runVisible("pip3", "install", "--user", "semgrep")
	}
	if onPATH("brew") {
		return runVisible("brew", "install", "semgrep")
	}
	ver := "1.69.0"
	url := fmt.Sprintf(semgrepReleaseURL, ver, ver)
	return downloadTGZ(url, "semgrep", "semgrep")
}

func ensureMadge() error {
	if !onPATH("npm") {
		return fmt.Errorf("npm not found — install Node.js: https://nodejs.org")
	}
	dir, err := ToolDir()
	if err != nil {
		return err
	}
	target := filepath.Join(dir, "madge")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	instDir := filepath.Join(dir, "madge-install")
	if err := os.MkdirAll(instDir, 0o755); err != nil {
		return err
	}
	cmd := exec.Command("npm", "install", "madge", "--prefix", instDir, "--no-save")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install madge: %w", err)
	}
	bins := []string{"madge", "madge.js"}
	for _, b := range bins {
		src := filepath.Join(instDir, "node_modules", ".bin", b)
		if _, err := os.Stat(src); err == nil {
			_ = os.Remove(target)
			return os.Symlink(src, target)
		}
	}
	return fmt.Errorf("madge binary not found after npm install")
}

// downloadTGZ fetches a tar.gz from url and extracts binaryName to ToolDir().
func downloadTGZ(url, toolName, binaryName string) error {
	dir, err := ToolDir()
	if err != nil {
		return err
	}
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("downloading %s: %w", toolName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %s: HTTP %d", toolName, resp.StatusCode)
	}
	return extractFromTGZ(resp.Body, toolName, binaryName, dir)
}

func extractFromTGZ(r io.Reader, toolName, binaryName, dir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("decompressing %s: %w", toolName, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar for %s: %w", toolName, err)
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		return writeBinary(tr, filepath.Join(dir, binaryName))
	}
	return fmt.Errorf("%s: binary %q not found in archive", toolName, binaryName)
}

const maxBinaryBytes = 256 << 20 // 256 MiB — guard against malicious/corrupt archives

func writeBinary(r io.Reader, dest string) error {
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, io.LimitReader(r, maxBinaryBytes))
	return err
}

func onPATH(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runVisible(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
