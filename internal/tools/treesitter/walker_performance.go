package treesitter

import (
	"regexp"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Performance anti-pattern detectors covering N+1 queries, memory allocation loops,
// async blocking, and unbounded resource growth across Go, Python, and Node.js.

// ── Rule 1: Database Query N+1 Patterns ──────────────────────────────────────────

var reNPlusOneGo = regexp.MustCompile(
	`(?i)(for\s+\w+\s*:=|for\s+\w+\s*,\s*\w+\s*:=)\s*range.*[\n\r][\s\S]*?` +
		`(db\.(Query|Exec|QueryRow|Prepare)|sqlc|bun|gorm).*(?:user|item|row|record|item|id)`,
)

func (w *fileWalker) checkDatabaseNPlusOne(lines []string) {
	if isTestFile(w.file) {
		return
	}
	// Check for loop-level database queries: a loop with a query inside
	for i, line := range lines {
		if !strings.Contains(line, "for ") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		// Look ahead for a query in the next 5 lines
		found := false
		for j := i + 1; j < len(lines) && j < i+5; j++ {
			nextLine := strings.TrimSpace(lines[j])
			if strings.HasPrefix(nextLine, "//") {
				continue
			}
			if w.hasQueryPattern(nextLine) {
				found = true
				break
			}
		}
		if found {
			w.emitFinding(analysis.Finding{
				Rule:        "performance.database_query_n_plus_one",
				Pillar:      "performance",
				Severity:    analysis.SeverityMajor,
				Line:        i + 1,
				Message:     "Database query inside loop — N+1 query pattern (one query per iteration)",
				Remediation: "Batch queries or use JOINs to fetch all records at once, reducing database round trips",
			})
		}
	}
}

func (w *fileWalker) hasQueryPattern(line string) bool {
	queryPatterns := []string{
		"db.Query", "db.Exec", "db.QueryRow", "db.Prepare",
		"session.query", "db.execute", "conn.execute",
		".query(", ".execute(", ".run(", ".get(", ".all(",
		"sqlc.", "bun.", "gorm.", "knex.", "sequelize.",
	}
	upper := strings.ToUpper(line)
	for _, pat := range queryPatterns {
		if strings.Contains(upper, strings.ToUpper(pat)) {
			return true
		}
	}
	return false
}

// ── Rule 2: Unnecessary Memory Allocation in Loops ────────────────────────────

func (w *fileWalker) checkUnnecessaryMemoryAllocation(lines []string) {
	if isTestFile(w.file) {
		return
	}
	for i, line := range lines {
		if !strings.Contains(line, "for ") && !strings.Contains(line, "while ") {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Look ahead for memory allocation in the next 3 lines
		for j := i + 1; j < len(lines) && j < i+3; j++ {
			nextLine := strings.TrimSpace(lines[j])
			if strings.HasPrefix(nextLine, "//") || strings.HasPrefix(nextLine, "#") {
				continue
			}
			if w.hasAllocationPattern(nextLine) {
				w.emitFinding(analysis.Finding{
					Rule:        "performance.unnecessary_memory_allocation",
					Pillar:      "performance",
					Severity:    analysis.SeverityMajor,
					Line:        j + 1,
					Message:     "Memory allocation inside loop — allocates new object on every iteration",
					Remediation: "Move allocation outside the loop: pre-allocate the collection or reuse a buffer",
				})
				break
			}
		}
	}
}

func (w *fileWalker) hasAllocationPattern(line string) bool {
	allocPatterns := []string{
		"make(", "append(", "new(", "= []", "= {}", "= []{}",
		"new Array", "new Object", "new Buffer",
		"list(", "dict(", "set(", "tuple(",
	}
	for _, pat := range allocPatterns {
		if strings.Contains(line, pat) {
			return true
		}
	}
	return false
}

// ── Rule 3: Synchronous Blocking in Async Context ────────────────────────────

func (w *fileWalker) checkSynchronousBlockAsync(lines []string) {
	if w.lang != analysis.LangJavaScript && w.lang != analysis.LangTypeScript &&
		w.lang != analysis.LangPython {
		return
	}
	if isTestFile(w.file) {
		return
	}
	var inAsync bool
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Detect async function entry
		if strings.Contains(line, "async ") && (strings.Contains(line, "function") ||
			strings.Contains(line, "=>") || strings.Contains(line, "def")) {
			inAsync = true
			continue
		}
		// Detect async function exit (closing brace at same indentation as opening)
		if inAsync && strings.TrimSpace(line) == "}" {
			inAsync = false
		}
		if inAsync && w.hasBlockingPattern(trimmed) {
			w.emitFinding(analysis.Finding{
				Rule:        "performance.synchronous_block_async",
				Pillar:      "performance",
				Severity:    analysis.SeverityMajor,
				Line:        i + 1,
				Message:     "Synchronous (blocking) operation in async function — blocks event loop",
				Remediation: "Replace with async equivalent: asyncio.sleep(), setImmediate(), or async I/O methods",
			})
		}
	}
}

func (w *fileWalker) hasBlockingPattern(line string) bool {
	blockingPatterns := []string{
		"time.sleep", ".sleep(",
		"setTimeout", "setInterval",
		"readFileSync", "writeFileSync",
		"readSync", "writeSync",
		"execSync", "spawnSync",
	}
	for _, pat := range blockingPatterns {
		if strings.Contains(line, pat) {
			return true
		}
	}
	return false
}

// ── Rule 4: Unbounded Resource Growth ────────────────────────────────────────

func (w *fileWalker) checkUnboundedResourceGrowth(lines []string) {
	if isTestFile(w.file) {
		return
	}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !w.hasGrowthPattern(trimmed) {
			continue
		}
		// Check if there's a size check or cleanup
		if w.hasSizeOrCleanupGuard(lines, i) {
			continue
		}
		w.emitFinding(analysis.Finding{
			Rule:        "performance.unbounded_resource_growth",
			Pillar:      "performance",
			Severity:    analysis.SeverityMajor,
			Line:        i + 1,
			Message:     "Collection grows without bounds — no max size, eviction, or cleanup",
			Remediation: "Add bounded size limits (e.g., max_size=1000), LRU eviction, or periodic cleanup",
		})
	}
}

func (w *fileWalker) hasGrowthPattern(line string) bool {
	growthPatterns := []string{
		"cache[", "queue[", "buffer[", "pool[", "map[",
		".push(", ".append(", ".add(", ".put(",
		"dict[", "list[", "set.add",
	}
	for _, pat := range growthPatterns {
		if strings.Contains(line, pat) {
			return true
		}
	}
	return false
}

func (w *fileWalker) hasSizeOrCleanupGuard(lines []string, idx int) bool {
	// Look ahead and behind for size/limit checks
	start := idx - 2
	if start < 0 {
		start = 0
	}
	end := idx + 3
	if end > len(lines) {
		end = len(lines)
	}
	lookAhead := strings.Join(lines[start:end], " ")
	guardPatterns := []string{
		"len(", "size(", "maxsize", "max_size", "limit",
		"evict", "clear", "reset", "cleanup", "prune",
		">= ", "== ", "< ", "limited",
	}
	for _, pat := range guardPatterns {
		if strings.Contains(lookAhead, pat) {
			return true
		}
	}
	return false
}
