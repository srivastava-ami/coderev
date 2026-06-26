package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// LoadToolConfig parses a tool_config.toml. If path is empty, sensible
// defaults are returned (all built-in adapters enabled, auto-detected binaries).
func LoadToolConfig(path string) (analysis.ToolConfig, error) {
	if path == "" {
		return defaultToolConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return analysis.ToolConfig{}, fmt.Errorf("reading tool config: %w", err)
	}
	var tc analysis.ToolConfig
	if _, err := toml.Decode(string(data), &tc); err != nil {
		return analysis.ToolConfig{}, fmt.Errorf("parsing tool config TOML: %w", err)
	}
	return tc, nil
}

func defaultToolConfig() analysis.ToolConfig {
	return analysis.ToolConfig{
		Adapters: analysis.AdaptersConfig{
			TreeSitter: analysis.TreeSitterConfig{
				Enabled: true,
				Rules:   []string{"complexity.*", "file_structure.*", "type_safety.*", "hardcoding.*", "documentation.*", "observability.logging", "stability.error_handling", "security.secret_fallback_literal"},
			},
			// Native (pure-Go) defaults — zero external dependencies.
			Secrets: analysis.NativeToolConfig{
				Enabled: true,
				Rules:   []string{"security.secrets"},
			},
			Imports: analysis.NativeToolConfig{
				Enabled: true,
				Rules:   []string{"file_structure.circular_deps", "nx_conventions.boundaries"},
			},
			// External tools are now OPTIONAL enrichment — off by default. Native
			// adapters above cover the same rules with no binary required. Enable
			// these in tool_config.toml only for extra depth.
			Semgrep: analysis.ExternalToolConfig{
				Enabled: false,
				Binary:  "semgrep",
				Rules:   []string{"security.injection.*", "security.auth.*", "security.cryptography"},
			},
			Gitleaks: analysis.ExternalToolConfig{
				Enabled: false,
				Binary:  "gitleaks",
				Rules:   []string{"security.secrets"},
			},
			Madge: analysis.ExternalToolConfig{
				Enabled: false,
				Binary:  "madge",
				Rules:   []string{"file_structure.circular_deps", "nx_conventions.boundaries"},
			},
			NpmAudit: analysis.ExternalToolConfig{
				Enabled: true,
				Binary:  "npm",
				Rules:   []string{"security.dependencies"},
			},
			Coverage: analysis.CoverageConfig{
				Enabled:   true,
				Threshold: 80.0,
			},
		},
	}
}
