package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	coverage "github.com/srivastava-ami/coderev/internal/adapters/coverage"
	depcve "github.com/srivastava-ami/coderev/internal/adapters/depcve"
	gitleaks "github.com/srivastava-ami/coderev/internal/adapters/gitleaks"
	importsadapter "github.com/srivastava-ami/coderev/internal/adapters/imports"
	madge "github.com/srivastava-ami/coderev/internal/adapters/madge"
	npmaudit "github.com/srivastava-ami/coderev/internal/adapters/npmaudit"
	script "github.com/srivastava-ami/coderev/internal/adapters/script"
	secrets "github.com/srivastava-ami/coderev/internal/adapters/secrets"
	semgrep "github.com/srivastava-ami/coderev/internal/adapters/semgrep"
	tsadapter "github.com/srivastava-ami/coderev/internal/adapters/treesitter"
	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/toolmgr"
)

// adapterEntry pairs an enabled flag with a lazy constructor so buildAdapters
// can stay a simple table + loop instead of a long if-chain.
type adapterEntry struct {
	enabled bool
	make    func() analysis.ToolAdapter
}

func buildAdapters(stds analysis.Standards, tc analysis.ToolConfig) []analysis.ToolAdapter {
	// Order matters: native (pure-Go, zero-dependency) adapters run first so their
	// findings take precedence; any enabled external tool is additive enrichment,
	// deduped by the runner.
	entries := []adapterEntry{
		{tc.Adapters.TreeSitter.Enabled, func() analysis.ToolAdapter { return tsadapter.New(stds) }},
		{tc.Adapters.DepCve.Enabled, func() analysis.ToolAdapter { return depcve.New(tc.Adapters.DepCve.SnapshotURL) }},
		{tc.Adapters.Secrets.Enabled, func() analysis.ToolAdapter { return secrets.New() }},
		{tc.Adapters.Imports.Enabled, func() analysis.ToolAdapter { return importsadapter.New() }},
		{tc.Adapters.Semgrep.Enabled, func() analysis.ToolAdapter { return semgrep.New(resolveTool("semgrep", tc.Adapters.Semgrep.Binary)) }},
		{tc.Adapters.Gitleaks.Enabled, func() analysis.ToolAdapter { return gitleaks.New(resolveTool("gitleaks", tc.Adapters.Gitleaks.Binary)) }},
		{tc.Adapters.Madge.Enabled, func() analysis.ToolAdapter { return madge.New(resolveTool("madge", tc.Adapters.Madge.Binary)) }},
		{tc.Adapters.NpmAudit.Enabled, func() analysis.ToolAdapter { return npmaudit.New(resolveTool("npm", tc.Adapters.NpmAudit.Binary)) }},
		{tc.Adapters.Coverage.Enabled, func() analysis.ToolAdapter { return coverage.New(tc.Adapters.Coverage) }},
	}
	var ads []analysis.ToolAdapter
	for _, e := range entries {
		if e.enabled {
			ads = append(ads, e.make())
		}
	}
	for _, c := range tc.Adapters.Custom {
		ads = appendCustomAdapter(ads, c)
	}
	return ads
}

// resolveTool picks the best binary path for an external tool:
//  1. ToolDir()/<name> if downloaded by toolmgr
//  2. $PATH/<name> if available on the system
//  3. The explicit binary from tool_config.toml (fallback)
func resolveTool(name, cfgBinary string) string {
	if p, err := toolmgr.ToolPath(name); err == nil {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if _, err := exec.LookPath(name); err == nil {
		return name
	}
	if cfgBinary != "" {
		return cfgBinary
	}
	return name
}

func appendCustomAdapter(ads []analysis.ToolAdapter, c analysis.CustomToolConfig) []analysis.ToolAdapter {
	if !c.Enabled {
		return ads
	}
	return append(ads, script.New(c.Name, c.Binary, c.Protocol, c.Rules, c.Args))
}

func adapterNames(ads []analysis.ToolAdapter) string {
	names := ""
	for i, a := range ads {
		if i > 0 {
			names += ", "
		}
		avail := "✓"
		if !a.IsAvailable() {
			avail = "✗ (will skip)"
		}
		names += a.Name() + " " + avail
	}
	return names
}

// gitDiffService implements analysis.DiffService by shelling out to git.
type gitDiffService struct{}

func (gitDiffService) ChangedFiles(target, baseRef string) (map[string]bool, error) {
	cmd := exec.Command("git", "-C", target, "diff", "--name-only", baseRef)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool)
	for _, rel := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if rel == "" {
			continue
		}
		set[filepath.Join(target, rel)] = true
	}
	return set, nil
}
