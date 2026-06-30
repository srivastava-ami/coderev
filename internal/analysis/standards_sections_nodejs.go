package analysis

// ── Node.js Conventions ───────────────────────────────────────────────────────
// Enterprise-grade production Node.js rules for async patterns and performance.
// Phase 1 (7 of 14 rules): Async Patterns (4) + Performance (3)

type NodeJSConventionsStd struct {
	Severity              string                      `toml:"severity"`
	AsyncPatterns         NodeJSAsyncPatternsStd      `toml:"async_patterns"`
	Performance           NodeJSPerformanceStd        `toml:"performance"`
}

// ── Async Patterns (4 rules) ──────────────────────────────────────────────────

type NodeJSAsyncPatternsStd struct {
	CallbackHell                  CallbackHellStd                `toml:"callback_hell"`
	PromiseSwallowing             PromiseSwallowingStd           `toml:"promise_swallowing"`
	AsyncIteratorIncomplete       AsyncIteratorIncompleteStd     `toml:"async_iterator_incomplete"`
	ConcurrentOperationsUnbounded ConcurrentOpsUnboundedStd      `toml:"concurrent_operations_unbounded"`
}

// CallbackHellStd detects deeply nested callback chains (>3 levels of .then())
// that reduce readability and make error handling unclear.
type CallbackHellStd struct {
	Rule        string `toml:"rule"`
	MaxDepth    int    `toml:"max_depth"`
	Remediation string `toml:"remediation"`
}

// PromiseSwallowingStd detects promises without error handling (.catch or try/await).
// Unhandled promise rejections cause uncaught exceptions and application crashes.
type PromiseSwallowingStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// AsyncIteratorIncompleteStd detects async iterators missing the return() method
// required for proper cleanup and iteration termination.
type AsyncIteratorIncompleteStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// ConcurrentOpsUnboundedStd detects unbounded concurrent operations (e.g., Promise.all
// without size limits) that can cause memory or CPU exhaustion.
type ConcurrentOpsUnboundedStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// ── Performance (3 rules) ─────────────────────────────────────────────────────

type NodeJSPerformanceStd struct {
	MemoryLeakTimers MemoryLeakTimersStd `toml:"memory_leak_timers"`
	UnboundedBuffer  UnboundedBufferStd  `toml:"unbounded_buffer"`
	CPUBlocking      CPUBlockingStd      `toml:"cpu_blocking"`
}

// MemoryLeakTimersStd detects setInterval without corresponding clearInterval,
// causing timers to persist indefinitely and leak memory.
type MemoryLeakTimersStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// UnboundedBufferStd detects unbounded internal buffering (array/queue growth
// without size limits or eviction) causing memory leaks under sustained load.
type UnboundedBufferStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// CPUBlockingStd detects synchronous operations on the event loop (e.g., fs.readFileSync,
// crypto.scryptSync) that block async I/O and degrade responsiveness.
type CPUBlockingStd struct {
	Rule        string   `toml:"rule"`
	BlockingOps []string `toml:"blocking_ops"`
	Remediation string   `toml:"remediation"`
}
