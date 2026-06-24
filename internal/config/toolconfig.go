package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ToolConfig is the typed representation of tool_config.toml.
// It controls which adapter handles which rules, and how each adapter is invoked.
type ToolConfig struct {
	Adapters AdaptersConfig `toml:"adapters"`
}

type AdaptersConfig struct {
	TreeSitter TreeSitterConfig  `toml:"treesitter"`
	Semgrep    ExternalToolConfig `toml:"semgrep"`
	Gitleaks   ExternalToolConfig `toml:"gitleaks"`
	Madge      ExternalToolConfig `toml:"madge"`
	NpmAudit   ExternalToolConfig `toml:"npmaudit"`
	Coverage   CoverageConfig    `toml:"coverage"`
	Custom     []CustomToolConfig `toml:"custom"`
}

// CoverageConfig controls the coverage-gating adapter.
type CoverageConfig struct {
	Enabled   bool    `toml:"enabled"`
	Threshold float64 `toml:"threshold"` // minimum line-coverage % (default 80)
	// LcovPath and GoCoverPath are auto-discovered if empty.
	LcovPath    string `toml:"lcov_path"`
	GoCoverPath string `toml:"gocover_path"`
}

type TreeSitterConfig struct {
	Enabled bool     `toml:"enabled"`
	Rules   []string `toml:"rules"`
}

// ExternalToolConfig covers semgrep, gitleaks, madge, npm-audit.
// Adding a new built-in adapter = one new field here.
type ExternalToolConfig struct {
	Enabled bool     `toml:"enabled"`
	Binary  string   `toml:"binary"`  // path or name on $PATH
	Rules   []string `toml:"rules"`
	Args    []string `toml:"args"`    // extra CLI args forwarded to the tool
}

// CustomToolConfig is the extension point: any external program that emits
// findings as NDJSON in our Finding schema can plug in here without touching
// Go source.
type CustomToolConfig struct {
	Name     string   `toml:"name"`
	Binary   string   `toml:"binary"`
	Enabled  bool     `toml:"enabled"`
	Protocol string   `toml:"protocol"` // "ndjson" | "json"
	Rules    []string `toml:"rules"`
	Args     []string `toml:"args"` // {{target}} is substituted at runtime
}

// LoadToolConfig parses a tool_config.toml. If path is empty, sensible
// defaults are returned (all built-in adapters enabled, auto-detected binaries).
func LoadToolConfig(path string) (ToolConfig, error) {
	if path == "" {
		return defaultToolConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ToolConfig{}, fmt.Errorf("reading tool config: %w", err)
	}
	var tc ToolConfig
	if _, err := toml.Decode(string(data), &tc); err != nil {
		return ToolConfig{}, fmt.Errorf("parsing tool config TOML: %w", err)
	}
	return tc, nil
}

func defaultToolConfig() ToolConfig {
	return ToolConfig{
		Adapters: AdaptersConfig{
			TreeSitter: TreeSitterConfig{
				Enabled: true,
				Rules:   []string{"complexity.*", "file_structure.*", "type_safety.*", "hardcoding.*", "documentation.*", "observability.logging", "stability.error_handling"},
			},
			Semgrep: ExternalToolConfig{
				Enabled: true,
				Binary:  "semgrep",
				Rules:   []string{"security.injection.*", "security.auth.*", "security.cryptography"},
			},
			Gitleaks: ExternalToolConfig{
				Enabled: true,
				Binary:  "gitleaks",
				Rules:   []string{"security.secrets"},
			},
			Madge: ExternalToolConfig{
				Enabled: true,
				Binary:  "madge",
				Rules:   []string{"file_structure.circular_deps", "nx_conventions.boundaries"},
			},
			NpmAudit: ExternalToolConfig{
				Enabled: true,
				Binary:  "npm",
				Rules:   []string{"security.dependencies"},
			},
			Coverage: CoverageConfig{
				Enabled:   true,
				Threshold: 80.0,
			},
		},
	}
}
