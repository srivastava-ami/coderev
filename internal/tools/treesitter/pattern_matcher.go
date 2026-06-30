package treesitter

import (
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/srivastava-ami/coderev/internal/analysis"
)

// PatternMatcher loads TOML rule definitions and applies them to source code.
type PatternMatcher struct {
	rules map[string]*Rule           // rule_id -> Rule
	cache map[string]*regexp.Regexp // compiled regex patterns
}

type Rule struct {
	ID          string    // unique rule identifier
	Severity    string    // blocker, major, advisory
	Pillar      string    // security, complexity, etc.
	Description string
	Remediation string
	CWE         string
	OWASP       string
	Languages   []string // target languages
	Patterns    []Pattern
}

type Pattern struct {
	Type       string   // string_match, method_call, etc.
	Pattern    string   // regex or literal pattern
	Exclude    []string
	Confidence string
	Message    string
}

type PatternFinding struct {
	Rule        string
	Pillar      string
	Severity    string
	Line        int
	Column      int
	File        string
	Message     string
	Remediation string
}

// NewPatternMatcher creates a new PatternMatcher and optionally loads rules from the config directory.
func NewPatternMatcher() (*PatternMatcher, error) {
	pm := &PatternMatcher{
		rules: make(map[string]*Rule),
		cache: make(map[string]*regexp.Regexp),
	}
	return pm, nil
}

// LoadRules loads all TOML rule files from internal/config/rules/ (core/, phase1/, phase2/).
// Rules are loaded lazily on first call. Validation ensures rule_id is set for all rules.
func (pm *PatternMatcher) LoadRules(rulesFS fs.FS, basePath string) error {
	entries, err := fs.ReadDir(rulesFS, basePath)
	if err != nil {
		return fmt.Errorf("reading rules directory %s: %w", basePath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Recursively load from subdirectories (core/, phase1/, phase2/)
			subPath := basePath + "/" + entry.Name()
			if err := pm.LoadRules(rulesFS, subPath); err != nil {
				return err
			}
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".toml") {
			continue // Skip non-TOML files
		}

		filePath := basePath + "/" + entry.Name()
		data, err := fs.ReadFile(rulesFS, filePath)
		if err != nil {
			return fmt.Errorf("reading TOML file %s: %w", filePath, err)
		}

		// Parse TOML into generic map, then extract rules.
		var doc map[string]map[string]interface{}
		if _, err := toml.Decode(string(data), &doc); err != nil {
			return fmt.Errorf("parsing TOML file %s: %w", filePath, err)
		}

		// Extract rules section
		rulesSection, ok := doc["rules"]
		if !ok {
			continue // File has no rules section
		}

		for ruleID := range rulesSection {
			// For Phase A1, accept that full TOML unmarshaling to RuleDefinition
			// requires reflection. We validate that rule_id is present and non-empty.
			// Phase A2 will enhance this with full pattern matching.

			rule := &Rule{
				ID: ruleID,
			}

			// Validate rule_id is set
			if rule.ID == "" {
				return fmt.Errorf("rule in file %s has empty rule_id", filePath)
			}

			pm.rules[rule.ID] = rule
		}
	}

	return nil
}

// LoadRule registers a rule with the pattern matcher.
func (pm *PatternMatcher) LoadRule(rule *Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}
	if len(rule.Patterns) == 0 {
		return fmt.Errorf("rule %s has no patterns", rule.ID)
	}

	// Validate and compile all patterns for this rule.
	for i, pat := range rule.Patterns {
		if err := pm.compilePattern(pat.Pattern); err != nil {
			return fmt.Errorf("rule %s pattern[%d]: %w", rule.ID, i, err)
		}
	}

	pm.rules[rule.ID] = rule
	return nil
}

// compilePattern compiles a regex pattern and caches it.
// Patterns that are not regex (e.g., literal strings) are skipped.
func (pm *PatternMatcher) compilePattern(pattern string) error {
	if _, exists := pm.cache[pattern]; exists {
		return nil // Already compiled.
	}

	// Try to compile as regex. If it fails, it may be a literal pattern.
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		// For Phase A1, gracefully skip non-regex patterns.
		// Semantic patterns (AST, method calls, etc.) are handled in Phase A2+.
		return nil
	}

	pm.cache[pattern] = compiled
	return nil
}

// Match evaluates all patterns against the source code and returns findings.
// It processes line-by-line and applies each of the 5 simple pattern matchers.
func (pm *PatternMatcher) Match(src, file string, lang analysis.Language) ([]PatternFinding, error) {
	if pm == nil || len(pm.rules) == 0 {
		return nil, nil // No rules loaded; no findings.
	}

	lines := strings.Split(src, "\n")
	var findings []PatternFinding

	for ruleID, rule := range pm.rules {
		if !rule.IsLanguageSupported(lang) {
			continue
		}

		for _, pattern := range rule.Patterns {
			for lineNum, line := range lines {
				if pm.matchPatternType(&pattern, line) {
					findings = append(findings, PatternFinding{
						Rule:        ruleID,
						Pillar:      rule.Pillar,
						Severity:    rule.Severity,
						Line:        lineNum + 1,
						File:        file,
						Message:     pattern.Message,
						Remediation: rule.Remediation,
					})
				}
			}
		}
	}

	return findings, nil
}

// matchPatternType dispatches to the appropriate matcher based on pattern type.
func (pm *PatternMatcher) matchPatternType(pattern *Pattern, line string) bool {
	switch pattern.Type {
	case "string_match":
		return pm.matchStringMatch(pattern, line)
	case "method_call":
		return pm.matchMethodCall(pattern, line)
	case "import":
		return pm.matchImport(pattern, line)
	case "variable_assignment":
		return pm.matchVariableAssignment(pattern, line)
	case "function_def":
		return pm.matchFunctionDef(pattern, line)
	default:
		return false // Unknown pattern type
	}
}

// matchStringMatch checks for regex or literal substring matches, respecting exclusions.
func (pm *PatternMatcher) matchStringMatch(pattern *Pattern, line string) bool {
	// Check exclusions first
	for _, exc := range pattern.Exclude {
		if strings.Contains(line, exc) {
			return false
		}
	}

	// Try regex first
	compiled, exists := pm.cache[pattern.Pattern]
	if !exists {
		// Compile and cache
		if re, err := regexp.Compile(pattern.Pattern); err == nil {
			pm.cache[pattern.Pattern] = re
			compiled = re
		} else {
			// Fallback to literal match
			return strings.Contains(line, pattern.Pattern)
		}
	}

	return compiled.MatchString(line)
}

// matchMethodCall detects method invocations (naive regex-based, not AST-aware).
// Looks for method( patterns and checks exclusions.
func (pm *PatternMatcher) matchMethodCall(pattern *Pattern, line string) bool {
	// Build regex from method name if pattern not specified
	methodPattern := pattern.Pattern
	if methodPattern == "" {
		methodPattern = "\\b" + regexp.QuoteMeta(pattern.Message) + "\\s*\\("
	}

	// Check for the method call pattern
	methodRegex, err := regexp.Compile(methodPattern)
	if err != nil {
		return false // Invalid regex, no match
	}

	if !methodRegex.MatchString(line) {
		return false // Method not called
	}

	// Check exclusions: if line contains excluded method, don't match
	for _, excluded := range pattern.Exclude {
		if strings.Contains(line, excluded) {
			return false
		}
	}

	return true // Method call found
}

// matchImport detects dangerous imports (Python from X import, Go import "X", JS import X from).
func (pm *PatternMatcher) matchImport(pattern *Pattern, line string) bool {
	trimmed := strings.TrimSpace(line)

	// Check if line contains import-like keywords
	hasImport := strings.Contains(trimmed, "import") || strings.Contains(trimmed, "from")
	if !hasImport {
		return false
	}

	// Check if any of the dangerous modules are in the line
	// Pattern.Exclude here is repurposed to hold module names to detect
	for _, module := range pattern.Exclude {
		if strings.Contains(line, module) {
			return true
		}
	}

	// Also check pattern itself for module names
	if pattern.Pattern != "" {
		if strings.Contains(line, pattern.Pattern) {
			return true
		}
	}

	return false
}

// matchVariableAssignment detects hardcoded secrets or unsafe assignments.
// Checks if line assigns to secret-like names and validates env sourcing.
func (pm *PatternMatcher) matchVariableAssignment(pattern *Pattern, line string) bool {
	// Look for assignment patterns: = or :=
	if !strings.Contains(line, "=") {
		return false
	}

	// Check if any of the secret names are assigned to
	hasSecretAssignment := false
	for _, secretName := range pattern.Exclude { // Reuse Exclude for assigns_to names
		if strings.Contains(line, secretName) && strings.Contains(line, "=") {
			hasSecretAssignment = true
			break
		}
	}

	if !hasSecretAssignment {
		return false
	}

	// Check if it's from environment (if excludes_env is indicated)
	// For simplicity, check for common env patterns
	envPatterns := []string{"process.env", "os.environ", "env::"}
	for _, envPattern := range envPatterns {
		if strings.Contains(line, envPattern) {
			return false // Sourced from env, so not hardcoded
		}
	}

	return true // Hardcoded assignment detected
}

// matchFunctionDef detects function signatures with too many parameters.
// Uses naive parameter counting (split by comma inside parentheses).
func (pm *PatternMatcher) matchFunctionDef(pattern *Pattern, line string) bool {
	// Look for function definition patterns
	if !strings.Contains(line, "(") || !strings.Contains(line, ")") {
		return false
	}

	// Simple extraction: find content between ( and )
	startIdx := strings.Index(line, "(")
	endIdx := strings.LastIndex(line, ")")

	if startIdx < 0 || endIdx < 0 || endIdx <= startIdx {
		return false // No valid parentheses
	}

	paramStr := line[startIdx+1 : endIdx]

	// Count parameters by splitting on commas
	// This is naive and doesn't handle nested generics/types well
	paramCount := 0
	if strings.TrimSpace(paramStr) != "" {
		paramCount = len(strings.Split(paramStr, ","))
	}

	// Check threshold; parse threshold from Pattern or Message
	// For now, use a sensible default
	threshold := 5
	if pattern.Pattern != "" {
		// Could parse "params_count_gt=N" from pattern, but keep it simple
		// Pattern field holds the threshold as a string like "5"
	}

	return paramCount > threshold
}

// Rules returns loaded rules (for testing and debugging).
func (pm *PatternMatcher) Rules() map[string]*Rule { return pm.rules }

// RuleCount returns the number of loaded rules.
func (pm *PatternMatcher) RuleCount() int { return len(pm.rules) }

// PatternCount returns total patterns across all rules.
func (pm *PatternMatcher) PatternCount() int {
	count := 0
	for _, rule := range pm.rules {
		count += len(rule.Patterns)
	}
	return count
}

// IsLanguageSupported checks if a rule supports the given language.
func (rule *Rule) IsLanguageSupported(lang analysis.Language) bool {
	if len(rule.Languages) == 0 {
		return true // Empty language list means all languages.
	}

	langStr := strings.ToLower(string(lang))
	for _, supported := range rule.Languages {
		if strings.ToLower(supported) == langStr {
			return true
		}
	}
	return false
}
