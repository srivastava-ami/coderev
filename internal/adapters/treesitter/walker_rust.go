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
