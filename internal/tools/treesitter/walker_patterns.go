package treesitter

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

var linePatternChecks = func(w *fileWalker) []func(string, int) {
	return []func(string, int){
		w.checkConsolLog, w.checkAnyType, w.checkEmptyCatch, w.checkHardcodedURL,
		w.checkEval, w.checkInnerHTML, w.checkWeakCrypto, w.checkPrototypePollution,
		w.checkThrowLiteral, w.checkNonNullAssertion, w.checkForceCast, w.checkDeepImport,
		w.checkGoFmtPrint, w.checkGoPanicInLib, w.checkGoSQLStringConcat,
		w.checkGoContextTODO, w.checkGoFmtErrorfNoFormat, w.checkFloatingPromise,
		w.checkGoGoroutineLeak, w.checkGoDeadlockPattern,
		w.checkGoDeferPanic, w.checkGoUncheckedError, w.checkGoInterfaceBloat,
		w.checkGoUnclosedBody, w.checkGoFileDescriptorLeak, w.checkGoNilSliceIteration,
		w.checkPythonPrint, w.checkPythonBareExcept, w.checkPythonEvalExec,
		w.checkPythonSQLStringConcat, w.checkPythonSubprocess, w.checkPythonMutableDefault,
		w.checkPythonWildcardImport,
		w.checkPythonConventionTypeHintsMissing, w.checkPythonConventionNoneCoercion,
		w.checkPythonConventionDynamicAttribute, w.checkPythonConventionTypeInconsistency,
		w.checkPythonConventionDuckTypingUnsafe, w.checkPythonConventionUnclosedAsyncResource,
		w.checkPythonConventionAsyncDeadlock, w.checkPythonConventionTaskLeak,
		w.checkPythonConventionEventLoopMismatch, w.checkPythonConventionBareExcept,
		w.checkPythonConventionExceptionSwallowing, w.checkPythonConventionExceptionChaining,
		w.checkPythonConventionFinallySideEffects, w.checkPythonConventionCircularImport,
		w.checkPythonConventionImportOrder, w.checkPythonConventionUnusedImport,
		w.checkPythonConventionResourceLeak, w.checkPythonConventionUnboundedGrowth,
		w.checkRustUnwrap, w.checkRustPanic, w.checkRustExpect, w.checkRustUnsafe,
		w.checkRustTransmute, w.checkRustCloneOnCopy, w.checkRustTodo, w.checkRustDbgMacro,
		w.checkRustUnsafeBlockJustif, w.checkRustPanicInLibrary, w.checkRustUnwrapInLibrary,
		w.checkRustMutableStatic, w.checkRustErrorPropagation, w.checkRustCloneHeavy,
		w.checkRustExpensiveOpLoop, w.checkRustIterCollectChain, w.checkRustAsyncCancelSafety,
		w.checkAnyTypeUsage, w.checkTypeCoercion, w.checkOptionalChainingOveruse,
		w.checkNullCoalescingCorrect, w.checkTypeAssertionUnsafe, w.checkUnhandledPromise,
		w.checkAsyncAwaitChaining, w.checkPromiseRaceHazard, w.checkCallbackHell,
		w.checkStreamNotPiped, w.checkBackpressureIgnored, w.checkStreamErrorUnhandled,
		w.checkStreamLeak, w.checkEventListenerLeak, w.checkOnceVsOn,
		w.checkErrorEventUnhandled, w.checkPromiseSwallowing, w.checkAsyncIteratorIncomplete,
		w.checkConcurrentOperationsUnbounded, w.checkMemoryLeakTimers,
		w.checkUnboundedBuffer, w.checkCpuBlocking,
	}
}

func (w *fileWalker) checkPatterns() {
	lines := strings.Split(string(w.src), "\n")
	for i, line := range lines {
		for _, check := range linePatternChecks(w) {
			check(line, i+1)
		}
	}
	w.checkMultiLinePatterns(lines)
	w.applyTOMLMatcher(lines)
}

func (w *fileWalker) checkMultiLinePatterns(lines []string) {
	w.checkAwaitInLoop(lines)
	w.checkGoDeferInLoop(lines)
	w.checkGoIOCopyNoLimit(lines)
	w.checkSecretFallbackInEnv(lines)
	w.checkInjectionPatterns(lines)
	w.checkTerraformConventions(lines)
	w.checkCallbackHellNJS(lines)
}

func (w *fileWalker) applyTOMLMatcher(lines []string) {
	if w.matcher == nil {
		return
	}
	findings, err := w.matcher.Match(string(w.src), w.file, w.lang)
	if err != nil {
		return
	}
	for _, pf := range findings {
		w.emitFinding(analysis.Finding{
			Rule: pf.Rule, Pillar: pf.Pillar, Severity: analysis.Severity(pf.Severity),
			Line: pf.Line, Message: pf.Message, Remediation: pf.Remediation,
		})
	}
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
	sev := analysis.SeverityBlocker
	if isTestFile(w.file) {
		sev = analysis.SeverityAdvisory
	}
	for _, pat := range []string{": any", "as any", "<any>", ": any[", "Array<any"} {
		if strings.Contains(line, pat) {
			w.emitFinding(analysis.Finding{Rule: "type_safety.no_any", Pillar: "type_safety", Severity: sev, Line: lineNum,
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
	if strings.HasSuffix(base, "_test.go") || strings.HasSuffix(base, "_test.rs") ||
		strings.HasSuffix(base, ".spec.ts") || strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".spec.tsx") || strings.HasSuffix(base, ".test.tsx") ||
		strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".test.js") {
		return true
	}
	normalized := filepath.ToSlash(path)
	for _, segment := range []string{"e2e", "__tests__", "test"} {
		if strings.Contains(normalized, "/"+segment+"/") || strings.HasPrefix(normalized, segment+"/") {
			return true
		}
	}
	return false
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
