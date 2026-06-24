package treesitter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

var reSecretKey = regexp.MustCompile(
	`(?i)(secret|password|passwd|token|api[_-]?key|private[_-]?key|` +
		`client[_-]?secret|credential|jwt|signing[_-]?key|encryption[_-]?key)`,
)

// reDotFallback matches: process.env.KEY ?? 'val' / process.env.KEY || 'val'
// [^']+? requires at least one character — empty fallback does not match.
var reDotFallback = regexp.MustCompile(
	`process\.env\.([A-Za-z_][A-Za-z0-9_]*)\s*(?:\?\?|\|\|)\s*` +
		"(?:'[^']+?'|\"[^\"]+?\"|`[^`]+?`)",
)

// reBracketFallback matches: process.env["KEY"] ?? 'val'
var reBracketFallback = regexp.MustCompile(
	`process\.env\[["']([A-Za-z_][A-Za-z0-9_]*)["']\]\s*(?:\?\?|\|\|)\s*` +
		"(?:'[^']+?'|\"[^\"]+?\"|`[^`]+?`)",
)

func (w *fileWalker) checkSecretFallbackInEnv(lines []string) {
	if w.lang != analysis.LangTypeScript && w.lang != analysis.LangJavaScript {
		return
	}
	if isTestFile(w.file) {
		return
	}
	s := &secretFallbackState{}
	for i, line := range lines {
		s.process(line, i+1, w)
	}
}

type secretFallbackState struct {
	depth       int
	guardDepths []int
}

func (s *secretFallbackState) inGuard() bool { return len(s.guardDepths) > 0 }

func (s *secretFallbackState) process(line string, lineNum int, w *fileWalker) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
		s.trackBraces(line)
		s.popClosed()
		return
	}
	s.checkGuardEntry(line)
	s.trackBraces(line)
	s.popClosed()
	s.checkLine(line, lineNum, w)
}

func (s *secretFallbackState) checkGuardEntry(line string) {
	if !strings.Contains(line, "NODE_ENV") || !strings.Contains(line, "{") {
		return
	}
	if strings.Contains(line, "!== 'production'") ||
		strings.Contains(line, "!== \"production\"") ||
		strings.Contains(line, "!= 'production'") ||
		strings.Contains(line, "!= \"production\"") ||
		strings.Contains(line, "=== 'development'") ||
		strings.Contains(line, "=== \"development\"") {
		s.guardDepths = append(s.guardDepths, s.depth)
	}
}

func (s *secretFallbackState) trackBraces(line string) {
	for _, ch := range line {
		if ch == '{' {
			s.depth++
		} else if ch == '}' {
			s.depth--
		}
	}
}

func (s *secretFallbackState) popClosed() {
	for len(s.guardDepths) > 0 && s.depth <= s.guardDepths[len(s.guardDepths)-1] {
		s.guardDepths = s.guardDepths[:len(s.guardDepths)-1]
	}
}

func (s *secretFallbackState) checkLine(line string, lineNum int, w *fileWalker) {
	var key string
	if m := reDotFallback.FindStringSubmatch(line); len(m) > 1 {
		key = m[1]
	} else if m := reBracketFallback.FindStringSubmatch(line); len(m) > 1 {
		key = m[1]
	}
	if key == "" || !reSecretKey.MatchString(key) {
		return
	}
	sev := analysis.SeverityBlocker
	if s.inGuard() {
		sev = analysis.SeverityAdvisory
	}
	w.emitFinding(analysis.Finding{
		Rule:     "security.secret_fallback_literal",
		Pillar:   "security",
		Severity: sev,
		Line:     lineNum,
		Message: fmt.Sprintf(
			"env var %s has a hardcoded literal fallback — if unset in a deployed environment "+
				"this uses a known, source-visible value; fail closed (throw) instead", key),
		Remediation: "Throw when the variable is missing: if (!process.env." + key +
			") throw new Error('" + key + " is required in this environment');",
	})
}
