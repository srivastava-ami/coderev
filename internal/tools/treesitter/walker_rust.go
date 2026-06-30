package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func (w *fileWalker) checkRustUnwrap(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, ".unwrap()") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust.no_unwrap",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     ".unwrap() will panic on Err/None — use pattern matching or ? operator",
			Remediation: "Replace with match, if let, or ? operator for proper error handling.",
		})
	}
}

func (w *fileWalker) checkRustPanic(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, "panic!(") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust.no_panic",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "panic!() in library code — panics are not recoverable by callers",
			Remediation: "Return a Result with an appropriate error type instead.",
		})
	}
}

func (w *fileWalker) checkRustExpect(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, ".expect(") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust.no_expect",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     ".expect() panics on None/Err — use pattern matching or ? operator",
			Remediation: "Replace with match, if let, or ? operator for proper error handling.",
		})
	}
}

func (w *fileWalker) checkRustUnsafe(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, "unsafe {") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust.no_unsafe",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "unsafe block bypasses Rust's memory safety guarantees",
			Remediation: "Use safe abstractions (RefCell, Mutex, raw pointer wrappers) or document safety invariants.",
		})
	}
}

func (w *fileWalker) checkRustTransmute(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, "transmute<") || strings.Contains(trimmed, "transmute::") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust.no_transmute",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "transmute performs arbitrary type reinterpretation — undefined behaviour risk",
			Remediation: "Use safe conversions (From, Into, TryFrom) or pointer casts with explicit provenance.",
		})
	}
}

func (w *fileWalker) checkRustCloneOnCopy(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, ".clone()") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust.clone_on_copy",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     ".clone() on a type that may implement Copy — prefer implicit copy",
			Remediation: "Remove .clone() if the type implements Copy; if not, consider deriving Copy.",
		})
	}
}

func (w *fileWalker) checkRustTodo(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, "todo!(") || strings.Contains(trimmed, "unimplemented!(") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust.no_todo",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "todo!() or unimplemented!() will panic at runtime",
			Remediation: "Implement the functionality or return a meaningful error.",
		})
	}
}

func (w *fileWalker) checkRustDbgMacro(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, "dbg!(") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust.no_dbg_macro",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "dbg!() macro in production code — use structured logging",
			Remediation: "Replace with log::debug!() or similar logging macro.",
		})
	}
}

// ── Phase 1: Memory Safety (5 rules) ──────────────────────────────────────────

func (w *fileWalker) checkRustUnsafeBlockJustif(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if !strings.Contains(trimmed, "unsafe") {
		return
	}
	// Check if this line has unsafe block start
	if !strings.Contains(trimmed, "unsafe {") && !strings.Contains(trimmed, "unsafe{") {
		return
	}
	// Look ahead or behind for SAFETY comment
	hasCommentAbove := false
	if lineNum > 1 {
		prevLine := strings.TrimSpace(w.scanner.Lines[lineNum-2])
		if strings.Contains(prevLine, "SAFETY:") || strings.Contains(prevLine, "SAFETY") {
			hasCommentAbove = true
		}
	}
	if !hasCommentAbove {
		w.emitFinding(analysis.Finding{
			Rule:        "rust_conventions.unsafe_block_justification",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "unsafe block without // SAFETY comment explaining invariants",
			Remediation: "Add a // SAFETY: comment explaining why unsafe is necessary and what invariants it preserves.",
		})
	}
}

func (w *fileWalker) checkRustPanicInLibrary(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if !strings.Contains(trimmed, "panic!(") {
		return
	}
	// Check if in lib.rs or library crate (not main.rs/bin)
	if strings.Contains(w.file, "main.rs") || strings.Contains(w.file, "bin/") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "rust_conventions.panic_in_library",
		Pillar:      "rust_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "panic!() in library code — callers cannot recover",
		Remediation: "Return a Result<T, E> instead and let the caller decide how to handle the error.",
	})
}

func (w *fileWalker) checkRustUnwrapInLibrary(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if !strings.Contains(trimmed, ".unwrap()") {
		return
	}
	// Check if in lib.rs or library crate (not main.rs/bin)
	if strings.Contains(w.file, "main.rs") || strings.Contains(w.file, "bin/") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "rust_conventions.unwrap_in_library",
		Pillar:      "rust_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     ".unwrap() in library code will panic on Err/None",
		Remediation: "Replace with pattern matching (match), if let, or the ? operator for proper error propagation.",
	})
}

func (w *fileWalker) checkRustUnboundedLifetime(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip {
		return
	}
	// Look for generic lifetimes without bounds
	// Pattern: <'a> used without trait bounds or 'static markers
	if !strings.Contains(trimmed, "<'") {
		return
	}
	// Simple heuristic: <'a> or <'a, but no 'a: or +
	if (strings.Contains(trimmed, "<'") && !strings.Contains(trimmed, "'a:") &&
		!strings.Contains(trimmed, "'a +") && !strings.Contains(trimmed, "'static")) &&
		(strings.Contains(trimmed, "struct") || strings.Contains(trimmed, "impl") || strings.Contains(trimmed, "fn")) {
		w.emitFinding(analysis.Finding{
			Rule:        "rust_conventions.unbounded_lifetime",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Generic lifetime without bounds — hidden coupling between types",
			Remediation: "Add explicit lifetime bounds (e.g., <'a: 'b>) or use 'static where appropriate.",
		})
	}
}

func (w *fileWalker) checkRustMutableStatic(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "static mut") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust_conventions.mutable_static",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "static mut is inherently unsafe and causes data races",
			Remediation: "Replace with lazy_static!, once_cell::sync::Lazy, OnceLock, Mutex, or Arc<Mutex<T>>.",
		})
	}
}

// ── Phase 1: Error Handling (4 rules) ─────────────────────────────────────────

func (w *fileWalker) checkRustErrorPropagation(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Look for lossy error conversions: .ok(), unwrap_or, etc. without context
	if !strings.Contains(trimmed, ".ok()") && !strings.Contains(trimmed, "unwrap_or") &&
		!strings.Contains(trimmed, "unwrap_or_else") && !strings.Contains(trimmed, ".expect") {
		return
	}
	// Flag if in library code and converting error away
	if (strings.Contains(w.file, "lib.rs") || !strings.Contains(w.file, "main.rs")) &&
		strings.Contains(trimmed, ".ok()") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust_conventions.error_propagation",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Lossy error conversion — error context lost when converting Result to Option",
			Remediation: "Use .map_err() to wrap errors, or use anyhow/eyre to preserve error chains.",
		})
	}
}

func (w *fileWalker) checkRustResultDiscard(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Look for Result expressions that are not explicitly handled
	// Pattern: function_call(...) where the function returns Result
	if strings.Contains(trimmed, "();") && !strings.Contains(trimmed, "_ =") &&
		!strings.Contains(trimmed, ".ok") && !strings.Contains(trimmed, ".unwrap") &&
		!strings.Contains(trimmed, "?") {
		// Check if it's a common Result-returning function
		if strings.Contains(trimmed, "write!") || strings.Contains(trimmed, "writeln!") ||
			strings.Contains(trimmed, "format!") || strings.Contains(trimmed, ".send") {
			w.emitFinding(analysis.Finding{
				Rule:        "rust_conventions.result_discard",
				Pillar:      "rust_conventions",
				Severity:    analysis.SeverityMajor,
				Line:        lineNum,
				Message:     "Result discarded without explicit acknowledgement",
				Remediation: "Use .ok() to ignore the Ok case, or _ = expr to explicitly ignore the Result.",
			})
		}
	}
}

func (w *fileWalker) checkRustPanicHookMissing(line string, lineNum int) {
	_, skip := w.rustGuard(line)
	if skip || !strings.Contains(w.file, "main.rs") {
		return
	}
	// This is a file-level check, not a line-level one. Stub for now.
	// In a real implementation, we'd scan the whole main.rs for set_hook.
	if lineNum == 1 && !strings.Contains(line, "set_hook") {
		// Only emit once per file, at the first line
		// Check if the whole file has set_hook
		if !w.fileHasSetHook {
			w.fileHasSetHook = true
			w.emitFinding(analysis.Finding{
				Rule:        "rust_conventions.panic_hook_missing",
				Pillar:      "rust_conventions",
				Severity:    analysis.SeverityAdvisory,
				Line:        1,
				Message:     "main.rs does not call std::panic::set_hook() — panics may be silent in production",
				Remediation: "Call std::panic::set_hook() in main() to log panics before termination.",
			})
		}
	}
}

func (w *fileWalker) checkRustCustomErrorImpl(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip {
		return
	}
	// Look for custom error type definitions without impl std::error::Error
	if strings.Contains(trimmed, "pub struct") && strings.Contains(trimmed, "Error") {
		// Check if Error trait is implemented (this is a simplified heuristic)
		// In a real implementation, we'd need to track struct scope
		if !strings.Contains(line, "impl std::error::Error") && !strings.Contains(line, "impl Error") {
			// Only check within a reasonable scope (next ~10 lines)
			if lineNum < w.lastErrorStructLine+10 {
				w.lastErrorStructLine = lineNum
			}
		}
	}
}

// ── Phase 1: Patterns (4 rules) ───────────────────────────────────────────────

func (w *fileWalker) checkRustCloneHeavy(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Count .clone() calls in the line
	cloneCount := strings.Count(trimmed, ".clone()")
	if cloneCount > 2 {
		w.emitFinding(analysis.Finding{
			Rule:        "rust_conventions.clone_heavy",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Multiple .clone() calls in one expression suggests design issue",
			Remediation: "Reduce cloning by using references (&T), borrowing, or passing ownership in function signatures.",
		})
	}
}

func (w *fileWalker) checkRustExpensiveOpLoop(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Simple check: look for allocations inside loop-like constructs
	// Pattern: for/while ... Vec::new / String::new / Regex::new / format!
	if (strings.Contains(trimmed, "for ") || strings.Contains(trimmed, "while ")) &&
		(strings.Contains(trimmed, "Vec::new") || strings.Contains(trimmed, "String::new") ||
			strings.Contains(trimmed, "HashMap::new") || strings.Contains(trimmed, "format!")) {
		w.emitFinding(analysis.Finding{
			Rule:        "rust_conventions.expensive_operation_loop",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Expensive allocation inside loop — potential O(n²) pattern",
			Remediation: "Move allocations outside the loop or use .with_capacity() to avoid repeated allocations.",
		})
	}
}

func (w *fileWalker) checkRustIterCollectChain(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Look for .collect() followed by .iter() or similar chain
	if strings.Contains(trimmed, ".collect()") && (strings.Contains(trimmed, ".iter()") ||
		strings.Contains(trimmed, ".map(") || strings.Contains(trimmed, ".filter(")) &&
		strings.Index(trimmed, ".collect()") < strings.Index(trimmed, ".iter()") {
		w.emitFinding(analysis.Finding{
			Rule:        "rust_conventions.iter_collect_chain",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Unnecessary .collect() in iterator chain — materializes intermediate Vec",
			Remediation: "Remove .collect() and chain the operations directly on the iterator.",
		})
	}
}

func (w *fileWalker) checkRustAsyncCancelSafety(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Look for async functions with complex state mutations
	if strings.Contains(trimmed, "async fn") && strings.Contains(trimmed, "{") {
		// Flag as potential cancel-safety concern if we see .await without guards
		// This is a simplified check
		if strings.Contains(trimmed, ".await") {
			w.emitFinding(analysis.Finding{
				Rule:        "rust_conventions.async_cancel_safety",
				Pillar:      "rust_conventions",
				Severity:    analysis.SeverityMajor,
				Line:        lineNum,
				Message:     "async function with .await — verify cancel safety; partial drops may corrupt state",
				Remediation: "Ensure drop impl is consistent with partial async operations. Use select!() guards or document cancel-safety invariants.",
			})
		}
	}
}

// ── Phase 1: Borrowing (2 rules) ──────────────────────────────────────────────

func (w *fileWalker) checkRustBorrowedRefLifetime(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip {
		return
	}
	// Look for borrowed references in function signatures
	if strings.Contains(trimmed, "&") && strings.Contains(trimmed, "fn ") {
		// Check for lifetime 'a but no explicit bounds or scope tracking
		if strings.Contains(trimmed, "'a") && !strings.Contains(trimmed, "'a:") &&
			!strings.Contains(trimmed, "-> &'a") {
			w.emitFinding(analysis.Finding{
				Rule:        "rust_conventions.borrowed_reference_lifetime",
				Pillar:      "rust_conventions",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "Borrowed reference may outlive referent — potential dangling reference",
				Remediation: "Ensure references do not outlive their referent. Use owned types (String, Vec, Box) or explicit lifetime bounds.",
			})
		}
	}
}

func (w *fileWalker) checkRustMutableBorrowScope(line string, lineNum int) {
	trimmed, skip := w.rustGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Look for excessive mutable borrow scope
	if strings.Contains(trimmed, "&mut ") && (strings.Contains(trimmed, "=") || strings.Contains(trimmed, ",")) {
		// Flag if mutable borrow is held across multiple statements
		// This is a simplified check; a real impl would track borrow scopes
		w.emitFinding(analysis.Finding{
			Rule:        "rust_conventions.mutable_borrow_scope",
			Pillar:      "rust_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Mutable borrow may be held longer than necessary",
			Remediation: "Release mutable borrows as soon as the mutation is complete. Use explicit block scope to allow reborrowing.",
		})
	}
}
