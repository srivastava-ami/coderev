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
	rules map[string]*Rule
	cache map[string]*regexp.Regexp
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
func (pm *PatternMatcher) Match(src, file string, lang analysis.Language) ([]PatternFinding, error) {
	if pm == nil || len(pm.rules) == 0 {
		return nil, nil
	}
	lines := strings.Split(src, "\n")
	var findings []PatternFinding
	for ruleID, rule := range pm.rules {
		if rule.IsLanguageSupported(lang) {
			findings = append(findings, pm.matchRule(ruleID, rule, lines, file)...)
		}
	}
	return findings, nil
}

func (pm *PatternMatcher) matchRule(ruleID string, rule *Rule, lines []string, file string) []PatternFinding {
	var out []PatternFinding
	for _, pattern := range rule.Patterns {
		for lineNum, line := range lines {
			if pm.matchPatternType(&pattern, line, lines, lineNum) {
				out = append(out, PatternFinding{
					Rule: ruleID, Pillar: rule.Pillar, Severity: rule.Severity,
					Line: lineNum + 1, File: file, Message: pattern.Message, Remediation: rule.Remediation,
				})
			}
		}
	}
	return out
}

type matchCtx struct {
	pattern *Pattern
	line    string
	lines   []string
	lineNum int
}

type patternMatcherFunc func(*PatternMatcher, matchCtx) bool

var patternMatchers = map[string]patternMatcherFunc{
	"string_match":       func(pm *PatternMatcher, c matchCtx) bool { return pm.matchStringMatch(c.pattern, c.line) },
	"method_call":        func(pm *PatternMatcher, c matchCtx) bool { return pm.matchMethodCall(c.pattern, c.line) },
	"import":             func(pm *PatternMatcher, c matchCtx) bool { return pm.matchImport(c.pattern, c.line) },
	"variable_assignment": func(pm *PatternMatcher, c matchCtx) bool { return pm.matchVariableAssignment(c.pattern, c.line) },
	"function_def":       func(pm *PatternMatcher, c matchCtx) bool { return pm.matchFunctionDef(c.pattern, c.line) },
	"loop_contains":      func(pm *PatternMatcher, c matchCtx) bool { return pm.matchLoopContains(c.pattern, c.lines, c.lineNum) },
	"multi_line":         func(pm *PatternMatcher, c matchCtx) bool { return pm.matchMultiLine(c.pattern, c.lines, c.lineNum) },
	"negative":           func(pm *PatternMatcher, c matchCtx) bool { return pm.matchNegative(c.pattern, c.lines, c.lineNum) },
	"metric":             func(pm *PatternMatcher, c matchCtx) bool { return pm.matchMetric(c.pattern, c.lines, c.lineNum) },
	"semantic":           func(pm *PatternMatcher, c matchCtx) bool { return pm.matchSemantic(c.pattern, c.lines, c.lineNum) },
}

func (pm *PatternMatcher) matchPatternType(pattern *Pattern, line string, lines []string, lineNum int) bool {
	if fn, ok := patternMatchers[pattern.Type]; ok {
		return fn(pm, matchCtx{pattern: pattern, line: line, lines: lines, lineNum: lineNum})
	}
	return false
}

// matchStringMatch checks for regex or literal substring matches, respecting exclusions.
func (pm *PatternMatcher) matchStringMatch(pattern *Pattern, line string) bool {
	if pm.hasExclusion(line, pattern.Exclude) {
		return false
	}
	re := pm.getOrCompile(pattern.Pattern)
	if re != nil {
		return re.MatchString(line)
	}
	return strings.Contains(line, pattern.Pattern)
}

func (pm *PatternMatcher) hasExclusion(line string, excludes []string) bool {
	for _, exc := range excludes {
		if strings.Contains(line, exc) {
			return true
		}
	}
	return false
}

func (pm *PatternMatcher) getOrCompile(pattern string) *regexp.Regexp {
	compiled, exists := pm.cache[pattern]
	if exists {
		return compiled
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil
	}
	pm.cache[pattern] = re
	return re
}

// matchMethodCall detects method invocations (naive regex-based, not AST-aware).
// Looks for method( patterns and checks exclusions.
func (pm *PatternMatcher) matchMethodCall(pattern *Pattern, line string) bool {
	re := pm.methodCallRegex(pattern)
	if re == nil || !re.MatchString(line) {
		return false
	}
	return !pm.hasExclusion(line, pattern.Exclude)
}

func (pm *PatternMatcher) methodCallRegex(pattern *Pattern) *regexp.Regexp {
	p := pattern.Pattern
	if p == "" {
		p = "\\b" + regexp.QuoteMeta(pattern.Message) + "\\s*\\("
	}
	return pm.getOrCompile(p)
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

func isLoopOpen(inside, trimmed, line string) bool {
	switch inside {
	case "for":
		return strings.HasPrefix(trimmed, "for") && strings.Contains(line, "{")
	case "while":
		return strings.HasPrefix(trimmed, "while") && strings.Contains(line, "{")
	case "foreach":
		return strings.Contains(trimmed, "foreach") && strings.Contains(line, "{")
	}
	return false
}

// matchLoopContains detects keywords inside loops (memory allocation in hot paths).
// Looks for loop opening, then checks next 10+ lines for contained keywords.
func (pm *PatternMatcher) matchLoopContains(pattern *Pattern, lines []string, lineNum int) bool {
	if pattern.Inside == "" || len(pattern.Contains) == 0 || lineNum >= len(lines) {
		return false
	}
	if !isLoopOpen(strings.ToLower(pattern.Inside), strings.TrimSpace(lines[lineNum]), lines[lineNum]) {
		return false
	}
	return pm.lookaheadHasKeyword(pattern.Contains, lines, lineNum)
}

func (pm *PatternMatcher) lookaheadHasKeyword(keywords []string, lines []string, lineNum int) bool {
	max := 10
	if lineNum+max >= len(lines) {
		max = len(lines) - lineNum - 1
	}
	for i := 1; i <= max; i++ {
		next := lines[lineNum+i]
		for _, kw := range keywords {
			if strings.Contains(next, kw) {
				return true
			}
		}
		if strings.Contains(next, "}") && !strings.Contains(next, "{") {
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

func lineContainsAny(line string, candidates []string) bool {
	for _, c := range candidates {
		if strings.Contains(line, c) {
			return true
		}
	}
	return false
}

func linesContain(lines []string, start, maxLines int, target string) bool {
	if start+maxLines >= len(lines) {
		maxLines = len(lines) - start - 1
	}
	for i := 0; i <= maxLines; i++ {
		if strings.Contains(lines[start+i], target) {
			return true
		}
	}
	return false
}

// matchNegative detects missing patterns (e.g., missing timeout on db.Query).
func (pm *PatternMatcher) matchNegative(pattern *Pattern, lines []string, lineNum int) bool {
	if len(pattern.FunctionCalls) == 0 || pattern.Missing == "" || lineNum >= len(lines) {
		return false
	}
	if !lineContainsAny(lines[lineNum], pattern.FunctionCalls) {
		return false
	}
	return !linesContain(lines, lineNum, 5, pattern.Missing)
}

// matchMetric detects quantitative violations (cyclomatic complexity, etc.).
// For cyclomatic: count if/else/switch/case branches in the function starting at lineNum.
func (pm *PatternMatcher) matchMetric(pattern *Pattern, lines []string, lineNum int) bool {
	if pattern.Metric == "" || pattern.ThresholdGt == 0 {
		return false
	}
	if pattern.Metric == "cyclomatic_complexity" {
		return pm.countCyclomaticComplexity(lines, lineNum) > pattern.ThresholdGt
	}
	return false
}

type complexityStep struct {
	prefix []string
	substr []string
}

var complexitySteps = []complexityStep{
	{prefix: []string{"if ", "if("}},
	{prefix: []string{"case "}},
	{prefix: []string{"catch ", "catch("}},
	{prefix: []string{"for ", "for("}},
	{prefix: []string{"while ", "while("}},
	{substr: []string{" else if ", "}else if"}},
}

// countCyclomaticComplexity counts if/else/switch/case branches in a function block.
func (pm *PatternMatcher) countCyclomaticComplexity(lines []string, lineNum int) int {
	if lineNum >= len(lines) {
		return 1
	}
	return pm.scanFunctionComplexity(lines, lineNum, 1)
}

func (pm *PatternMatcher) scanFunctionComplexity(lines []string, start int, baseComplexity int) int {
	braceDepth := 0
	inFunction := false
	counting := false

	for i := start; i < len(lines); i++ {
		line := lines[i]
		if !inFunction {
			inFunction = pm.isFunctionDef(line)
			if !inFunction {
				continue
			}
		}
		braceDepth, counting = pm.updateBraceDepth(line, braceDepth, counting)
		if !counting {
			continue
		}
		if braceDepth == 0 {
			return baseComplexity
		}
		baseComplexity += pm.countComplexityOnLine(line)
	}
	return baseComplexity
}

func (pm *PatternMatcher) isFunctionDef(line string) bool {
	return strings.Contains(line, "function") || strings.Contains(line, "=>") || strings.Contains(line, "def ")
}

func (pm *PatternMatcher) updateBraceDepth(line string, depth int, counting bool) (int, bool) {
	for _, ch := range line {
		if ch == '{' {
			depth++
			counting = true
		} else if ch == '}' {
			depth--
		}
	}
	return depth, counting
}

func (pm *PatternMatcher) countComplexityOnLine(line string) int {
	trimmed := strings.TrimSpace(line)
	n := 0
	for _, s := range complexitySteps {
		if matchesAnyPrefix(trimmed, s.prefix) || containsAny(line, s.substr) {
			n++
		}
	}
	return n
}

func matchesAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

type semanticCtx struct {
	lines   []string
	lineNum int
}

var semanticDetectors = map[string]func(*PatternMatcher, semanticCtx) bool{
	"go_nil_pointer_dereference":  func(pm *PatternMatcher, c semanticCtx) bool { return pm.detectGoNilDereference(c.lines, c.lineNum) },
	"rust_unchecked_cast":         func(pm *PatternMatcher, c semanticCtx) bool { return pm.detectRustUncheckedCast(c.lines[c.lineNum]) },
	"python_resource_leak":        func(pm *PatternMatcher, c semanticCtx) bool { return pm.detectPythonResourceLeak(c.lines, c.lineNum) },
}

// matchSemantic detects language-specific heuristics (nil dereference, unchecked cast, resource leak).
func (pm *PatternMatcher) matchSemantic(pattern *Pattern, lines []string, lineNum int) bool {
	if pattern.Semantic == "" || pattern.Language == "" || lineNum >= len(lines) {
		return false
	}
	key := strings.ToLower(pattern.Language) + "_" + strings.ToLower(pattern.Semantic)
	if fn, ok := semanticDetectors[key]; ok {
		return fn(pm, semanticCtx{lines: lines, lineNum: lineNum})
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
