package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// goGuard returns (trimmed, skip=true) when the line is not Go or is a comment.
func (w *fileWalker) goGuard(line string) (string, bool) {
	if w.lang != analysis.LangGo {
		return "", true
	}
	t := strings.TrimSpace(line)
	return t, strings.HasPrefix(t, "//") || strings.HasPrefix(t, "/*") || strings.HasPrefix(t, "* ")
}

// checkGoFmtPrint flags fmt.Println/Printf/Print in non-test Go files.
// fmt.Print* bypasses the structured logger — same class of issue as console.log in JS.
func (w *fileWalker) checkGoFmtPrint(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	for _, pat := range []string{"fmt.Println(", "fmt.Printf(", "fmt.Print(", "fmt.Fprintf(", "fmt.Fprintln(", "fmt.Fprint("} {
		if strings.Contains(trimmed, pat) {
			w.emitFinding(analysis.Finding{
				Rule:        "go.fmt_print",
				Pillar:      "observability",
				Severity:    analysis.SeverityAdvisory,
				Line:        lineNum,
				Message:     "fmt.Println/Printf in production code — bypasses structured logging",
				Remediation: "Use slog, zap, or zerolog with structured fields and log-level control.",
			})
			return
		}
	}
}

// checkGoPanicInLib flags panic() in non-test Go files.
// panic in library code gives callers no recovery path.
func (w *fileWalker) checkGoPanicInLib(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if isWordCall(trimmed, "panic(") {
		w.emitFinding(analysis.Finding{
			Rule:        "go.panic_in_lib",
			Pillar:      "stability",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "panic() in library code crashes callers with no recovery path",
			Remediation: "Return an error instead of panicking. Reserve panic for programmer errors in init().",
		})
	}
}

// isWordCall reports whether s contains a call to name where the character
// immediately before name (if any) is not a Go identifier character.
// This prevents "nopanic(" from matching a check for "panic(".
func isWordCall(s, name string) bool {
	idx := strings.Index(s, name)
	if idx < 0 {
		return false
	}
	if idx == 0 {
		return true
	}
	prev := s[idx-1]
	return !isGoIdentByte(prev)
}

func isGoIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

// checkGoContextTODO flags context placeholder calls in non-test Go files.
// The placeholder signals incomplete context threading and must not reach production.
func (w *fileWalker) checkGoContextTODO(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if strings.Contains(trimmed, "context.TODO()") {
		w.emitFinding(analysis.Finding{
			Rule:        "go.context_todo",
			Pillar:      "stability",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "context.TODO() must be replaced before production — signals incomplete context threading",
			Remediation: "Replace with the ctx propagated from the calling function.",
		})
	}
}

// checkGoDeferInLoop performs stateful multi-line detection of defer inside a for loop.
// Deferred calls in a loop accumulate until the enclosing function returns, not per-iteration —
// a resource leak for file handles, locks, and connections.
func (w *fileWalker) checkGoDeferInLoop(lines []string) {
	if w.lang != analysis.LangGo {
		return
	}
	s := &goDeferLoopState{}
	for i, line := range lines {
		if lineNum := i + 1; s.process(line) {
			w.emitFinding(analysis.Finding{
				Rule:        "go.defer_in_loop",
				Pillar:      "stability",
				Severity:    analysis.SeverityMajor,
				Line:        lineNum,
				Message:     "defer inside a for loop — deferred calls accumulate until function return, not per-iteration",
				Remediation: "Extract the loop body to a helper function so defer executes per-iteration.",
			})
		}
	}
}

type goDeferLoopState struct {
	depth      int
	loopDepths []int
	funcDepths []int // func literal/goroutine boundary depths inside a loop
}

// process returns true when the line contains a bare defer directly inside a
// for-loop scope. defer inside a goroutine (go func(){...}()) is excluded
// because the deferred call runs when the goroutine exits, not the loop.
func (s *goDeferLoopState) process(line string) bool {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") {
		return false
	}
	if isGoForLoopOpen(trimmed, line) {
		s.loopDepths = append(s.loopDepths, s.depth)
	}
	if len(s.loopDepths) > 0 && isGoFuncLiteralOpen(trimmed, line) {
		s.funcDepths = append(s.funcDepths, s.depth)
	}
	// Capture before trackBraces so single-line "go func(){ defer }()" is
	// correctly seen as inside a func literal before the closing brace pops it.
	inLoop := len(s.loopDepths) > 0
	inFuncLit := len(s.funcDepths) > 0
	hasDefer := strings.Contains(trimmed, "defer ")
	s.trackBraces(line)
	return inLoop && !inFuncLit && hasDefer
}

func (s *goDeferLoopState) trackBraces(line string) {
	for _, ch := range line {
		switch ch {
		case '{':
			s.depth++
		case '}':
			s.depth--
			if len(s.funcDepths) > 0 && s.depth <= s.funcDepths[len(s.funcDepths)-1] {
				s.funcDepths = s.funcDepths[:len(s.funcDepths)-1]
			}
			if len(s.loopDepths) > 0 && s.depth <= s.loopDepths[len(s.loopDepths)-1] {
				s.loopDepths = s.loopDepths[:len(s.loopDepths)-1]
			}
		}
	}
}

func isGoForLoopOpen(trimmed, line string) bool {
	isFor := strings.HasPrefix(trimmed, "for ") || trimmed == "for {" || strings.HasPrefix(trimmed, "for{")
	return isFor && strings.Contains(line, "{")
}

// isGoFuncLiteralOpen detects func literal openings (go func(){, func(){, func(args){)
// so deferred calls inside them are excluded from the loop defer check.
func isGoFuncLiteralOpen(trimmed, line string) bool {
	hasFuncKw := strings.Contains(trimmed, "func(") || strings.Contains(trimmed, "func (")
	return hasFuncKw && strings.Contains(line, "{")
}


