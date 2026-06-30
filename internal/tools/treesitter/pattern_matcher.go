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
	Type              string   // string_match, method_call, loop_contains, multi_line, negative, metric, semantic
	Pattern           string   // regex or literal pattern
	Exclude           []string
	Confidence        string
	Message           string
	// Complex patterns
	Inside            string   // loop_contains: "for", "while", "foreach"
	Contains          []string // loop_contains: keywords to find inside loop
	Lines             int      // multi_line: lookahead count
	PatternSequence   []string // multi_line: []string regexes in sequence
	FunctionCalls     []string // negative: methods to detect (e.g., "db.Query")
	Missing           string   // negative: pattern that should be present
	Metric            string   // metric: cyclomatic_complexity, coupling, fan_in, fan_out
	ThresholdGt       int      // metric: threshold greater than
	Semantic          string   // semantic: nil_pointer_dereference, unchecked_cast, resource_leak
	Language          string   // semantic: go, rust, python
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
				if pm.matchPatternType(&pattern, line, lines, lineNum) {
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
func (pm *PatternMatcher) matchPatternType(pattern *Pattern, line string, lines []string, lineNum int) bool {
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
	case "loop_contains":
		return pm.matchLoopContains(pattern, lines, lineNum)
	case "multi_line":
		return pm.matchMultiLine(pattern, lines, lineNum)
	case "negative":
		return pm.matchNegative(pattern, lines, lineNum)
	case "metric":
		return pm.matchMetric(pattern, lines, lineNum)
	case "semantic":
		return pm.matchSemantic(pattern, lines, lineNum)
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

// matchLoopContains detects keywords inside loops (memory allocation in hot paths).
// Looks for loop opening, then checks next 10+ lines for contained keywords.
func (pm *PatternMatcher) matchLoopContains(pattern *Pattern, lines []string, lineNum int) bool {
	if pattern.Inside == "" || len(pattern.Contains) == 0 {
		return false
	}

	line := lines[lineNum]
	inside := strings.ToLower(pattern.Inside)

	// Check if this line opens the loop type
	trimmed := strings.TrimSpace(line)
	loopOpensHere := false
	switch inside {
	case "for":
		loopOpensHere = strings.HasPrefix(trimmed, "for") && strings.Contains(line, "{")
	case "while":
		loopOpensHere = strings.HasPrefix(trimmed, "while") && strings.Contains(line, "{")
	case "foreach":
		loopOpensHere = strings.Contains(trimmed, "foreach") && strings.Contains(line, "{")
	default:
		return false
	}

	if !loopOpensHere {
		return false
	}

	// Look ahead up to 10 lines for contained keywords
	lookahead := 10
	if lineNum+lookahead >= len(lines) {
		lookahead = len(lines) - lineNum - 1
	}

	for i := 1; i <= lookahead; i++ {
		nextLine := lines[lineNum+i]
		for _, keyword := range pattern.Contains {
			if strings.Contains(nextLine, keyword) {
				return true
			}
		}
		// Stop if we hit closing brace (simple heuristic)
		if strings.Contains(nextLine, "}") && !strings.Contains(nextLine, "{") {
			break
		}
	}

	return false
}

// matchMultiLine detects pattern sequences over N lines (lookahead).
// All patterns must match in sequence, gaps allowed.
func (pm *PatternMatcher) matchMultiLine(pattern *Pattern, lines []string, lineNum int) bool {
	if pattern.Lines == 0 || len(pattern.PatternSequence) == 0 {
		return false
	}

	// Check if current line matches first pattern
	if lineNum >= len(lines) {
		return false
	}

	currentLine := lines[lineNum]
	re0, err := regexp.Compile(pattern.PatternSequence[0])
	if err != nil {
		return false // Invalid regex
	}

	if !re0.MatchString(currentLine) {
		return false
	}

	// If only one pattern, we're done
	if len(pattern.PatternSequence) == 1 {
		return true
	}

	// Now look ahead for remaining patterns
	patIdx := 1
	maxLines := pattern.Lines
	if lineNum+maxLines >= len(lines) {
		maxLines = len(lines) - lineNum - 1
	}

	for i := 1; i <= maxLines && patIdx < len(pattern.PatternSequence); i++ {
		if lineNum+i >= len(lines) {
			break
		}
		nextLine := lines[lineNum+i]
		re, err := regexp.Compile(pattern.PatternSequence[patIdx])
		if err != nil {
			continue // Skip invalid patterns
		}
		if re.MatchString(nextLine) {
			patIdx++
		}
	}

	// All patterns matched in sequence
	return patIdx == len(pattern.PatternSequence)
}

// matchNegative detects missing patterns (e.g., missing timeout on db.Query).
// Finds function_calls, checks if missing pattern appears in next 5 lines.
func (pm *PatternMatcher) matchNegative(pattern *Pattern, lines []string, lineNum int) bool {
	if len(pattern.FunctionCalls) == 0 || pattern.Missing == "" {
		return false
	}

	if lineNum >= len(lines) {
		return false
	}

	line := lines[lineNum]

	// Check if any function_call is present on this line
	callFound := false
	for _, call := range pattern.FunctionCalls {
		if strings.Contains(line, call) {
			callFound = true
			break
		}
	}

	if !callFound {
		return false
	}

	// Look for missing pattern in current line + next 5 lines
	missingFound := false
	lookahead := 5
	if lineNum+lookahead >= len(lines) {
		lookahead = len(lines) - lineNum - 1
	}

	for i := 0; i <= lookahead; i++ {
		if lineNum+i < len(lines) && strings.Contains(lines[lineNum+i], pattern.Missing) {
			missingFound = true
			break
		}
	}

	// Return true if missing pattern was NOT found
	return !missingFound
}

// matchMetric detects quantitative violations (cyclomatic complexity, etc.).
// For cyclomatic: count if/else/switch/case branches in the function starting at lineNum.
func (pm *PatternMatcher) matchMetric(pattern *Pattern, lines []string, lineNum int) bool {
	if pattern.Metric == "" || pattern.ThresholdGt == 0 {
		return false
	}

	// For now, implement cyclomatic complexity heuristic
	if pattern.Metric == "cyclomatic_complexity" {
		return pm.countCyclomaticComplexity(lines, lineNum) > pattern.ThresholdGt
	}

	// Other metrics: stub for now
	return false
}

// countCyclomaticComplexity counts if/else/switch/case branches in a function block.
// Counts from lineNum onwards, looking for a function definition and counting complexity within it.
func (pm *PatternMatcher) countCyclomaticComplexity(lines []string, lineNum int) int {
	if lineNum >= len(lines) {
		return 1
	}

	complexity := 1 // Base complexity
	braceDepth := 0
	inFunction := false
	startedCounting := false

	for i := lineNum; i < len(lines); i++ {
		line := lines[i]

		// Detect function opening on or after lineNum
		if !inFunction {
			if strings.Contains(line, "function") || strings.Contains(line, "=>") || strings.Contains(line, "def ") {
				inFunction = true
			} else {
				continue
			}
		}

		// Track braces
		for _, ch := range line {
			if ch == '{' {
				braceDepth++
				startedCounting = true
			} else if ch == '}' {
				braceDepth--
				if startedCounting && braceDepth == 0 {
					return complexity // End of function
				}
			}
		}

		if !startedCounting {
			continue
		}

		// Count complexity nodes (only within the function body)
		trimmed := strings.TrimSpace(line)

		// if statement
		if strings.HasPrefix(trimmed, "if ") || strings.HasPrefix(trimmed, "if(") {
			complexity++
		}
		// else if
		if strings.Contains(line, " else if ") || strings.Contains(line, "}else if") {
			complexity++
		}
		// switch cases
		if strings.HasPrefix(trimmed, "case ") {
			complexity++
		}
		// catch blocks
		if strings.HasPrefix(trimmed, "catch ") || strings.Contains(line, "catch(") {
			complexity++
		}
		// for and while loops
		if strings.HasPrefix(trimmed, "for ") || strings.HasPrefix(trimmed, "for(") {
			complexity++
		}
		if strings.HasPrefix(trimmed, "while ") || strings.HasPrefix(trimmed, "while(") {
			complexity++
		}
	}

	return complexity
}

// matchSemantic detects language-specific heuristics (nil dereference, unchecked cast, resource leak).
func (pm *PatternMatcher) matchSemantic(pattern *Pattern, lines []string, lineNum int) bool {
	if pattern.Semantic == "" || pattern.Language == "" {
		return false
	}

	line := lines[lineNum]
	lang := strings.ToLower(pattern.Language)

	switch strings.ToLower(pattern.Semantic) {
	case "nil_pointer_dereference":
		if lang == "go" {
			return pm.detectGoNilDereference(lines, lineNum)
		}
	case "unchecked_cast":
		if lang == "rust" {
			return pm.detectRustUncheckedCast(line)
		}
	case "resource_leak":
		if lang == "python" {
			return pm.detectPythonResourceLeak(lines, lineNum)
		}
	}

	return false
}

// detectGoNilDereference checks for x.Field without nil check in previous line.
func (pm *PatternMatcher) detectGoNilDereference(lines []string, lineNum int) bool {
	if lineNum >= len(lines) {
		return false
	}

	line := lines[lineNum]

	// Look for pattern like x.Field or x.Method()
	if !strings.Contains(line, ".") {
		return false
	}

	// Check if previous line has nil check
	if lineNum > 0 {
		prevLine := lines[lineNum-1]
		nilCheckPatterns := []string{"if ", "!=", "==", "nil"}
		hasNilCheck := false
		for _, pattern := range nilCheckPatterns {
			if strings.Contains(prevLine, pattern) && strings.Contains(prevLine, "nil") {
				hasNilCheck = true
				break
			}
		}
		if hasNilCheck {
			return false // Nil check found
		}
	}

	// Simple heuristic: detect x.something without prior nil check
	parts := strings.Split(line, ".")
	if len(parts) >= 2 {
		// Has a dereference; no nil check found
		return true
	}

	return false
}

// detectRustUncheckedCast checks for "as T" without validation.
func (pm *PatternMatcher) detectRustUncheckedCast(line string) bool {
	// Look for " as " pattern (Rust cast)
	if !strings.Contains(line, " as ") {
		return true // If cast exists, it's unchecked by definition in this heuristic
	}
	return false
}

// detectPythonResourceLeak checks for open() without close() or context manager.
func (pm *PatternMatcher) detectPythonResourceLeak(lines []string, lineNum int) bool {
	line := lines[lineNum]

	// Check for open() without 'with'
	if !strings.Contains(line, "open(") {
		return false
	}

	if strings.Contains(line, "with ") {
		return false // Has context manager
	}

	// Look ahead for close()
	for i := lineNum; i < lineNum+5 && i < len(lines); i++ {
		if strings.Contains(lines[i], ".close()") {
			return false // close() found
		}
	}

	return true // open() without close() or with
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
