package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	coverage "github.com/srivastava-ami/coderev/internal/adapters/coverage"
	gitleaks "github.com/srivastava-ami/coderev/internal/adapters/gitleaks"
	madge "github.com/srivastava-ami/coderev/internal/adapters/madge"
	npmaudit "github.com/srivastava-ami/coderev/internal/adapters/npmaudit"
	script "github.com/srivastava-ami/coderev/internal/adapters/script"
	semgrep "github.com/srivastava-ami/coderev/internal/adapters/semgrep"
	tsadapter "github.com/srivastava-ami/coderev/internal/adapters/treesitter"
	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/toolmgr"
)

func buildAdapters(stds analysis.Standards, tc analysis.ToolConfig) []analysis.ToolAdapter {
	var ads []analysis.ToolAdapter
	if tc.Adapters.TreeSitter.Enabled {
		ads = append(ads, tsadapter.New(stds))
	}
	if tc.Adapters.Semgrep.Enabled {
		ads = append(ads, semgrep.New(resolveTool("semgrep", tc.Adapters.Semgrep.Binary)))
	}
	if tc.Adapters.Gitleaks.Enabled {
		ads = append(ads, gitleaks.New(resolveTool("gitleaks", tc.Adapters.Gitleaks.Binary)))
	}
	if tc.Adapters.Madge.Enabled {
		ads = append(ads, madge.New(resolveTool("madge", tc.Adapters.Madge.Binary)))
	}
	if tc.Adapters.NpmAudit.Enabled {
		ads = append(ads, npmaudit.New(resolveTool("npm", tc.Adapters.NpmAudit.Binary)))
	}
	if tc.Adapters.Coverage.Enabled {
		ads = append(ads, coverage.New(tc.Adapters.Coverage))
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
