package treesitter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func (w *fileWalker) checkPatterns() {
	lines := strings.Split(string(w.src), "\n")
	for i, line := range lines {
		lineNum := i + 1
		w.checkConsolLog(line, lineNum)
		w.checkAnyType(line, lineNum)
		w.checkEmptyCatch(line, lineNum)
		w.checkHardcodedURL(line, lineNum)
		w.checkEval(line, lineNum)
		w.checkInnerHTML(line, lineNum)
		w.checkWeakCrypto(line, lineNum)
		w.checkPrototypePollution(line, lineNum)
		w.checkThrowLiteral(line, lineNum)
		w.checkNonNullAssertion(line, lineNum)
		w.checkForceCast(line, lineNum)
		w.checkDeepImport(line, lineNum)
		// Go-specific checks
		w.checkGoFmtPrint(line, lineNum)
		w.checkGoPanicInLib(line, lineNum)
		w.checkGoSQLStringConcat(line, lineNum)
		w.checkGoContextTODO(line, lineNum)
		w.checkFloatingPromise(line, lineNum)
	}
	w.checkAwaitInLoop(lines)
	w.checkGoDeferInLoop(lines)
}

func (w *fileWalker) checkNonNullAssertion(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip || w.lang != analysis.LangTypeScript {
		return
	}
	if strings.Contains(line, "!.") || strings.Contains(line, "![") {
		w.emitFinding(analysis.Finding{
			Rule:        "type_safety.no_non_null_assertion",
			Pillar:      "type_safety",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "non-null assertion (!) suppresses TypeScript null checks — potential runtime null dereference",
			Remediation: "Use optional chaining (?.) with a null check or narrow the type explicitly.",
		})
	}
}

func (w *fileWalker) checkForceCast(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip || w.lang != analysis.LangTypeScript {
		return
	}
	if strings.Contains(line, "as unknown as ") {
		w.emitFinding(analysis.Finding{
			Rule:        "type_safety.no_force_cast",
			Pillar:      "type_safety",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "double cast (as unknown as T) defeats the type system entirely",
			Remediation: "Use a proper type guard or narrow the type through legitimate means.",
		})
	}
}

func (w *fileWalker) checkConsolLog(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	if !strings.Contains(line, "console.log") && !strings.Contains(line, "console.error") && !strings.Contains(line, "console.warn") {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "observability.logging", Pillar: "observability", Severity: analysis.SeverityBlocker, Line: lineNum,
		Message:     "console.log/error/warn in production code — use structured logger",
		Remediation: "Replace with injected logger that emits structured JSON with correlationId."})
}

func (w *fileWalker) checkAnyType(line string, lineNum int) {
	if w.lang != analysis.LangTypeScript {
		return
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") {
		return
	}
	for _, pat := range []string{": any", "as any", "<any>", ": any[", "Array<any"} {
		if strings.Contains(line, pat) {
			w.emitFinding(analysis.Finding{Rule: "type_safety.no_any", Pillar: "type_safety", Severity: analysis.SeverityBlocker, Line: lineNum,
				Message:     fmt.Sprintf("'any' type usage: %s", trimmed),
				Remediation: "Use 'unknown' with type guards, or generate types from schema."})
			return
		}
	}
	if strings.Contains(line, "// @ts-ignore") && !strings.Contains(line, "// @ts-ignore:") {
		w.emitFinding(analysis.Finding{Rule: "type_safety.no_any", Pillar: "type_safety", Severity: analysis.SeverityMajor, Line: lineNum,
			Message:     "@ts-ignore without justification comment",
			Remediation: "Add a comment explaining why the suppression is necessary."})
	}
}

func (w *fileWalker) checkEmptyCatch(line string, lineNum int) {
	trimmed, skip := w.jsTSGuard(line)
	if skip {
		return
	}
	if !strings.HasSuffix(trimmed, "catch (e) {}") && trimmed != "} catch {}" && !strings.HasSuffix(trimmed, "catch(e){}") {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "stability.error_handling", Pillar: "stability", Severity: analysis.SeverityBlocker, Line: lineNum,
		Message:     "empty catch block silently swallows errors",
		Remediation: "Handle the error or re-throw with context. Never swallow silently."})
}

func isTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".test.js")
}

func (w *fileWalker) checkHardcodedURL(line string, lineNum int) {
	if isTestFile(w.file) || codeLineSkip(line) {
		return
	}
	trimmed := strings.TrimSpace(line)
	for _, prefix := range []string{`"http://`, `"https://`, `'http://`, `'https://`} {
		idx := strings.Index(line, prefix)
		if idx < 0 {
			continue
		}
		urlPart := line[idx+len(prefix):]
		// Require a domain dot — avoids flagging URL prefix patterns in analysis code
		if !strings.Contains(urlPart, ".") {
			continue
		}
		if strings.Contains(urlPart, "localhost") || strings.Contains(urlPart, "127.0.0.1") {
			continue
		}
		w.emitFinding(analysis.Finding{Rule: "hardcoding.urls_and_paths", Pillar: "hardcoding", Severity: analysis.SeverityBlocker, Line: lineNum,
			Message:     fmt.Sprintf("hardcoded external URL detected: %s", trimmed),
			Remediation: "Read base URL from configuration / environment variable."})
		return
	}
}
