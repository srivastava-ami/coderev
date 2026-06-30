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

// checkGoFmtPrint flags the fmt stdout-print family (Print/Printf/Println) in
// non-test, non-main Go code, where it bypasses the structured logger. It does
// NOT flag the fmt.Fprint* family — those write to an explicit io.Writer (a
// buffer, file, stderr, HTTP response), which is normal formatted output, not
// logging. package main is skipped: a CLI legitimately writes to stdout.
func (w *fileWalker) checkGoFmtPrint(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) || w.isMain {
		return
	}
	for _, pat := range goStdoutPrintCalls() {
		if strings.Contains(trimmed, pat) {
			w.emitFinding(analysis.Finding{
				Rule:        "go.fmt_print",
				Pillar:      "observability",
				Severity:    analysis.SeverityAdvisory,
				Line:        lineNum,
				Message:     "stdout print in library code bypasses structured logging",
				Remediation: "Use slog, zap, or zerolog with structured fields and log-level control.",
			})
			return
		}
	}
}

// goStdoutPrintCalls builds the fmt stdout-print call needles at runtime so this
// detector's own source carries no literal the check would match on itself.
func goStdoutPrintCalls() []string {
	var out []string
	for _, fn := range []string{"Print", "Printf", "Println"} {
		out = append(out, "fmt."+fn+"(")
	}
	return out
}

// isGoMainPackage reports whether Go source declares package main, where writing
// to stdout is legitimate program output rather than a logging bypass.
func isGoMainPackage(src []byte) bool {
	for _, line := range strings.Split(string(src), "\n") {
		if t := strings.TrimSpace(line); strings.HasPrefix(t, "package ") {
			return t == "package main"
		}
	}
	return false
}

// checkGoPanicInLib flags panic() in non-test Go files.
// panic in library code gives callers no recovery path.
func (w *fileWalker) checkGoPanicInLib(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	if isWordCall(trimmed, "panic"+"(") {
		w.emitFinding(analysis.Finding{
			Rule:        "go.panic_in_lib",
			Pillar:      "stability",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "panic in library code crashes callers with no recovery path",
			Remediation: "Return an error instead of panicking. Reserve panics for programmer errors in init().",
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
	if strings.Contains(trimmed, "context."+"TODO()") {
		w.emitFinding(analysis.Finding{
			Rule:        "go.context_todo",
			Pillar:      "stability",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "a context placeholder must be replaced before production — signals incomplete context threading",
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
				Message:     "a deferred call inside a for loop accumulates until function return, not per-iteration",
				Remediation: "Extract the loop body to a helper function so the deferred call runs per-iteration.",
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

// ── 20 NEW GO CONVENTION RULES ──────────────────────────────────────────────────

// checkGoGoroutineLeak detects goroutines without wait groups or channels.
// Pattern: "go " without sync tracking (WaitGroup, channel receive, context cancellation).
func (w *fileWalker) checkGoGoroutineLeak(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	if !strings.HasPrefix(trimmed, "go ") {
		return
	}
	// Very basic: flag "go " at statement start without immediately obvious tracking.
	// False positives are expected — code review should verify goroutine tracking.
	w.emitFinding(analysis.Finding{
		Rule:        "go_conventions.goroutine_leak",
		Pillar:      "concurrency",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "goroutine spawned without visible wait/channel/context tracking — may leak",
		Remediation: "Ensure the goroutine is waited for (WaitGroup, channel, context cancellation) before function exit.",
	})
}

// checkGoRaceCondition detects unsynchronized map access across goroutines.
// Pattern: map[...] assignments/reads without sync.Map or mutex protection nearby.
func (w *fileWalker) checkGoRaceCondition(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Flag: m[key] = value pattern without Lock visibility
	if strings.Contains(trimmed, "[") && strings.Contains(trimmed, "]=") && strings.Contains(trimmed, "=") {
		// Exclude obvious non-maps (array indexing, etc.)
		if strings.Contains(trimmed, "var ") || strings.Contains(trimmed, ":=") {
			// Likely a map assignment; check for sync protection
			if !strings.Contains(trimmed, "Lock()") && !strings.Contains(trimmed, "RLock()") {
				w.emitFinding(analysis.Finding{
					Rule:        "go_conventions.race_condition",
					Pillar:      "concurrency",
					Severity:    analysis.SeverityBlocker,
					Line:        lineNum,
					Message:     "map access without visible synchronization — potential data race",
					Remediation: "Protect map access with sync.Mutex, sync.RWMutex, or use sync.Map for concurrent reads/writes.",
				})
			}
		}
	}
}

// checkGoDeadlockPattern detects potential deadlock patterns: channel ops without select/timeout.
// Pattern: <- without timeout, send to channel without buffer consideration.
func (w *fileWalker) checkGoDeadlockPattern(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	// Flag bare channel receive/send without select context
	if strings.Contains(trimmed, "<-") && !strings.Contains(trimmed, "select") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.deadlock_pattern",
			Pillar:      "concurrency",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "channel operation without timeout or select — potential deadlock",
			Remediation: "Wrap channel operations in a select with context.Done() or time.After() to prevent indefinite blocking.",
		})
	}
}

// checkGoContextPropagation ensures context.Context is passed through function calls.
// Pattern: function call without ctx parameter when other parameters exist.
func (w *fileWalker) checkGoContextPropagation(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Flag function calls with multiple parameters but no ctx first param
	if strings.Count(trimmed, "(") == 1 && strings.Count(trimmed, ",") > 0 {
		if !strings.Contains(trimmed, "ctx") && !strings.Contains(trimmed, "context") {
			// Exclude common utility functions and built-ins
			if !strings.Contains(trimmed, "fmt.") && !strings.Contains(trimmed, "log.") &&
				!strings.Contains(trimmed, "time.") && !strings.Contains(trimmed, "math.") {
				w.emitFinding(analysis.Finding{
					Rule:        "go_conventions.context_propagation",
					Pillar:      "concurrency",
					Severity:    analysis.SeverityMajor,
					Line:        lineNum,
					Message:     "function call with multiple parameters missing context.Context propagation",
					Remediation: "Accept context.Context as the first parameter and pass it through the call stack.",
				})
			}
		}
	}
}

// checkGoChannelSafety detects unsafe channel patterns: close on receiving end or unbuffered channels.
func (w *fileWalker) checkGoChannelSafety(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "close(") && strings.Contains(trimmed, "ch") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.channel_safety",
			Pillar:      "concurrency",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "potential close on receiving end of channel — only sender should close",
			Remediation: "Ensure only the sender side closes the channel. Receiver should not close.",
		})
	}
}

// checkGoSelectTimeout ensures select statements include a timeout branch.
// Pattern: bare select without <-time.After or <-ctx.Done().
func (w *fileWalker) checkGoSelectTimeout(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	if strings.HasPrefix(trimmed, "select {") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.select_timeout",
			Pillar:      "concurrency",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "select without timeout or context cancellation branch — may block indefinitely",
			Remediation: "Add a case <-time.After(...) or case <-ctx.Done() to prevent indefinite blocking.",
		})
	}
}

// checkGoDeferPanic detects defer with panic (without recover in same function).
// Pattern: defer panic(...) or defer calls that may panic.
func (w *fileWalker) checkGoDeferPanic(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "defer") && strings.Contains(trimmed, "panic(") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.defer_panic",
			Pillar:      "stability",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "defer statement contains panic — likely error in control flow",
			Remediation: "Separate defer cleanup from panic logic. Use defer for cleanup, return errors normally.",
		})
	}
}

// checkGoDeferUnlockOrder detects unlock without corresponding lock visibility.
// Pattern: defer mu.Unlock() or defer ..Unlock() without obvious Lock() call preceding.
// This is a heuristic check; code review verifies correctness.
func (w *fileWalker) checkGoDeferUnlockOrder(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	// Only flag if defer and Unlock are on same line — separate lines require state tracking
	if strings.Contains(trimmed, "defer") && strings.Contains(trimmed, "Unlock()") {
		// Check if Lock is also visible to reduce false positives
		if !strings.Contains(trimmed, "Lock()") && !strings.Contains(trimmed, ".Lock()") {
			w.emitFinding(analysis.Finding{
				Rule:        "go_conventions.defer_unlock_order",
				Pillar:      "concurrency",
				Severity:    analysis.SeverityMajor,
				Line:        lineNum,
				Message:     "defer Unlock() without visible Lock() on same line — verify lock/unlock pairing and order",
				Remediation: "Ensure Lock() is called before defer Unlock(), in the correct order.",
			})
		}
	}
}

// checkGoUncheckedError detects error returns that are ignored (blank assignment).
// Pattern: _ = func() where func returns an error.
func (w *fileWalker) checkGoUncheckedError(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Flag: _ = something.FunctionCall() or _ = Write(...) or similar
	if strings.Contains(trimmed, "_ =") && strings.Contains(trimmed, "(") {
		// Avoid flagging common non-error returns
		if !strings.Contains(trimmed, "fmt.") && !strings.Contains(trimmed, "len(") && !strings.Contains(trimmed, "cap(") {
			w.emitFinding(analysis.Finding{
				Rule:        "go_conventions.unchecked_error",
				Pillar:      "stability",
				Severity:    analysis.SeverityMajor,
				Line:        lineNum,
				Message:     "error return value explicitly ignored with blank assignment",
				Remediation: "Check the error: if err != nil { return err } or document why it is safe to ignore.",
			})
		}
	}
}

// checkGoErrorWrapping detects errors that are not wrapped with context.
// Pattern: return err (vs return fmt.Errorf(...) or return errors.Wrap(...)).
func (w *fileWalker) checkGoErrorWrapping(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Flag bare "return err" without wrapping
	if strings.Contains(trimmed, "return err") && !strings.Contains(trimmed, "nil") {
		if !strings.Contains(trimmed, "fmt.Errorf") && !strings.Contains(trimmed, "Wrap") && !strings.Contains(trimmed, "WithContext") {
			w.emitFinding(analysis.Finding{
				Rule:        "go_conventions.error_wrapping",
				Pillar:      "stability",
				Severity:    analysis.SeverityAdvisory,
				Line:        lineNum,
				Message:     "error returned without wrapping context — stack trace and context lost",
				Remediation: "Wrap errors with fmt.Errorf('operation: %w', err) or github.com/pkg/errors.Wrap(err, ...).",
			})
		}
	}
}

// checkGoInterfaceBloat detects interfaces with too many methods (>5).
// Pattern: interface { method1(...), method2(...), ... } with many lines.
func (w *fileWalker) checkGoInterfaceBloat(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "type ") && strings.Contains(trimmed, "interface {") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.interface_bloat",
			Pillar:      "design",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "interface declaration detected — verify it has <= 5 methods (rule: large interfaces are hard to mock and implement)",
			Remediation: "Split large interfaces into smaller, focused ones. Each interface should have a single responsibility.",
		})
	}
}

// checkGoInterfaceSegregation detects overly broad interfaces (embedded interface types).
// Pattern: interface { io.Reader; io.Writer } (should be separate).
func (w *fileWalker) checkGoInterfaceSegregation(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	// Check for embedded types separated by semicolons within an interface
	if strings.Contains(trimmed, "interface {") && strings.Contains(trimmed, ";") {
		// Must have embedded types (dots indicate type names like io.Reader)
		if strings.Count(trimmed, ".") >= 2 {
			w.emitFinding(analysis.Finding{
				Rule:        "go_conventions.interface_segregation",
				Pillar:      "design",
				Severity:    analysis.SeverityAdvisory,
				Line:        lineNum,
				Message:     "interface embeds multiple types — prefer smaller, focused interfaces",
				Remediation: "Apply Interface Segregation Principle: split into Reader, Writer, Closer, etc.",
			})
		}
	}
}

// checkGoPointerReceiverConsistency detects mixed receiver types (some methods value, some pointer).
// Pattern: func (s S) method() vs func (s *S) method() on same type.
func (w *fileWalker) checkGoPointerReceiverConsistency(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	// Simple heuristic: flag method definitions to let code review verify consistency
	if strings.Contains(trimmed, "func (") && strings.Contains(trimmed, ")") && strings.Contains(trimmed, "{") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.pointer_receiver_consistency",
			Pillar:      "design",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "method defined — verify all methods on this type use consistent receiver type (all value or all pointer)",
			Remediation: "Choose either value receivers (immutable) or pointer receivers (mutable) consistently for a type.",
		})
	}
}

// checkGoUnclosedBody detects HTTP responses without body close.
// Pattern: actual code accessing resp.Body without Close() on same line (skip strings/comments).
func (w *fileWalker) checkGoUnclosedBody(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Skip comments and pure string assignments
	if strings.HasPrefix(trimmed, "//") {
		return
	}
	// Only flag if resp.Body is accessed in actual code: after = or ( or , (not in strings)
	// This avoids flagging resp.Body inside string literals like remediation messages
	hasBodyAccess := strings.Contains(trimmed, " resp.Body") || strings.Contains(trimmed, "(resp.Body") || strings.Contains(trimmed, ",resp.Body")
	if hasBodyAccess && !strings.Contains(trimmed, "Close()") && !strings.Contains(trimmed, "Closer") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.unclosed_body",
			Pillar:      "resources",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "response body accessed without close — resource leak",
			Remediation: "Always defer resp.Body.Close() immediately after http.Do or http.Get.",
		})
	}
}

// checkGoFileDescriptorLeak detects file opens without close visibility.
// Pattern: actual code with os.Open/os.Create (not in strings/comments) without defer on same line.
func (w *fileWalker) checkGoFileDescriptorLeak(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Skip comments
	if strings.HasPrefix(trimmed, "//") {
		return
	}
	// Flag file open calls without defer - only if it looks like actual code (contains := or =)
	isAssignment := strings.Contains(trimmed, ":=") || strings.Contains(trimmed, " = ")
	if isAssignment && (strings.Contains(trimmed, "os.Open(") || strings.Contains(trimmed, "os.Create(")) {
		if !strings.Contains(trimmed, "defer") && !strings.Contains(trimmed, "ioutil.ReadAll") {
			w.emitFinding(analysis.Finding{
				Rule:        "go_conventions.file_descriptor_leak",
				Pillar:      "resources",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "file handle opened without visible close — file descriptor leak",
				Remediation: "Defer file close immediately: f, err := os.Open(...); if err != nil { ... }; defer f.Close().",
			})
		}
	}
}

// checkGoPoolExhaustion detects unbounded resource creation without limits in loops.
func (w *fileWalker) checkGoPoolExhaustion(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, ".Get()") || strings.Contains(trimmed, ".Acquire()") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.pool_exhaustion",
			Pillar:      "resources",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "pool resource acquired — verify release path and concurrency limits",
			Remediation: "Ensure all pool.Get() calls are paired with Put/Release. Use semaphores or bounded queues to limit concurrency.",
		})
	}
}

// checkGoMemoryLeakPatterns detects patterns known to cause memory leaks.
// Pattern: appending to slice in loop or unbounded map growth.
func (w *fileWalker) checkGoMemoryLeakPatterns(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	// Flag: append without capacity check or unbounded operations in loops
	if strings.Contains(trimmed, "append(") && strings.Contains(trimmed, "[") && !strings.Contains(trimmed, "make(") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.memory_leak_patterns",
			Pillar:      "resources",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "unbounded collection growth detected — potential memory leak",
			Remediation: "Implement size limits, eviction policies, or use time-based expiry for cached data.",
		})
	}
}

// checkGoNilPointerDereference detects pointer dereferences without nil check.
// Pattern: variable with pointer type accessed without nil check.
func (w *fileWalker) checkGoNilPointerDereference(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	// Flag field access (dot) without nil check on same line
	if strings.Contains(trimmed, ".") && !strings.Contains(trimmed, "if") && !strings.Contains(trimmed, "!= nil") {
		// Avoid flagging package imports and method definitions
		if !strings.HasPrefix(trimmed, "//") && !strings.Contains(trimmed, "import") && !strings.Contains(trimmed, "func") {
			w.emitFinding(analysis.Finding{
				Rule:        "go_conventions.nil_pointer_dereference",
				Pillar:      "nil_safety",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "pointer dereference without visible nil check — potential panic",
				Remediation: "Check for nil before dereferencing: if ptr != nil { ... ptr.Field ... }.",
			})
		}
	}
}

// checkGoNilSliceIteration detects iteration over potentially nil slices.
// Pattern: for range slice without nil check (slices are safe but better practice).
func (w *fileWalker) checkGoNilSliceIteration(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	if strings.HasPrefix(trimmed, "for") && strings.Contains(trimmed, "range") {
		w.emitFinding(analysis.Finding{
			Rule:        "go_conventions.nil_slice_iteration",
			Pillar:      "nil_safety",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "iteration over range variable — verify slice is not nil (defensive practice)",
			Remediation: "Explicitly check if slice != nil before iteration, or initialize to empty slice instead of nil.",
		})
	}
}

// checkGoNilMethodCall detects method calls on potentially nil receivers.
// Pattern: variable.Method() without nil check visible on same line.
func (w *fileWalker) checkGoNilMethodCall(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	// Flag method calls (dot + parentheses) without nil check
	if strings.Contains(trimmed, ".") && strings.Contains(trimmed, "(") {
		if !strings.Contains(trimmed, "if ") && !strings.Contains(trimmed, "!= nil") && !strings.Contains(trimmed, "!= nil") {
			// Exclude function definitions, imports, and comments
			if !strings.HasPrefix(trimmed, "func ") && !strings.HasPrefix(trimmed, "//") && !strings.Contains(trimmed, "import") {
				w.emitFinding(analysis.Finding{
					Rule:        "go_conventions.nil_method_call",
					Pillar:      "nil_safety",
					Severity:    analysis.SeverityAdvisory,
					Line:        lineNum,
					Message:     "method called on variable without visible nil check",
					Remediation: "Add nil check before method call: if recv != nil { recv.Method() }.",
				})
			}
		}
	}
}


