package analysis

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)
type AdapterWarning struct {
	Adapter string
	Reason  string
}

// RunResult is the complete output of a scan.
type RunResult struct {
	Files    []FileInfo
	Findings []Finding
	Warnings []AdapterWarning
}

// DiffService abstracts the SCM operation to list files changed since a ref.
type DiffService interface {
	ChangedFiles(target, baseRef string) (map[string]bool, error)
}

// Runner orchestrates file walking, adapter dispatch, and result merging.
type Runner struct {
	stds     Standards
	tc       ToolConfig
	adapters []ToolAdapter
	diffSvc  DiffService // nil means no diff filtering
	baseRef  string
}

func NewRunner(stds Standards, tc ToolConfig, ads []ToolAdapter) *Runner {
	return &Runner{stds: stds, tc: tc, adapters: ads}
}

// WithDiff returns a copy of the runner that only analyses files changed since ref.
func (r *Runner) WithDiff(ref string, svc DiffService) *Runner {
	c := *r
	c.baseRef = ref
	c.diffSvc = svc
	return &c
}

// runSession holds shared mutable state for a single Run call.
type runSession struct {
	mu       sync.Mutex
	target   string
	files    []FileInfo
	findings []Finding
	warnings []AdapterWarning
}

// Run walks target, dispatches to each adapter, and merges all findings.
func (r *Runner) Run(ctx context.Context, target string) (RunResult, error) {
	files, err := walkFiles(target)
	if err != nil {
		return RunResult{}, fmt.Errorf("walking target: %w", err)
	}
	if r.diffSvc != nil && r.baseRef != "" {
		changed, err := r.diffSvc.ChangedFiles(target, r.baseRef)
		if err != nil {
			return RunResult{}, fmt.Errorf("resolving diff against %s: %w", r.baseRef, err)
		}
		files = filterFiles(files, changed)
	}

	sess := &runSession{target: target, files: files}
	var wg sync.WaitGroup
	for _, ad := range r.adapters {
		ad := ad
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.dispatchAdapter(ctx, ad, sess)
		}()
	}
	wg.Wait()
	filtered := applyExceptions(dedup(sess.findings), r.stds.Exceptions)
	return RunResult{Files: files, Findings: filtered, Warnings: sess.warnings}, nil
}

func (r *Runner) dispatchAdapter(ctx context.Context, ad ToolAdapter, sess *runSession) {
	if !ad.IsAvailable() {
		sess.mu.Lock()
		sess.warnings = append(sess.warnings, AdapterWarning{
			Adapter: ad.Name(),
			Reason:  "binary not found or prerequisites not met — skipped",
		})
		sess.mu.Unlock()
		return
	}

	req := RunRequest{Target: sess.target, Files: sess.files, RuleIDs: ad.Capabilities()}
	got, err := ad.Run(ctx, req)
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if err != nil {
		sess.warnings = append(sess.warnings, AdapterWarning{Adapter: ad.Name(), Reason: err.Error()})
	}
	sess.findings = append(sess.findings, got...)
}

// walkFiles collects all source files under target, skipping well-known noise dirs.
func walkFiles(target string) ([]FileInfo, error) {
	skipDirs := map[string]bool{
		"node_modules": true, ".git": true, "dist": true, "build": true,
		".nx": true, "coverage": true, ".cache": true, "vendor": true,
		"__pycache__": true, "target": true, ".cargo": true,
	}

	var files []FileInfo
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		lang, ok := langForExt(filepath.Ext(path))
		if !ok {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		files = append(files, FileInfo{
			Path:     path,
			Language: lang,
			Lines:    strings.Count(string(content), "\n") + 1,
			Content:  content,
		})
		return nil
	})
	return files, err
}

func langForExt(ext string) (Language, bool) {
	lang, ok := ExtToLanguage[ext]
	return lang, ok
}

func dedupKey(f Finding) string {
	return fmt.Sprintf("%s|%s|%d", f.Rule, f.File, f.Line)
}

// dedup removes identical (Rule, File, Line) tuples, keeping the first.
func dedup(findings []Finding) []Finding {
	seen := make(map[string]bool, len(findings))
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		key := dedupKey(f)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, f)
	}
	return out
}

// applyExceptions removes findings that match a declared exception entry.
func applyExceptions(findings []Finding, exceptions []Exception) []Finding {
	if len(exceptions) == 0 {
		return findings
	}
	out := findings[:0]
	for _, f := range findings {
		if !matchesException(f, exceptions) {
			out = append(out, f)
		}
	}
	return out
}

func matchesException(f Finding, exceptions []Exception) bool {
	for _, ex := range exceptions {
		if ex.Rule == "" || ex.FileOrModule == "" {
			continue
		}
		if ex.Rule != f.Rule {
			continue
		}
		if matchExceptionPath(f.File, ex.FileOrModule) {
			return true
		}
	}
	return false
}
func matchExceptionPath(file, pattern string) bool {
	// glob match — pattern contains * or ?
	if strings.ContainsAny(pattern, "*?") {
		matched, _ := filepath.Match(pattern, file)
		if matched {
			return true
		}
		// also try matching against just the filename
		_, base := filepath.Split(file)
		matched, _ = filepath.Match(pattern, base)
		return matched
	}

	// exact suffix match anchored at a separator boundary
	if strings.HasSuffix(file, pattern) {
		suffixStart := len(file) - len(pattern)
		if suffixStart == 0 || file[suffixStart-1] == '/' || file[suffixStart-1] == '\\' {
			return true
		}
	}

	return false
}

func filterFiles(files []FileInfo, changed map[string]bool) []FileInfo {
	out := make([]FileInfo, 0, len(changed))
	for _, f := range files {
		if changed[f.Path] {
			out = append(out, f)
		}
	}
	return out
}
