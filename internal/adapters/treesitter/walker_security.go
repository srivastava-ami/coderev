package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func (w *fileWalker) checkEval(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	if strings.Contains(line, "eval(") || strings.Contains(line, "new Function(") {
		w.emitFinding(analysis.Finding{
			Rule:        "security.no_eval",
			Pillar:      "security",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "eval() / new Function() executes arbitrary code — injection vector",
			Remediation: "Replace with a safe alternative: JSON.parse, a lookup table, or a sandboxed evaluator.",
		})
	}
}

func (w *fileWalker) checkInnerHTML(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	if strings.Contains(line, ".innerHTML =") || strings.Contains(line, ".innerHTML=") {
		w.emitFinding(analysis.Finding{
			Rule:        "security.no_inner_html",
			Pillar:      "security",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "innerHTML assignment is an XSS vector — never set from user-controlled data",
			Remediation: "Use textContent for text, createElement+appendChild for DOM nodes, or a sanitiser library.",
		})
	}
}

func (w *fileWalker) checkWeakCrypto(line string, lineNum int) {
	if codeLineSkip(line) {
		return
	}
	for _, pat := range []string{`"MD5"`, `'MD5'`, `"SHA1"`, `'SHA1'`, `"SHA-1"`, `'SHA-1'`, "md5(", "sha1("} {
		if strings.Contains(line, pat) {
			w.emitFinding(analysis.Finding{
				Rule:        "security.no_weak_crypto",
				Pillar:      "security",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "MD5/SHA-1 are cryptographically broken — do not use for security-sensitive operations",
				Remediation: "Use SHA-256 or stronger (SHA-384, SHA-512, BLAKE2). For passwords use bcrypt/argon2.",
			})
			return
		}
	}
}

func (w *fileWalker) checkPrototypePollution(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	if strings.Contains(line, ".__proto__") || strings.Contains(line, `["__proto__"]`) || strings.Contains(line, `["constructor"]`) {
		w.emitFinding(analysis.Finding{
			Rule:        "security.no_prototype_pollution",
			Pillar:      "security",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "prototype pollution risk — direct __proto__ or constructor manipulation",
			Remediation: "Use Object.create(null) for safe maps; validate user-supplied keys before property assignment.",
		})
	}
}
