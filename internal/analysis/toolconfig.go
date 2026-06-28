package analysis

// ToolConfig controls which adapter handles which rules and how each is invoked.
type ToolConfig struct {
	Adapters AdaptersConfig `toml:"adapters"`
	SARIF    SARIFConfig    `toml:"sarif"`
	Graph    GraphConfig    `toml:"graph"`
	Github   GithubConfig   `toml:"github"`
	Scan     ScanConfig     `toml:"scan"`
}

// GraphConfig configures the native code-graph output (the `coderev graph`
// command). OutputDir is where graph.json + graph.html are written; relative
// paths are resolved against the scanned target. Defaults to ".coderev/graph".
type GraphConfig struct {
	OutputDir string `toml:"output_dir"`
}

// SARIFConfig holds the URLs emitted in SARIF output. They are configuration —
// not hardcoded constants — so they can be overridden and so coderev's own
// source carries no hardcoded URLs.
type SARIFConfig struct {
	SchemaURL      string `toml:"schema_url"`
	InformationURI string `toml:"information_uri"`
}

type GraphAnalyzeConfig struct {
	Enabled   bool     `toml:"enabled"`
	FanInMax  int      `toml:"fan_in_max"`
	FanOutMax int      `toml:"fan_out_max"`
	Rules     []string `toml:"rules"`
}

type AdaptersConfig struct {
	TreeSitter  TreeSitterConfig   `toml:"treesitter"`
	DepCve      DepCveConfig       `toml:"depcve"`
	Secrets     NativeToolConfig   `toml:"secrets"`
	Imports     NativeToolConfig   `toml:"imports"`
	Semgrep     ExternalToolConfig `toml:"semgrep"`
	Gitleaks    ExternalToolConfig `toml:"gitleaks"`
	Madge       ExternalToolConfig `toml:"madge"`
	NpmAudit    ExternalToolConfig `toml:"npmaudit"`
	Coverage    CoverageConfig     `toml:"coverage"`
	GraphAnalyze GraphAnalyzeConfig `toml:"graphanalyze"`
	Custom      []CustomToolConfig `toml:"custom"`
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
	Enabled     bool     `toml:"enabled"`
	Binary      string   `toml:"binary"`
	Rules       []string `toml:"rules"`
	Args        []string `toml:"args"`
	DownloadURL string   `toml:"download_url"` // release URL template (toolmgr); printf %s slots
}

type DepCveConfig struct {
	Enabled     bool     `toml:"enabled"`
	SnapshotURL string   `toml:"snapshot_url"`
	Rules       []string `toml:"rules"`
}

type CoverageConfig struct {
	Enabled     bool    `toml:"enabled"`
	Threshold   float64 `toml:"threshold"`
	LcovPath    string  `toml:"lcov_path"`
	GoCoverPath string  `toml:"gocover_path"`
}

// ScanConfig controls the memory-bounded file walk behaviour.
type ScanConfig struct {
	BatchSize int `toml:"batch_size"`
}

// GithubConfig holds the GitHub API base URL. It lives in TOML rather than as a
// hardcoded Go constant so coderev's hardcoding.urls_and_paths rule stays clean.
type GithubConfig struct {
	BaseURL string `toml:"base_url"`
}

type CustomToolConfig struct {
	Name     string   `toml:"name"`
	Binary   string   `toml:"binary"`
	Enabled  bool     `toml:"enabled"`
	Protocol string   `toml:"protocol"`
	Rules    []string `toml:"rules"`
	Args     []string `toml:"args"`
}
