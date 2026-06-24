package analysis

// Severity mirrors the severity ladder in code_review_standards.toml.
type Severity string

const (
	SeverityBlocker  Severity = "blocker"
	SeverityMajor    Severity = "major"
	SeverityAdvisory Severity = "advisory"
	SeverityInfo     Severity = "info"
)

// Finding is one violation of a standard rule.
type Finding struct {
	Rule        string   // e.g. "complexity.cyclomatic"
	Pillar      string   // e.g. "complexity"
	Severity    Severity
	File        string
	Line        int
	Column      int
	Message     string
	Remediation string
	Snippet     string   // surrounding source lines for context
	Source      string   // adapter/analyzer that produced this finding
	Tags        []string // e.g. ["owasp:A03:2021", "cwe:89"]
	Standards   []string // e.g. ["OWASP-2021-A03", "CWE-89"]
}

// Language represents a supported source language.
type Language string

const (
	LangTypeScript  Language = "typescript"
	LangJavaScript  Language = "javascript"
	LangGo          Language = "go"
	LangPython      Language = "python"
	LangRust        Language = "rust"
	LangUnknown     Language = "unknown"
)

// FileInfo carries per-file metadata collected during the walk.
type FileInfo struct {
	Path     string
	Language Language
	Lines    int
	Content  []byte
}

// ExtToLanguage maps file extensions to languages.
var ExtToLanguage = map[string]Language{
	".ts":  LangTypeScript,
	".tsx": LangTypeScript,
	".js":  LangJavaScript,
	".jsx": LangJavaScript,
	".mjs": LangJavaScript,
	".cjs": LangJavaScript,
	".go":  LangGo,
	".py":  LangPython,
	".rs":  LangRust,
}
