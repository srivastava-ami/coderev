package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Node.js convention checkers covering streams, event emitters, async patterns, and performance.

// ── Streams (4 rules) ──────────────────────────────────────────────────────────

// checkStreamNotPiped detects streams created but not piped or consumed.
func (w *fileWalker) checkStreamNotPiped(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	if codeLineSkip(line) {
		return
	}
	// Detect: fs.createReadStream(...) or stream.Readable.from(...) without .pipe or .on('data')
	streamPatterns := []string{
		"fs.createReadStream", "fs.createWriteStream",
		"stream.Readable", "stream.Writable", "stream.Transform", "stream.PassThrough",
	}
	hasStreamCreate := false
	for _, pat := range streamPatterns {
		if strings.Contains(line, pat) {
			hasStreamCreate = true
			break
		}
	}
	if !hasStreamCreate {
		return
	}
	// Check if followed by .pipe, .on('data'), or other consumption pattern
	consumptionPatterns := []string{
		".pipe(", ".on('data", ".on(\"data", ".on('chunk", ".on(\"chunk",
		".on('readable", ".on(\"readable", "stream.Readable.from(",
	}
	for _, pat := range consumptionPatterns {
		if strings.Contains(line, pat) {
			return
		}
	}
	// If assigned to a variable, it might be consumed later
	if strings.Contains(line, "const ") || strings.Contains(line, "let ") || strings.Contains(line, "var ") {
		w.emitFinding(analysis.Finding{
			Rule:        "nodejs_conventions.stream_not_piped",
			Pillar:      "nodejs_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Stream created but not piped or consumed — memory leak if not handled",
			Remediation: "Pipe the stream (.pipe()) or attach data handlers (.on('data', ...))",
		})
	}
}

// checkBackpressureIgnored detects stream.write() calls without checking return value.
func (w *fileWalker) checkBackpressureIgnored(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	if codeLineSkip(line) {
		return
	}
	if !strings.Contains(line, ".write(") {
		return
	}
	// If the line has a return value check, it's handling backpressure
	checkPatterns := []string{
		"const ", "let ", "var ", "if (", "&&", "||",
		".on('drain", ".on(\"drain", ".pause()", ".resume()",
	}
	for _, pat := range checkPatterns {
		if strings.Contains(line, pat) {
			return
		}
	}
	// Standalone write() call without return value check
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.backpressure_ignored",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "stream.write() return value not checked — ignoring backpressure causes memory buildup",
		Remediation: "Check if write() returns false and handle the 'drain' event before writing more",
	})
}

// checkStreamErrorUnhandled detects streams without error handlers.
func (w *fileWalker) checkStreamErrorUnhandled(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	if codeLineSkip(line) {
		return
	}
	// Detect stream creation
	streamPatterns := []string{
		"fs.createReadStream", "fs.createWriteStream",
		"stream.Readable", "stream.Writable", "stream.Transform",
	}
	hasStreamCreate := false
	for _, pat := range streamPatterns {
		if strings.Contains(line, pat) {
			hasStreamCreate = true
			break
		}
	}
	if !hasStreamCreate {
		return
	}
	// Check if error handler is already attached
	errorHandlers := []string{
		".on('error", `.on("error`, ".once('error", `.once("error`,
		".addListener('error", `.addListener("error`,
	}
	for _, handler := range errorHandlers {
		if strings.Contains(line, handler) {
			return
		}
	}
	// Stream without error handler
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.stream_error_unhandled",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "Stream created without error handler — unhandled errors crash the application",
		Remediation: "Attach .on('error', (err) => {...}) immediately after stream creation",
	})
}

// checkStreamLeak detects error handlers without stream.destroy() cleanup.
func (w *fileWalker) checkStreamLeak(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	// Look for error handlers without cleanup
	if !strings.Contains(line, ".on('error") && !strings.Contains(line, `.on("error`) {
		return
	}
	// Check if the error handler contains cleanup
	cleanupPatterns := []string{
		".destroy()", ".close()", "finally", "defer",
	}
	for _, pat := range cleanupPatterns {
		if strings.Contains(line, pat) {
			return
		}
	}
	// Error handler without cleanup
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.stream_leak",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "Error handler present but stream not destroyed — resource leak",
		Remediation: "Call stream.destroy() in error handlers; use try/finally for guaranteed cleanup",
	})
}

// ── Event Emitters (3 rules) ───────────────────────────────────────────────────

// checkEventListenerLeak detects listeners added but never removed.
func (w *fileWalker) checkEventListenerLeak(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	if codeLineSkip(line) {
		return
	}
	// Detect: emitter.on(...) without .removeListener or .off
	if !strings.Contains(line, ".on(") {
		return
	}
	// If has .off or .removeListener, no leak
	if strings.Contains(line, ".off(") || strings.Contains(line, ".removeListener(") {
		return
	}
	// Long-lived listener without removal
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.event_listener_leak",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     "Event listener added but never removed — accumulated listeners cause memory leak",
		Remediation: "Store listener references and call .removeListener() or .off() during cleanup or object destruction",
	})
}

// checkOnceVsOn detects using .on() for single-event listeners.
func (w *fileWalker) checkOnceVsOn(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	// Single-event keywords that should use .once()
	singleEventKeywords := []string{
		"'ready'", `"ready"`,
		"'connected'", `"connected"`,
		"'close'", `"close"`,
		"'exit'", `"exit"`,
		"'end'", `"end"`,
		"'started'", `"started"`,
	}
	hasKeyword := false
	for _, kw := range singleEventKeywords {
		if strings.Contains(line, ".on("+kw) {
			hasKeyword = true
			break
		}
	}
	if !hasKeyword {
		return
	}
	// If already using .once, no issue
	if strings.Contains(line, ".once(") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.once_vs_on",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityAdvisory,
		Line:        lineNum,
		Message:     "Using .on() for single-event listener — should use .once() for efficiency",
		Remediation: "Replace .on() with .once() when the handler should only fire once",
	})
}

// checkErrorEventUnhandled detects EventEmitters without error handler.
func (w *fileWalker) checkErrorEventUnhandled(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	// EventEmitter patterns
	emitterPatterns := []string{
		"EventEmitter", "http.createServer", "https.createServer",
		"net.createServer", "dgram.createSocket",
	}
	hasEmitter := false
	for _, pat := range emitterPatterns {
		if strings.Contains(line, pat) {
			hasEmitter = true
			break
		}
	}
	if !hasEmitter {
		return
	}
	// Check for error handler
	if strings.Contains(line, ".on('error") || strings.Contains(line, `.on("error`) {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.error_event_unhandled",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "EventEmitter created without 'error' handler — uncaught errors crash the application",
		Remediation: "Attach .on('error', (err) => {...}) immediately after EventEmitter creation",
	})
}

// ── Async Patterns (4 rules) ───────────────────────────────────────────────────
// Note: checkCallbackHell is implemented in walker_javascript.go and applies here.

// checkPromiseSwallowing detects promises without error handling.
func (w *fileWalker) checkPromiseSwallowing(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	if codeLineSkip(line) {
		return
	}
	// Detect promise-returning function calls
	promisePatterns := []string{
		"fetch(",
		".then(",
		"new Promise(",
		"Promise.resolve(",
		"Promise.reject(",
	}
	hasPromise := false
	for _, pat := range promisePatterns {
		if strings.Contains(line, pat) {
			hasPromise = true
			break
		}
	}
	if !hasPromise {
		return
	}
	// If .catch() or await is present, it's handled
	if strings.Contains(line, ".catch(") || strings.Contains(line, "await ") || strings.Contains(line, "try ") {
		return
	}
	// If it's a .then() definition itself, don't flag (catch comes after)
	if strings.Contains(line, ".then(") && !strings.HasSuffix(strings.TrimSpace(line), ";") {
		return
	}
	// Fire-and-forget promise
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.promise_swallowing",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "Promise without error handling — unhandled rejections crash the application",
		Remediation: "Add .catch() or use try/catch with await; never fire-and-forget promises",
	})
}

// checkAsyncIteratorIncomplete detects async iterators missing return() method.
func (w *fileWalker) checkAsyncIteratorIncomplete(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	// Detect Symbol.asyncIterator
	if !strings.Contains(line, "Symbol.asyncIterator") {
		return
	}
	// Check if return() method is implemented
	if strings.Contains(line, "return()") || strings.Contains(line, "return:") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.async_iterator_incomplete",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     "Async iterator missing return() method — cleanup code won't run",
		Remediation: "Implement return() alongside next() to handle early termination cleanup",
	})
}

// checkConcurrentOperationsUnbounded detects unbounded concurrent operations.
func (w *fileWalker) checkConcurrentOperationsUnbounded(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	// Detect unbounded Promise.all or Promise.allSettled
	if !strings.Contains(line, "Promise.all") && !strings.Contains(line, "Promise.allSettled") {
		return
	}
	// If there's a size check or limit, it's okay
	if strings.Contains(line, ".slice(") || strings.Contains(line, "chunk") || strings.Contains(line, "limit") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.concurrent_operations_unbounded",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     "Unbounded concurrent operations — memory/CPU exhaustion risk",
		Remediation: "Cap concurrency with pLimit, p-queue, batch processing, or manual size checks",
	})
}

// ── Performance (3 rules) ──────────────────────────────────────────────────────

// checkMemoryLeakTimers detects setInterval without clearInterval.
func (w *fileWalker) checkMemoryLeakTimers(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	if !strings.Contains(line, "setInterval(") {
		return
	}
	// If stored in a variable or has clearInterval, it's handled
	if strings.Contains(line, "const ") || strings.Contains(line, "let ") || strings.Contains(line, "var ") {
		return
	}
	if strings.Contains(line, "clearInterval") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.memory_leak_timers",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     "setInterval without clearInterval — timer persists indefinitely",
		Remediation: "Store timer ID: const id = setInterval(...); then call clearInterval(id) in cleanup",
	})
}

// checkUnboundedBuffer detects unbounded buffer growth.
func (w *fileWalker) checkUnboundedBuffer(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	if !strings.Contains(line, ".push(") {
		return
	}
	// If there's a length check or size limit, it's okay
	if strings.Contains(line, ".length") || strings.Contains(line, "maxSize") || strings.Contains(line, "if (") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.unbounded_buffer",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     "Unbounded buffer growth without size limits — memory leak",
		Remediation: "Add size checks and eviction: if (queue.length > maxSize) queue.shift()",
	})
}

// checkCpuBlocking detects synchronous operations blocking the event loop.
func (w *fileWalker) checkCpuBlocking(line string, lineNum int) {
	if !w.isNodeJS() {
		return
	}
	// Synchronous (blocking) operations
	blockingOps := []string{
		"fs.readFileSync(", "fs.writeFileSync(",
		"child_process.spawnSync(", "crypto.scryptSync(",
	}
	hasBlocking := false
	for _, op := range blockingOps {
		if strings.Contains(line, op) {
			hasBlocking = true
			break
		}
	}
	if !hasBlocking {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "nodejs_conventions.cpu_blocking",
		Pillar:      "nodejs_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     "Synchronous operation blocks event loop — degrades responsiveness",
		Remediation: "Use async variants (fs.readFile) or move to Worker threads with worker_threads",
	})
}

// Helper function to check if file is JavaScript/TypeScript
func (w *fileWalker) isNodeJS() bool {
	return w.lang == analysis.LangJavaScript || w.lang == analysis.LangTypeScript
}

// checkCallbackHellNJS detects >3 levels of .then() nesting (multi-line check, Node.js version)
func (w *fileWalker) checkCallbackHellNJS(lines []string) {
	if !w.isNodeJS() {
		return
	}
	// Track consecutive .then() chains to detect nesting depth
	for i, line := range lines {
		depth := strings.Count(line, ".then(")
		// Count only nested cases: look for pattern like .then(...).then(...).then(...).then(
		if depth >= 4 {
			w.emitFinding(analysis.Finding{
				Rule:        "nodejs_conventions.callback_hell",
				Pillar:      "nodejs_conventions",
				Severity:    analysis.SeverityMajor,
				Line:        i + 1,
				Message:     "Promise chain with 4+ levels of .then() nesting — difficult to read and maintain",
				Remediation: "Use async/await syntax instead: async () => { const r1 = await p1(); const r2 = await p2(r1); ... }",
			})
			return
		}
	}
}
