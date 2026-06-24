package config

// ── Security ──────────────────────────────────────────────────────────────────

type SecurityStd struct {
	Severity string         `toml:"severity"`
	Secrets  SecretsStd     `toml:"secrets"`
	Supply   SupplyChainStd `toml:"supply_chain"`
}

type SecretsStd struct {
	Rule        string   `toml:"rule"`
	Patterns    []string `toml:"patterns"`
	Remediation string   `toml:"remediation"`
}

type SupplyChainStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

// ── Stability ─────────────────────────────────────────────────────────────────

type StabilityStd struct {
	Severity      string           `toml:"severity"`
	ErrorHandling ErrorHandlingStd `toml:"error_handling"`
}

type ErrorHandlingStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

// ── Hardcoding ────────────────────────────────────────────────────────────────

type HardcodingStd struct {
	Severity          string          `toml:"severity"`
	EnvironmentValues EnvValuesStd    `toml:"environment_values"`
	MagicNumbers      MagicNumbersStd `toml:"magic_numbers"`
}

type EnvValuesStd struct {
	Rule        string   `toml:"rule"`
	Examples    []string `toml:"examples"`
	Remediation string   `toml:"remediation"`
}

type MagicNumbersStd struct {
	Severity    string `toml:"severity"`
	Rule        string `toml:"rule"`
	Exceptions  []int  `toml:"exceptions"`
	Remediation string `toml:"remediation"`
}

// ── Type Safety ───────────────────────────────────────────────────────────────

type TypeSafetyStd struct {
	Severity   string        `toml:"severity"`
	NoAny      NoAnyStd      `toml:"no_any"`
	NullSafety NullSafetyStd `toml:"null_safety"`
}

type NoAnyStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

type NullSafetyStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

// ── Complexity ────────────────────────────────────────────────────────────────

type ComplexityStd struct {
	Severity    string            `toml:"severity"`
	Cyclomatic  CyclomaticStd     `toml:"cyclomatic"`
	Cognitive   CognitiveStd      `toml:"cognitive"`
	Function    FunctionLengthStd `toml:"function_length"`
	Parameters  ParameterStd      `toml:"parameter_count"`
	Nesting     NestingStd        `toml:"nesting"`
	Duplication DuplicationStd    `toml:"duplication"`
}

type CyclomaticStd struct {
	MaxValue    int    `toml:"max_value"`
	AdvisoryAt  int    `toml:"advisory_at"`
	HardBlockAt int    `toml:"hard_block_at"`
	Remediation string `toml:"remediation"`
}

type CognitiveStd struct {
	MaxValue int    `toml:"max_value"`
	Tool     string `toml:"tool"`
}

type FunctionLengthStd struct {
	MaxLines    int    `toml:"max_lines"`
	AdvisoryAt  int    `toml:"advisory_at"`
	Remediation string `toml:"remediation"`
}

type ParameterStd struct {
	MaxCount    int    `toml:"max_count"`
	Remediation string `toml:"remediation"`
}

type NestingStd struct {
	MaxDepth    int    `toml:"max_depth"`
	Remediation string `toml:"remediation"`
}

type DuplicationStd struct {
	Rule            string `toml:"rule"`
	ThresholdTokens int    `toml:"threshold_tokens"`
	Remediation     string `toml:"remediation"`
}

// ── File Structure ────────────────────────────────────────────────────────────

type FileStructureStd struct {
	Severity    string         `toml:"severity"`
	FileLength  FileLengthStd  `toml:"file_length"`
	ClassLength ClassLengthStd `toml:"class_length"`
}

type FileLengthStd struct {
	MaxLines    int    `toml:"max_lines"`
	AdvisoryAt  int    `toml:"advisory_at"`
	Remediation string `toml:"remediation"`
}

type ClassLengthStd struct {
	MaxLines    int    `toml:"max_lines"`
	Remediation string `toml:"remediation"`
}

// ── Observability ─────────────────────────────────────────────────────────────

type ObservabilityStd struct {
	Severity string     `toml:"severity"`
	Logging  LoggingStd `toml:"logging"`
}

type LoggingStd struct {
	Rule            string   `toml:"rule"`
	RequiredFields  []string `toml:"required_fields"`
	Checks          []string `toml:"checks"`
	ForbiddenLevels []string `toml:"forbidden_levels"`
}

// ── Documentation ─────────────────────────────────────────────────────────────

type DocumentationStd struct {
	Severity       string            `toml:"severity"`
	CommentQuality CommentQualityStd `toml:"comment_quality"`
	NoTombstones   NoTombstonesStd   `toml:"no_comment_tombstones"`
	TodoFormat     TodoFormatStd     `toml:"todo_format"`
}

type CommentQualityStd struct {
	Rule        string   `toml:"rule"`
	BadPatterns []string `toml:"bad_patterns"`
	Remediation string   `toml:"remediation"`
}

type NoTombstonesStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Description string `toml:"description"`
	Remediation string `toml:"remediation"`
}

type TodoFormatStd struct {
	Rule        string `toml:"rule"`
	Pattern     string `toml:"pattern"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

// ── Testing ───────────────────────────────────────────────────────────────────

type TestingStd struct {
	Severity string      `toml:"severity"`
	Coverage CoverageStd `toml:"coverage"`
}

type CoverageStd struct {
	Lines          int `toml:"lines"`
	Branches       int `toml:"branches"`
	Functions      int `toml:"functions"`
	Statements     int `toml:"statements"`
	NewCodeMinimum int `toml:"new_code_minimum"`
}

// ── Performance ───────────────────────────────────────────────────────────────

type PerformanceStd struct {
	Database PerformanceDBStd    `toml:"database"`
	Async    PerformanceAsyncStd `toml:"async"`
}

type PerformanceDBStd struct {
	Severity string   `toml:"severity"`
	Checks   []string `toml:"checks"`
}

type PerformanceAsyncStd struct {
	Severity string   `toml:"severity"`
	Checks   []string `toml:"checks"`
}

// ── Python Conventions ─────────────────────────────────────────────────────────

type PythonConventionsStd struct {
	Severity string `toml:"severity"`
}

// ── Rust Conventions ───────────────────────────────────────────────────────────

type RustConventionsStd struct {
	Severity string `toml:"severity"`
}

// ── NX Conventions ────────────────────────────────────────────────────────────

type NxConventionsStd struct {
	Severity   string        `toml:"severity"`
	Boundaries BoundariesStd `toml:"boundaries"`
	Tags       TagsStd       `toml:"tags"`
}

type BoundariesStd struct {
	Rule        string `toml:"rule"`
	Description string `toml:"description"`
	Tool        string `toml:"tool"`
}

type TagsStd struct {
	Rule            string   `toml:"rule"`
	RequiredTagAxes []string `toml:"required_axes"`
}

