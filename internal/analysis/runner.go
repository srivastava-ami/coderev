package analysis

import (
	"context"
	"fmt"
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
	// Content-free file list: paths/metadata only, no file bytes. Target adapters
	// and imports work from this directly; the content-consuming adapters
	// (treesitter/secrets) get bytes streamed in bounded batches below — so peak
	// memory is one batch, not the whole tree.
	files, err := ListSourceFiles(target)
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

	var contentAds, targetAds []ToolAdapter
	for _, ad := range r.adapters {
		if isContentAdapter(ad.Name()) {
			contentAds = append(contentAds, ad)
		} else {
			targetAds = append(targetAds, ad)
		}
	}

	r.runContentAdapters(ctx, sess, contentAds)
	r.runTargetAdapters(ctx, sess, targetAds)
	filtered := applyExceptions(dedup(sess.findings), r.stds.Exceptions)
	return RunResult{Files: files, Findings: filtered, Warnings: sess.warnings}, nil
}

// isContentAdapter reports adapters that need per-file Content fed to them.
// imports is NOT here: it reads file bytes by path itself, so it works from the
// content-free list like the target adapters.
func isContentAdapter(name string) bool {
	switch name {
	case "treesitter", "secrets":
		return true
	}
	return false
}

// runContentAdapters streams file content in bounded batches and runs the
// content adapters per batch, so only one batch of bytes is resident at a time.
// Batches are filtered to sess.files so diff-mode (and .gitignore) filtering is
// honoured.
func (r *Runner) runContentAdapters(ctx context.Context, sess *runSession, ads []ToolAdapter) {
	avail := r.availableAdapters(sess, ads)
	if len(avail) == 0 {
		return
	}
	allowed := make(map[string]bool, len(sess.files))
	for _, f := range sess.files {
		allowed[f.Path] = true
	}
	_ = StreamSourceFiles(sess.target, r.tc.Scan.BatchSize, func(batch []FileInfo) error {
		kept := batch[:0]
		for _, f := range batch {
			if allowed[f.Path] {
				kept = append(kept, f)
			}
		}
		if len(kept) > 0 {
			for _, ad := range avail {
				got, err := ad.Run(ctx, RunRequest{Target: sess.target, Files: kept, RuleIDs: ad.Capabilities()})
				sess.addResult(ad, got, err)
			}
		}
		return nil
	})
}

func (r *Runner) runTargetAdapters(ctx context.Context, sess *runSession, ads []ToolAdapter) {
	avail := r.availableAdapters(sess, ads)
	var wg sync.WaitGroup
	for _, ad := range avail {
		ad := ad
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.dispatchAdapter(ctx, ad, sess)
		}()
	}
	wg.Wait()
}

// availableAdapters returns the available adapters, recording one warning per
// unavailable adapter (so warnings match the non-streamed behaviour).
func (r *Runner) availableAdapters(sess *runSession, ads []ToolAdapter) []ToolAdapter {
	out := make([]ToolAdapter, 0, len(ads))
	for _, ad := range ads {
		if ad.IsAvailable() {
			out = append(out, ad)
			continue
		}
		sess.mu.Lock()
		sess.warnings = append(sess.warnings, AdapterWarning{
			Adapter: ad.Name(),
			Reason:  "binary not found or prerequisites not met — skipped",
		})
		sess.mu.Unlock()
	}
	return out
}

// addResult records an adapter's findings (and a warning on error) under lock.
func (s *runSession) addResult(ad ToolAdapter, got []Finding, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.warnings = append(s.warnings, AdapterWarning{Adapter: ad.Name(), Reason: err.Error()})
	}
	s.findings = append(s.findings, got...)
}

// dispatchAdapter runs a target adapter over the (content-free) session files.
func (r *Runner) dispatchAdapter(ctx context.Context, ad ToolAdapter, sess *runSession) {
	got, err := ad.Run(ctx, RunRequest{Target: sess.target, Files: sess.files, RuleIDs: ad.Capabilities()})
	sess.addResult(ad, got, err)
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
