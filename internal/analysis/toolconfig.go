package analysis

// ToolConfig controls which adapter handles which rules and how each is invoked.
type ToolConfig struct {
	Adapters AdaptersConfig `toml:"adapters"`
}

type AdaptersConfig struct {
	TreeSitter TreeSitterConfig   `toml:"treesitter"`
	Secrets    NativeToolConfig   `toml:"secrets"`
	Imports    NativeToolConfig   `toml:"imports"`
	Semgrep    ExternalToolConfig `toml:"semgrep"`
	Gitleaks   ExternalToolConfig `toml:"gitleaks"`
	Madge      ExternalToolConfig `toml:"madge"`
	NpmAudit   ExternalToolConfig `toml:"npmaudit"`
	Coverage   CoverageConfig     `toml:"coverage"`
	Custom     []CustomToolConfig `toml:"custom"`
}

type TreeSitterConfig struct {
	Enabled bool     `toml:"enabled"`
	Rules   []string `toml:"rules"`
}

// NativeToolConfig configures a pure-Go adapter (no external binary). These are
// the zero-dependency defaults; the matching ExternalToolConfig adapters
// (gitleaks/madge/semgrep) are optional enrichment, off by default.
type NativeToolConfig struct {
	Enabled bool     `toml:"enabled"`
	Rules   []string `toml:"rules"`
}

type ExternalToolConfig struct {
	Enabled bool     `toml:"enabled"`
	Binary  string   `toml:"binary"`
	Rules   []string `toml:"rules"`
	Args    []string `toml:"args"`
}

type CoverageConfig struct {
	Enabled     bool    `toml:"enabled"`
	Threshold   float64 `toml:"threshold"`
	LcovPath    string  `toml:"lcov_path"`
	GoCoverPath string  `toml:"gocover_path"`
}

type CustomToolConfig struct {
	Name     string   `toml:"name"`
	Binary   string   `toml:"binary"`
	Enabled  bool     `toml:"enabled"`
	Protocol string   `toml:"protocol"`
	Rules    []string `toml:"rules"`
	Args     []string `toml:"args"`
}
