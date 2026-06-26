package treesitter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// walker_injection.go ports the destructuring-default form of coderev's owned
// semgrep rule security.secret_fallback_literal to a native tree-sitter walker.
//
// The dot-notation (process.env.KEY ?? 'val') and bracket-notation
// (process.env['KEY'] ?? 'val') forms are already native in
// walker_security_fallback.go. The remaining form — only ever covered by the
// owned semgrep YAML — is the destructuring default: a const, let or var that
// destructures process.env and gives a secret-named key a string-literal
// default value, possibly spanning several lines.
//
// Porting it here makes the native engine the DEFAULT for security.pattern.*;
// semgrep is no longer required for the owned rule set.

// reDestructureEnv matches a `const|let|var { ... } = process.env` block.
// [^{}]* spans newlines (a newline is neither '{' nor '}'), so multi-line
// destructuring is captured in group 1 without needing the (?s) flag.
var reDestructureEnv = regexp.MustCompile(
	`(?:const|let|var)\s*\{([^{}]*)\}\s*=\s*process\.env`,
)

// reDestructureDefault matches a key with a string-literal default inside a
// destructuring block. The optional rename group handles a renamed key (a name
// followed by a local alias and then the default); group 1 is always the
// environment variable name. The literal alternation requires at least one
// character, so an empty default never matches — mirroring the semgrep pattern.
var reDestructureDefault = regexp.MustCompile(
	`([A-Za-z_$][A-Za-z0-9_$]*)\s*(?::\s*[A-Za-z_$][A-Za-z0-9_$]*\s*)?=\s*` +
		"(?:'[^']+'|\"[^\"]+\"|`[^`]+`)",
)

// checkInjectionPatterns runs the native ports of coderev's owned semgrep
// injection/pattern rules. Registered at the end of checkPatterns().
func (w *fileWalker) checkInjectionPatterns(lines []string) {
	if w.lang != analysis.LangTypeScript && w.lang != analysis.LangJavaScript {
		return
	}
	if isTestFile(w.file) {
		return
	}
	w.checkSecretFallbackDestructure(lines)
}

// checkSecretFallbackDestructure flags secret-ish env vars that carry a
// non-empty string-literal default inside a `{ ... } = process.env`
// destructuring. Same rule ID / severity / test-skip behavior as the other
// secret_fallback_literal forms: blocker by default, advisory inside a
// NODE_ENV !== 'production' guard.
func (w *fileWalker) checkSecretFallbackDestructure(lines []string) {
	src := strings.Join(lines, "\n")
	guard := guardMask(lines)
	for _, outer := range reDestructureEnv.FindAllStringSubmatchIndex(src, -1) {
		contentStart, contentEnd := outer[2], outer[3]
		if contentStart < 0 {
			continue
		}
		content := src[contentStart:contentEnd]
		for _, inner := range reDestructureDefault.FindAllStringSubmatchIndex(content, -1) {
			key := content[inner[2]:inner[3]]
			if !reSecretKey.MatchString(key) {
				continue
			}
			lineNum := strings.Count(src[:contentStart+inner[2]], "\n") + 1
			sev := analysis.SeverityBlocker
			if lineNum-1 < len(guard) && guard[lineNum-1] {
				sev = analysis.SeverityAdvisory
			}
			w.emitFinding(analysis.Finding{
				Rule:     "security.secret_fallback_literal",
				Pillar:   "security",
				Severity: sev,
				Line:     lineNum,
				Message: fmt.Sprintf(
					"env var %s has a hardcoded literal fallback in a destructuring default — "+
						"if unset in a deployed environment this uses a known, source-visible value; "+
						"fail closed (throw) instead", key),
				Remediation: "Destructure without a default and validate explicitly: const { " + key +
					" } = process.env; if (!" + key + ") throw new Error('" + key + " is required in this environment');",
			})
		}
	}
}

// guardMask returns, per line, whether that line sits inside a
// NODE_ENV !== 'production' (or === 'development') guard. It reuses the brace /
// guard primitives of secretFallbackState so the destructuring port downgrades
// to advisory under exactly the same conditions as the line-scanner forms.
func guardMask(lines []string) []bool {
	s := &secretFallbackState{}
	mask := make([]bool, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			s.trackBraces(line)
			s.popClosed()
			mask[i] = s.inGuard()
			continue
		}
		s.checkGuardEntry(line)
		s.trackBraces(line)
		s.popClosed()
		mask[i] = s.inGuard()
	}
	return mask
}
