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
