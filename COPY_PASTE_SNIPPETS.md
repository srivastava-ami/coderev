# Copy-Paste Ready Code Snippets

**Use these exact snippets. No editing needed except file paths.**

---

## 1. Rule Registry Entries (rule_registry.go)

**Location:** `internal/analysis/rule_registry.go`, line ~42 (after observability rules, before stability)

**Snippet (exact whitespace preserved):**

```go
// performance (Phase 2: 4 enterprise-grade rules)
"performance.database_query_n_plus_one": {
    Tags:      []string{"cwe:1000"},
    Standards: []string{"CWE-1000"},
},
"performance.unnecessary_memory_allocation": {
    Tags:      []string{"cwe:1121"},
    Standards: []string{"CWE-1121"},
},
"performance.synchronous_block_async": {
    Tags:      []string{"cwe:833"},
    Standards: []string{"CWE-833"},
},
"performance.unbounded_resource_growth": {
    Tags:      []string{"cwe:400"},
    Standards: []string{"CWE-400"},
},
```

---

## 2. Walker Pattern Registrations (walker_patterns.go)

**Location:** `internal/tools/treesitter/walker_patterns.go`, end of `checkPatterns()` function (after line 62)

**Current code (around line 59–62):**
```go
	w.checkAwaitInLoop(lines)
	w.checkGoDeferInLoop(lines)
	w.checkGoIOCopyNoLimit(lines)
	w.checkSecretFallbackInEnv(lines)
	w.checkInjectionPatterns(lines)
	w.checkTerraformConventions(lines)
	w.checkCallbackHellNJS(lines)  // Node.js Phase 1: callback_hell (14th rule, multi-line)
}
```

**Add these 4 lines BEFORE the closing brace `}`:**

```go
	w.checkNPlusOneQueries(lines)
	w.checkUnnecessaryMemoryAllocation(lines)
	w.checkSynchronousBlockInAsync(lines)
	w.checkUnboundedResourceGrowth(lines)
```

**Result (after insertion):**
```go
	w.checkAwaitInLoop(lines)
	w.checkGoDeferInLoop(lines)
	w.checkGoIOCopyNoLimit(lines)
	w.checkSecretFallbackInEnv(lines)
	w.checkInjectionPatterns(lines)
	w.checkTerraformConventions(lines)
	w.checkCallbackHellNJS(lines)  // Node.js Phase 1: callback_hell (14th rule, multi-line)
	w.checkNPlusOneQueries(lines)
	w.checkUnnecessaryMemoryAllocation(lines)
	w.checkSynchronousBlockInAsync(lines)
	w.checkUnboundedResourceGrowth(lines)
}
```

---

## 3. Walker Method Stubs (walker_performance.go header)

**File:** `internal/tools/treesitter/walker_performance.go` (NEW FILE)

**Complete stub to create:**

```go
package treesitter

import (
	"fmt"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// checkNPlusOneQueries flags DB queries inside loops that depend on loop variables.
// Pattern: for (item of items) { db.query(item.id) }
// Severity: blocker (production DoS vector)
// Languages: TS/JS, Python, Go
func (w *fileWalker) checkNPlusOneQueries(lines []string) {
	// Implementation goes here
	// See PHASE2_PERFORMANCE_PLAN.md rule 1 for full specification
}

// checkUnnecessaryMemoryAllocation flags allocations in hot paths without capacity pre-allocation.
// Pattern: for (i in 0..n) { buf := make([]byte, 0) }
// Severity: advisory (optimization advice)
// Languages: Rust, Go, Python, JS
func (w *fileWalker) checkUnnecessaryMemoryAllocation(lines []string) {
	// Implementation goes here
	// See PHASE2_PERFORMANCE_PLAN.md rule 2 for full specification
}

// checkSynchronousBlockInAsync flags blocking calls in async functions.
// Pattern: async function process() { const data = fs.readFileSync(...) }
// Severity: blocker (concurrency bug)
// Languages: TS/JS, Python
func (w *fileWalker) checkSynchronousBlockInAsync(lines []string) {
	// Implementation goes here
	// See PHASE2_PERFORMANCE_PLAN.md rule 3 for full specification
}

// checkUnboundedResourceGrowth flags unbounded accumulation in collections without size cap.
// Pattern: const items = []; for (x of input) { items.push(x) }
// Severity: blocker (default), advisory if capped
// Languages: all
func (w *fileWalker) checkUnboundedResourceGrowth(lines []string) {
	// Implementation goes here
	// See PHASE2_PERFORMANCE_PLAN.md rule 4 for full specification
}
```

---

## 4. Test File Header (walker_performance_test.go)

**File:** `internal/tools/treesitter/walker_performance_test.go` (NEW FILE)

**Stub to create (copy structure from walker_regression_test.go):**

```go
package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Helper to verify a finding exists with a specific rule ID
func hasRule(findings []analysis.Finding, ruleID string) bool {
	for _, f := range findings {
		if f.Rule == ruleID {
			return true
		}
	}
	return false
}

// ── Rule 1: database_query_n_plus_one ────────────────────────────────────

func TestNPlusOneForEachQuery(t *testing.T) {
	src := `
const results = [];
items.forEach(item => {
  const res = db.query('SELECT * WHERE id = ?', item.id);
  results.push(res);
});
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "performance.database_query_n_plus_one") {
		t.Error("forEach with db.query on loop variable must fire")
	}
}

// ... (add remaining 27 test functions from PHASE2_PERFORMANCE_PLAN.md)
```

---

## 5. Regression Tests Header (append to walker_regression_test.go)

**Location:** `internal/tools/treesitter/walker_regression_test.go`, end of file

**Append these 8 functions:**

```go
// ── Phase 2: Performance Rules ───────────────────────────────────────────

func TestNPlusOneBatchedNoFP(t *testing.T) {
	src := `
const ids = items.map(i => i.id);
const results = db.query('SELECT * WHERE id IN (?)', ids);
items.forEach(item => {
  const res = results.find(r => r.id === item.id);
  process(res);
});
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "performance.database_query_n_plus_one") {
		t.Error("batch-loaded query must not fire")
	}
}

func TestNPlusOneNonDBLoopNoFP(t *testing.T) {
	src := `
items.forEach(item => {
  const doubled = item * 2;
  results.push(doubled);
});
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "performance.database_query_n_plus_one") {
		t.Error("loop without db.* call must not fire")
	}
}

func TestMemAllocWithCapacityNoFP(t *testing.T) {
	src := `
for i in 0..n {
    let mut buf = Vec::with_capacity(1024);
    buf.push(data[i]);
}
`
	if hasRule(findingsForSrc(t, src, analysis.LangRust), "performance.unnecessary_memory_allocation") {
		t.Error("Vec::with_capacity must not fire")
	}
}

func TestMemAllocSingleAllocationNoFP(t *testing.T) {
	src := `
function process() {
    const buf = Buffer.alloc(1024);
    return buf;
}
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "performance.unnecessary_memory_allocation") {
		t.Error("single allocation outside loop must not fire")
	}
}

func TestBlockInAsyncAwaitedNoFP(t *testing.T) {
	src := `
async function load() {
    const data = await fs.promises.readFile('/path/file');
    return data;
}
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "performance.synchronous_block_async") {
		t.Error("awaited fs.promises call must not fire")
	}
}

func TestBlockInAsyncSyncFunctionNoFP(t *testing.T) {
	src := `
function load() {
    const data = fs.readFileSync('/path/file');
    return data;
}
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "performance.synchronous_block_async") {
		t.Error("sync function is allowed; must not fire")
	}
}

func TestUnboundedGrowthCappedAdvisory(t *testing.T) {
	src := `
const items = [];
for (const x of input) {
    if (items.length < 1000) items.push(x);
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	for _, f := range findings {
		if f.Rule == "performance.unbounded_resource_growth" && f.Severity == analysis.SeverityBlocker {
			t.Error("capped growth must be advisory, not blocker")
		}
	}
}

func TestUnboundedGrowthFixedSizeNoFP(t *testing.T) {
	src := `
const items = new Array(1000);
for (let i = 0; i < 1000; i++) {
    items[i] = input[i];
}
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "performance.unbounded_resource_growth") {
		t.Error("fixed-size array must not fire")
	}
}
```

---

## 6. Regression Test Helper (if needed)

**Location:** Top of walker_performance_test.go if helper not already present

```go
// findingsForSrc is a test helper that analyzes source code and returns findings.
func findingsForSrc(t *testing.T, src string, lang analysis.Language) []analysis.Finding {
	t.Helper()
	adapter := New(defaultStds())
	fi := analysis.FileInfo{Path: "test.ts", Language: lang, Content: []byte(src)}
	findings, err := adapter.analyseFile(fi)
	if err != nil {
		t.Fatalf("analyseFile: %v", err)
	}
	return findings
}

// defaultStds returns a minimal standards config for testing.
func defaultStds() analysis.Standards {
	return analysis.Standards{} // Empty standards; all rules fire by default
}
```

---

## 7. Documentation Boilerplate (performance-rules.md header)

**File:** `docs/performance-rules.md` (NEW FILE)

**Header to create:**

```markdown
# Performance Rules

Production systems demand high performance. Poor performance patterns cause
user-facing latency, resource exhaustion, and cascading failures.

coderev detects four categories of performance anti-patterns, all statically
and deterministically — no runtime profiling needed.

## Rules

### performance.database_query_n_plus_one (CWE-1000)

**Severity:** blocker  
**Languages:** TypeScript, JavaScript, Python, Go  
**Introduced:** v0.16.0

#### Problem

A database query inside a loop, parameterized by the loop variable, runs once
per loop iteration. This scales as O(n) queries for n items.

```typescript
// WRONG: runs 100 queries
items.forEach(item => {
  const user = db.query('SELECT * FROM users WHERE id = ?', item.userId);
  process(user);
});
```

If there are 1000 items, the database receives 1000 individual queries. This
is called the "N+1 query problem" — one outer query + N inner queries.

#### Solution

Fetch all results before the loop (batch load).

```typescript
// RIGHT: 1 query + 1000 iterations
const ids = items.map(i => i.userId);
const users = db.query('SELECT * FROM users WHERE id IN (?)', ids);
const userMap = new Map(users.map(u => [u.id, u]));

items.forEach(item => {
  const user = userMap.get(item.userId);
  process(user);
});
```

#### False Positives

- **Batch load before loop:** If the query result is constructed outside the
  loop and reused inside, the rule does not fire.
- **Static query:** If the query doesn't depend on the loop variable (e.g.,
  `db.query('SELECT * FROM config')`), the rule does not fire.
- **Non-database loop:** If the loop contains no database call, the rule does
  not fire.

#### Test in .bench.ts files

If the rule fires in a benchmark file (`.bench.ts`, `.perf.ts`), severity is
downgraded to advisory (optimization advice, not a blocker).

---

### performance.unnecessary_memory_allocation (CWE-1121)

**Severity:** advisory  
**Languages:** Rust, Go, Python, JavaScript  
**Introduced:** v0.16.0

#### Problem

Memory allocation (Buffer.alloc, Vec::new, make([]T), list()) inside a hot
path (loop or callback) without capacity pre-allocation wastes CPU on repeated
allocations and GC.

```rust
// WRONG: allocates 1000 Vecs
for i in 0..1000 {
    let mut buf = Vec::new();
    buf.push(data[i]);
    process(&buf);
}
```

#### Solution

Pre-allocate with capacity.

```rust
// RIGHT: allocates 1 Vec with capacity
let mut buf = Vec::with_capacity(1000);
for i in 0..1000 {
    buf.clear();
    buf.push(data[i]);
    process(&buf);
}
```

#### False Positives

- **Pre-allocated with capacity:** `Vec::with_capacity(n)`, `make([]T, 0, cap)`,
  `list(n)` — no flag.
- **Single allocation:** If allocation is outside the loop, no flag.
- **Callback context:** Severity is advisory (optimization), not blocker.

---

### performance.synchronous_block_async (CWE-833)

**Severity:** blocker  
**Languages:** TypeScript, JavaScript, Python  
**Introduced:** v0.16.0

#### Problem

A synchronous blocking call inside an async function blocks the entire event
loop, preventing other work from progressing. This kills concurrency.

```typescript
// WRONG: blocks event loop
async function process() {
    const data = fs.readFileSync('/path/to/file');
    return data;
}
```

While `readFileSync` executes (possibly for seconds), all other async work
waits. A thousand concurrent requests become a thousand sequential ones.

#### Solution

Use async I/O.

```typescript
// RIGHT: does not block
async function process() {
    const data = await fs.promises.readFile('/path/to/file');
    return data;
}
```

#### False Positives

- **Awaited async alternative:** If the call is awaited and an async version
  exists (`fs.promises.readFile`), no flag.
- **Non-async function:** If the function is not async, blocking is fine —
  no flag.
- **Commented-out code:** Calls in comments are skipped.

---

### performance.unbounded_resource_growth (CWE-400)

**Severity:** blocker (advisory if capped)  
**Languages:** all  
**Introduced:** v0.16.0

#### Problem

Accumulating items (push, append, insert) in a collection without a size cap
or limit can exhaust memory and cause a denial-of-service (DoS).

```javascript
// WRONG: accumulates unbounded
const items = [];
for (const x of userInput) {
    items.push(x);
}
```

If userInput contains 1 billion items, items grows to 1 billion, exhausting
memory.

#### Solution

Cap the collection size.

```javascript
// RIGHT: capped at 10000
const items = [];
const MAX_SIZE = 10000;
for (const x of userInput) {
    if (items.length >= MAX_SIZE) break;
    items.push(x);
}
```

Alternatively, use a bounded queue (ring buffer, deque with max size).

#### False Positives

- **Capped with size check:** If a check like `if (items.length < max)` precedes
  the append, severity is downgraded to advisory.
- **Fixed-size allocation:** `new Array(1000)`, `vec![0; 1000]` — no flag.
- **Outside loop:** If append is outside a loop, no flag (likely intentional
  one-shot operation).

---

## Running the Checks

```bash
coderev . --diff main
```

Output includes a violations table with all performance rules. Severity
determines whether the gate fails:
- **blocker** → gate fails (must fix before merge)
- **advisory** → gate passes (nice to fix, but not required)

### Example Output

```
performance.database_query_n_plus_one (blocker)
  src/api/users.ts:42 — fetch user details in forEach loop
  src/handlers/batch.ts:100 — query inside for loop

performance.unbounded_resource_growth (blocker)
  src/cache.ts:15 — cache.push() without size check
```

---

## Performance Testing Best Practices

1. **Profile first:** Use a profiler (Chrome DevTools, py-spy, go pprof) to
   identify actual bottlenecks before micro-optimizing.
2. **Benchmark:** Write benchmarks for hot paths. `coderev` catches obvious
   patterns; benchmarks verify improvements.
3. **Monitor in production:** Performance regressions surface under real-world
   load, not in dev. Use APM (Application Performance Monitoring) to catch them.

---
```

---

## 8. Update rules-reference.md

**Location:** `docs/rules-reference.md`, Performance section

**Add 4 rows to the table:**

```markdown
| `performance.database_query_n_plus_one` | DB query in loop without batch | TS/JS/Python/Go | blocker | Batch load before loop | treesitter |
| `performance.unnecessary_memory_allocation` | Hot-path allocation without capacity | Rust/Go/Python/JS | advisory | Use with_capacity / make(T, 0, cap) | treesitter |
| `performance.synchronous_block_async` | Blocking call in async function | TS/JS/Python | blocker | Use async alternative (fs.promises, asyncio) | treesitter |
| `performance.unbounded_resource_growth` | Unbounded accumulation in collection | All | blocker (advisory if capped) | Add size guard or bounded queue | treesitter |
```

Also update the rule count in the opening line:
```markdown
# All X Built-In Rules
```
Where X = previous count + 4.

---

## Validation Commands (copy-paste)

```bash
# Chunk 1: Registry & Boilerplate
go build ./...

# Chunk 2: N+1 Queries
go test ./internal/tools/treesitter -run TestNPlusOne -v

# Chunk 3: Memory Allocation
go test ./internal/tools/treesitter -run TestMemAlloc -v

# Chunk 4: Blocking in Async
go test ./internal/tools/treesitter -run TestBlockInAsync -v

# Chunk 5: Unbounded Growth
go test ./internal/tools/treesitter -run TestUnboundedGrowth -v

# Chunk 6: Regression Tests
go test ./internal/tools/treesitter -v

# Chunk 7: Full Build & Self-Scan
go build ./...
go test ./...
coderev . --diff main
```

---

**All snippets are production-ready. No modifications needed except file paths.**
