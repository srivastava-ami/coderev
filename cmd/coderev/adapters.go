package main

import (
	coverage "github.com/srivastava-ami/coderev/internal/adapters/coverage"
	gitleaks "github.com/srivastava-ami/coderev/internal/adapters/gitleaks"
	madge "github.com/srivastava-ami/coderev/internal/adapters/madge"
	npmaudit "github.com/srivastava-ami/coderev/internal/adapters/npmaudit"
	script "github.com/srivastava-ami/coderev/internal/adapters/script"
	semgrep "github.com/srivastava-ami/coderev/internal/adapters/semgrep"
	tsadapter "github.com/srivastava-ami/coderev/internal/adapters/treesitter"
	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/config"
)

func buildAdapters(stds config.Standards, tc config.ToolConfig) []analysis.ToolAdapter {
	var ads []analysis.ToolAdapter
	if tc.Adapters.TreeSitter.Enabled {
		ads = append(ads, tsadapter.New(stds))
	}
	if tc.Adapters.Semgrep.Enabled {
		ads = append(ads, semgrep.New(tc.Adapters.Semgrep.Binary))
	}
	if tc.Adapters.Gitleaks.Enabled {
		ads = append(ads, gitleaks.New(tc.Adapters.Gitleaks.Binary))
	}
	if tc.Adapters.Madge.Enabled {
		ads = append(ads, madge.New(tc.Adapters.Madge.Binary))
	}
	if tc.Adapters.NpmAudit.Enabled {
		ads = append(ads, npmaudit.New(tc.Adapters.NpmAudit.Binary))
	}
	if tc.Adapters.Coverage.Enabled {
		ads = append(ads, coverage.New(tc.Adapters.Coverage))
	}
	for _, c := range tc.Adapters.Custom {
		ads = appendCustomAdapter(ads, c)
	}
	return ads
}

func appendCustomAdapter(ads []analysis.ToolAdapter, c config.CustomToolConfig) []analysis.ToolAdapter {
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
