package treesitter

import (
	"regexp"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// JavaScript/TypeScript convention checkers

// checkAnyTypeUsage detects bare 'any' types in TypeScript
func (w *fileWalker) checkAnyTypeUsage(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip || w.lang != analysis.LangTypeScript {
		return
	}
	if strings.Contains(line, ": any") || strings.Contains(line, ":any") {
		w.emitFinding(analysis.Finding{
			Rule:        "javascript_conventions.any_type_usage",
			Pillar:      "javascript_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Use of bare 'any' type — loses type safety benefits; use 'unknown' with type narrowing instead",
			Remediation: "Replace 'any' with a specific type or use 'unknown' with explicit type guards.",
		})
	}
}

// checkTypeCoercion detects loose equality comparisons (==) that can cause type coercion bugs
func (w *fileWalker) checkTypeCoercion(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	// Look for == but not === or !==
	if strings.Contains(line, "==") && !strings.Contains(line, "===") && !strings.Contains(line, "!==") {
		// Simple check for " == " pattern
		re := regexp.MustCompile(`\s==\s`)
		if re.MatchString(line) {
			w.emitFinding(analysis.Finding{
				Rule:        "javascript_conventions.type_coercion",
				Pillar:      "javascript_conventions",
				Severity:    analysis.SeverityMajor,
				Line:        lineNum,
				Message:     "Loose equality (==) causes implicit type coercion — use strict equality (===) instead",
				Remediation: "Replace '==' with '===' to enforce strict type comparison.",
			})
		}
	}
}

// checkOptionalChainingOveruse detects unnecessary use of optional chaining
func (w *fileWalker) checkOptionalChainingOveruse(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip || w.lang != analysis.LangTypeScript {
		return
	}
	// Detect chaining on values known to be non-null (like string literals, numbers, etc.)
	re := regexp.MustCompile(`["']\w+["']\s*\?\.\w+`)
	if re.MatchString(line) {
		w.emitFinding(analysis.Finding{
			Rule:        "javascript_conventions.optional_chaining_overuse",
			Pillar:      "javascript_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Optional chaining (?.) on provably non-null value — unnecessarily defensive",
			Remediation: "Use direct property access when the value is known to be non-null.",
		})
	}
}

// checkNullCoalescingCorrect detects potential issues with null coalescing
func (w *fileWalker) checkNullCoalescingCorrect(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	// Detect ?? followed by JSON.parse or similar error-prone calls
	if strings.Contains(line, "??") && strings.Contains(line, "JSON.parse") {
		w.emitFinding(analysis.Finding{
			Rule:        "javascript_conventions.null_coalescing_correct",
			Pillar:      "javascript_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Null coalescing (??) with error-prone function call — consider try/catch",
			Remediation: "Wrap the function call in try/catch or validate input before calling.",
		})
	}
}

// checkTypeAssertionUnsafe detects unsafe type assertions
func (w *fileWalker) checkTypeAssertionUnsafe(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip || w.lang != analysis.LangTypeScript {
		return
	}
	// Detect 'as any' or 'as unknown as' pattern
	if strings.Contains(line, "as any") || strings.Contains(line, "as unknown as") {
		w.emitFinding(analysis.Finding{
			Rule:        "javascript_conventions.type_assertion_unsafe",
			Pillar:      "javascript_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Unsafe type assertion — bypasses type checking",
			Remediation: "Use proper type guards or interfaces instead of assertions.",
		})
	}
}

// checkUnhandledPromise detects promises that are not handled (no .catch or try/await)
func (w *fileWalker) checkUnhandledPromise(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	// Detect pattern: .then( without corresponding .catch or in async context
	if strings.Contains(line, ".then(") && !strings.Contains(line, ".catch") {
		// Check if it's not in try/catch context (simple heuristic)
		if !strings.Contains(line, "try") && !strings.Contains(line, "await") {
			w.emitFinding(analysis.Finding{
				Rule:        "javascript_conventions.unhandled_promise",
				Pillar:      "javascript_conventions",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "Promise without error handler (.catch) — unhandled rejections crash the app",
				Remediation: "Add .catch(err => {}) or use try/await with proper error handling.",
			})
		}
	}
}

// checkAsyncAwaitChaining detects improper chaining of async/await
func (w *fileWalker) checkAsyncAwaitChaining(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	// Detect pattern: await Promise.all(...).then(...) - mixing async/await with .then
	if strings.Contains(line, "await") && strings.Contains(line, ".then(") {
		w.emitFinding(analysis.Finding{
			Rule:        "javascript_conventions.async_await_chain",
			Pillar:      "javascript_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Mixing await with .then() — use consistent async/await or promise chain style",
			Remediation: "Stick to either async/await or promise chains, not both in the same line.",
		})
	}
}

// checkPromiseRaceHazard detects improper use of Promise.race
func (w *fileWalker) checkPromiseRaceHazard(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	if strings.Contains(line, "Promise.race(") {
		w.emitFinding(analysis.Finding{
			Rule:        "javascript_conventions.promise_race_hazard",
			Pillar:      "javascript_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Promise.race() is error-prone — losing promise can cause resource leaks or unhandled rejections",
			Remediation: "Use Promise.any() or Promise.allSettled() or manage all promises explicitly.",
		})
	}
}

// checkCallbackHell detects deeply nested callbacks (3+ levels of .then())
// Emits nodejs_conventions.callback_hell (the primary Node.js convention rule).
func (w *fileWalker) checkCallbackHell(line string, lineNum int) {
	if _, skip := w.jsTSGuard(line); skip {
		return
	}
	// Detect nested callbacks by counting .then( or callback(...) patterns
	thenCount := strings.Count(line, ".then(")
	if thenCount >= 3 {
		w.emitFinding(analysis.Finding{
			Rule:        "javascript_conventions.callback_hell",
			Pillar:      "javascript_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Callback hell — deeply nested .then() calls reduce readability",
			Remediation: "Convert to async/await or extract named functions for clarity.",
		})
	}
}
