package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Performance anti-pattern tests covering all 4 rules across multiple languages.

// ── N+1 Query Tests ────────────────────────────────────────────────────────────

func TestNPlusOneQueryGoLoop(t *testing.T) {
	src := `
for _, user := range users {
  row := db.QueryRow("SELECT * FROM posts WHERE user_id = $1", user.ID)
}
`
	f := findingsForSrc(t, src, analysis.LangGo)
	if !hasRule(f, "performance.database_query_n_plus_one") {
		t.Error("must flag N+1 query in Go loop")
	}
}

func TestNPlusOneQueryGoNoViolation(t *testing.T) {
	src := `
posts := db.QueryAll("SELECT * FROM posts WHERE user_id IN ($1)", userIDs)
for _, post := range posts {
  fmt.Println(post)
}
`
	f := findingsForSrc(t, src, analysis.LangGo)
	if hasRule(f, "performance.database_query_n_plus_one") {
		t.Error("must NOT flag batched query")
	}
}

func TestNPlusOneQueryNodeJSLoop(t *testing.T) {
	src := `
for (const user of users) {
  const results = db.query("SELECT * FROM posts WHERE user_id = ?", user.id);
}
`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "performance.database_query_n_plus_one") {
		t.Error("must flag N+1 query in Node.js loop")
	}
}

func TestNPlusOneQueryPythonLoop(t *testing.T) {
	src := `
for user in users:
    result = session.query(Post).filter(Post.user_id == user.id).all()
`
	f := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(f, "performance.database_query_n_plus_one") {
		t.Error("must flag N+1 query in Python loop")
	}
}

// ── Memory Allocation in Loops Tests ───────────────────────────────────────────

func TestMemoryAllocationLoopGo(t *testing.T) {
	src := `
for i := 0; i < 1000000; i++ {
  x := make([]int, 100)
}
`
	f := findingsForSrc(t, src, analysis.LangGo)
	if !hasRule(f, "performance.unnecessary_memory_allocation") {
		t.Error("must flag memory allocation in Go loop")
	}
}

func TestMemoryAllocationPreallocated(t *testing.T) {
	src := `
buffer := make([]int, 100000)
for i := 0; i < 1000; i++ {
  buffer[i] = i * 2
}
`
	f := findingsForSrc(t, src, analysis.LangGo)
	if hasRule(f, "performance.unnecessary_memory_allocation") {
		t.Error("must NOT flag pre-allocated buffer")
	}
}

func TestMemoryAllocationLoopNodeJS(t *testing.T) {
	src := `
for (let i = 0; i < 1000000; i++) {
  const arr = new Array(100);
  arr[0] = i;
}
`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "performance.unnecessary_memory_allocation") {
		t.Error("must flag memory allocation in Node.js loop")
	}
}

func TestMemoryAllocationLoopPython(t *testing.T) {
	src := `
for i in range(1000000):
    x = [0] * 100
    x[0] = i
`
	f := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(f, "performance.unnecessary_memory_allocation") {
		t.Error("must flag memory allocation in Python loop")
	}
}

// ── Async Blocking Tests ───────────────────────────────────────────────────────

func TestAsyncBlockingNodeJS(t *testing.T) {
	src := `
async function fetch() {
  const fs = require('fs');
  const data = fs.readFileSync('file.txt');
  await process(data);
}
`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "performance.synchronous_block_async") {
		t.Error("must flag sync operation in async function")
	}
}

func TestAsyncBlockingPython(t *testing.T) {
	src := `
async def fetch():
    import time
    time.sleep(1)
    data = await load_data()
`
	f := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(f, "performance.synchronous_block_async") {
		t.Error("must flag blocking sleep in async function")
	}
}

func TestAsyncNonBlockingNodeJS(t *testing.T) {
	src := `
async function fetch() {
  await new Promise(resolve => setTimeout(resolve, 1000));
  const data = await loadData();
}
`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "performance.synchronous_block_async") {
		t.Error("must NOT flag async sleep")
	}
}

// ── Unbounded Resource Growth Tests ────────────────────────────────────────────

func TestUnboundedCacheGrowth(t *testing.T) {
	src := `
const cache = {};
for (const item of items) {
  cache[item.id] = item;
}
`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(f, "performance.unbounded_resource_growth") {
		t.Error("must flag unbounded cache growth")
	}
}

func TestBoundedCacheGrowth(t *testing.T) {
	src := `
const cache = {};
const MAX_SIZE = 1000;
for (const item of items) {
  if (Object.keys(cache).length >= MAX_SIZE) {
    delete cache[Object.keys(cache)[0]];
  }
  cache[item.id] = item;
}
`
	f := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(f, "performance.unbounded_resource_growth") {
		t.Error("must NOT flag bounded cache with size limit")
	}
}

func TestUnboundedQueueGrowth(t *testing.T) {
	src := `
queue = []
while True:
    item = get_item()
    queue.append(item)
`
	f := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(f, "performance.unbounded_resource_growth") {
		t.Error("must flag unbounded queue growth")
	}
}

func TestBoundedQueueGrowth(t *testing.T) {
	src := `
from collections import deque
queue = deque(maxlen=100)
for item in items:
    queue.append(item)
`
	f := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(f, "performance.unbounded_resource_growth") {
		t.Error("must NOT flag bounded deque with maxlen")
	}
}

// ── Test Files Should Not Trigger Rules ──────────────────────────────────────

func TestPerformanceRulesSkipTestFile(t *testing.T) {
	src := `
for (let i = 0; i < 1000; i++) {
  const arr = new Array(100);
}
`
	f := findingsForPath(t, src, "performance_test.js", analysis.LangJavaScript)
	if hasRule(f, "performance.unnecessary_memory_allocation") {
		t.Error("must NOT flag performance issues in test files")
	}
}
