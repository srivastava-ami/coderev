package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func (w *fileWalker) checkThrowLiteral(line string, lineNum int) {
	trimmed, skip := w.jsTSGuard(line)
	if skip {
		return
	}
	if !strings.HasPrefix(trimmed, "throw ") {
		return
	}
	after := strings.TrimSpace(strings.TrimPrefix(trimmed, "throw "))
	if strings.HasPrefix(after, `"`) || strings.HasPrefix(after, "`") || strings.HasPrefix(after, "'") ||
		(len(after) > 0 && after[0] >= '0' && after[0] <= '9') {
		w.emitFinding(analysis.Finding{
			Rule:        "stability.no_throw_literal",
			Pillar:      "stability",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "throwing a literal (string/number) — callers cannot catch by type",
			Remediation: "Throw an Error object: throw new Error('message') or a custom Error subclass.",
		})
	}
}

func (w *fileWalker) checkFloatingPromise(line string, lineNum int) {
	trimmed, skip := w.jsTSGuard(line)
	if skip {
		return
	}
	// Heuristic: statement-level async call not anchored
	if !strings.Contains(line, "(") {
		return
	}
	if strings.Contains(line, "await ") || strings.Contains(line, "return ") ||
		strings.Contains(line, ".then(") || strings.Contains(line, ".catch(") ||
		strings.Contains(line, "= ") || strings.Contains(line, "void ") {
		return
	}
	// Looks like a bare async function call: ends with ); or )
	t := strings.TrimSuffix(strings.TrimSuffix(trimmed, ";"), "")
	if strings.HasSuffix(t, ")") && (strings.Contains(trimmed, "Async(") ||
		strings.Contains(trimmed, "async(") || strings.HasPrefix(trimmed, "fetch(") ||
		strings.HasSuffix(strings.TrimSuffix(trimmed, ";"), "()")) {
		w.emitFinding(analysis.Finding{
			Rule:        "stability.no_floating_promise",
			Pillar:      "stability",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "possible unhandled promise — async call result is not awaited or caught",
			Remediation: "Add await, return, .catch(), or void if intentionally fire-and-forget.",
		})
	}
}

// checkAwaitInLoop performs a stateful multi-line scan for `await` inside loops.
func (w *fileWalker) checkAwaitInLoop(lines []string) {
	if w.lang != analysis.LangTypeScript && w.lang != analysis.LangJavaScript {
		return
	}
	state := &loopScanState{}
	for i, line := range lines {
		state.process(line, i+1, w)
	}
}

type loopScanState struct {
	depth      int
	loopDepths []int
}

func (s *loopScanState) process(line string, lineNum int, w *fileWalker) {
	trimmed := strings.TrimSpace(line)
	s.checkLoopEntry(trimmed, s.depth, line)
	s.trackBraces(line)
	s.checkAwait(line, trimmed, lineNum, w)
}

func (s *loopScanState) checkLoopEntry(trimmed string, depthBefore int, line string) {
	isLoop := strings.Contains(trimmed, "for (") || strings.Contains(trimmed, "for(") ||
		strings.Contains(trimmed, "while (") || strings.Contains(trimmed, "while(")
	if isLoop && strings.Contains(line, "{") {
		s.loopDepths = append(s.loopDepths, depthBefore)
	}
}

func (s *loopScanState) trackBraces(line string) {
	for _, ch := range line {
		s.trackBrace(ch)
	}
}

func (s *loopScanState) trackBrace(ch rune) {
	if ch == '{' {
		s.depth++
		return
	}
	if ch == '}' {
		s.depth--
		s.popLoopIfClosed()
	}
}

func (s *loopScanState) popLoopIfClosed() {
	if len(s.loopDepths) > 0 && s.depth <= s.loopDepths[len(s.loopDepths)-1] {
		s.loopDepths = s.loopDepths[:len(s.loopDepths)-1]
	}
}

func (s *loopScanState) checkAwait(line, trimmed string, lineNum int, w *fileWalker) {
	if len(s.loopDepths) == 0 {
		return
	}
	if !strings.Contains(line, "await ") {
		return
	}
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "stability.no_await_in_loop",
		Pillar:      "stability",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     "await inside a loop serialises async calls — use Promise.all() for concurrency",
		Remediation: "Collect promises in an array and await Promise.all(promises) outside the loop.",
	})
}
