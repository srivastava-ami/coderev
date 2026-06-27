package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// builtinSkipDirs are always skipped during file discovery, even when there is
// no .gitignore. They are universal build/VCS noise, never source.
var builtinSkipDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true,
	".nx": true, "coverage": true, ".cache": true, "vendor": true,
	"__pycache__": true, "target": true, ".cargo": true,
}

// Ignorer decides which paths to skip during file discovery. Policy: adhere to
// the repo-root .gitignore (resolved via git, so the match is exact) when it
// exists; if there is no root .gitignore — or git is unavailable — fall back to
// builtinSkipDirs only and otherwise scan everything.
type Ignorer struct {
	ignored map[string]bool // absolute paths git reports as ignored
}

// NewIgnorer builds an Ignorer for the given scan root.
func NewIgnorer(root string) *Ignorer {
	ig := &Ignorer{ignored: map[string]bool{}}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ig
	}
	if _, err := os.Stat(filepath.Join(absRoot, ".gitignore")); err != nil {
		return ig // no root .gitignore → scan everything (minus builtins)
	}
	// List ignored files and (collapsed) ignored directories, per .gitignore.
	out, err := exec.Command("git", "-C", absRoot,
		"ls-files", "--others", "--ignored", "--exclude-standard", "--directory").Output()
	if err != nil {
		return ig // git unavailable → builtins only
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		ig.ignored[filepath.Join(absRoot, strings.TrimSuffix(line, "/"))] = true
	}
	return ig
}

// SkipDir reports whether a directory (its path and base name) should be skipped.
func (ig *Ignorer) SkipDir(path, name string) bool {
	return builtinSkipDirs[name] || ig.isIgnored(path)
}

// SkipFile reports whether a file should be skipped.
func (ig *Ignorer) SkipFile(path string) bool { return ig.isIgnored(path) }

func (ig *Ignorer) isIgnored(path string) bool {
	if len(ig.ignored) == 0 {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return ig.ignored[abs]
}
