package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func (w *fileWalker) checkPythonPrint(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "print(") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.fmt_print",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "print() call in production code — use structured logging",
			Remediation: "Replace with logger.debug() or appropriate logging.",
		})
	}
}

func (w *fileWalker) checkPythonBareExcept(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "except:") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.no_bare_except",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "bare except: catches all exceptions including SystemExit/KeyboardInterrupt",
			Remediation: "Catch specific exception types: except SpecificError:",
		})
	}
}

func (w *fileWalker) checkPythonEvalExec(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	for _, pat := range []string{"eval(", "exec("} {
		if strings.Contains(trimmed, pat) {
			w.emitFinding(analysis.Finding{
				Rule:        "python.no_eval_exec",
				Pillar:      "python_conventions",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "eval()/exec() of dynamic code — remote code execution risk",
				Remediation: "Use ast.literal_eval() for trusted data or a proper parser.",
			})
			return
		}
	}
}

func (w *fileWalker) checkPythonSQLStringConcat(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	upper := strings.ToUpper(trimmed)
	hasSQL := false
	for _, kw := range []string{"SELECT ", "INSERT ", "UPDATE ", "DELETE "} {
		if strings.Contains(upper, kw) {
			hasSQL = true
			break
		}
	}
	hasSQLFragment := strings.Contains(upper, " WHERE ") || strings.Contains(upper, " AND ") || strings.Contains(upper, " OR ")
	hasSQLVar := strings.Contains(trimmed, "sql +") || strings.Contains(trimmed, "query +") ||
		strings.Contains(trimmed, "+ sql") || strings.Contains(trimmed, "+ query")
	if !hasSQL && !hasSQLFragment && !hasSQLVar {
		return
	}
	if !strings.Contains(trimmed, "+") && !strings.Contains(trimmed, `f"`) && !strings.Contains(trimmed, "f'") {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "python.sql_injection",
		Pillar:      "python_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "possible SQL injection — use parameterized queries, not string formatting",
		Remediation: "Use parameterized queries: cursor.execute('SELECT * FROM t WHERE id = %s', (id,))",
	})
}

func (w *fileWalker) checkPythonSubprocess(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "os.system(") || strings.Contains(trimmed, "os.popen(") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.no_subprocess_shell",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "subprocess with shell=True enables shell injection attacks",
			Remediation: "Remove shell=True and pass args as a list, or use shlex.quote() on inputs.",
		})
		return
	}
	if strings.Contains(trimmed, "subprocess.") && strings.Contains(trimmed, "shell=True") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.no_subprocess_shell",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "subprocess with shell=True enables shell injection attacks",
			Remediation: "Remove shell=True and pass args as a list, or use shlex.quote() on inputs.",
		})
	}
}

func (w *fileWalker) checkPythonMutableDefault(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	for _, pat := range []string{"=[]", "={}", "=set()"} {
		if strings.Contains(trimmed, pat) && strings.HasPrefix(trimmed, "def ") {
			w.emitFinding(analysis.Finding{
				Rule:        "python.no_mutable_default",
				Pillar:      "python_conventions",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "mutable default argument — the same list/dict/set is shared across all calls",
				Remediation: "Use None as default and create the mutable inside the function: if x is None: x = []",
			})
			return
		}
	}
}

func (w *fileWalker) checkPythonWildcardImport(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "import *") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.no_wildcard_import",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "wildcard import (from module import *) pollutes the namespace",
			Remediation: "Import only what you need: from module import SpecificName",
		})
	}
}

// ── Type Safety (5 rules) ─────────────────────────────────────────────────

func (w *fileWalker) checkPythonConventionTypeHintsMissing(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.HasPrefix(trimmed, "def ") && !strings.Contains(trimmed, "->") &&
		!strings.Contains(line, ":") {
		return // false positive: method def with no return hint
	}
	if strings.HasPrefix(trimmed, "def ") {
		hasParams := strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")")
		params := trimmed
		if idx := strings.Index(params, "("); idx >= 0 {
			params = params[idx : strings.Index(params, ")")+1]
		}
		if hasParams && !strings.Contains(params, ":") {
			w.emitFinding(analysis.Finding{
				Rule:        "python_conventions.type_hints_missing",
				Pillar:      "python_conventions",
				Severity:    analysis.SeverityAdvisory,
				Line:        lineNum,
				Message:     "Function definition lacks type hints for parameters or return type",
				Remediation: "Add type hints: def fn(x: int, y: str) -> bool:",
			})
		}
	}
}

func (w *fileWalker) checkPythonConventionNoneCoercion(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "if ") && !strings.Contains(trimmed, "is not None") &&
		!strings.Contains(trimmed, "is None") && (strings.Contains(trimmed, "if x:") ||
		strings.Contains(trimmed, "if obj:") || strings.Contains(trimmed, "if val:")) {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.none_coercion",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Implicit None check via falsy comparison — use explicit 'is not None'",
			Remediation: "Use: if x is not None: instead of if x:",
		})
	}
}

func (w *fileWalker) checkPythonConventionDynamicAttribute(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if (strings.Contains(trimmed, "setattr(") || strings.Contains(trimmed, "getattr(")) &&
		!strings.Contains(trimmed, "default=") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.dynamic_attribute",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Dynamic attribute access via setattr/getattr — prefer explicit attributes",
			Remediation: "Define class attributes explicitly or use __slots__",
		})
	}
}

func (w *fileWalker) checkPythonConventionTypeInconsistency(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.HasPrefix(trimmed, "return ") && !strings.Contains(trimmed, ":") {
		// This is a very basic check; full analysis requires AST
		if strings.Contains(trimmed, "if ") || strings.Contains(trimmed, "else") {
			w.emitFinding(analysis.Finding{
				Rule:        "python_conventions.type_inconsistency",
				Pillar:      "python_conventions",
				Severity:    analysis.SeverityAdvisory,
				Line:        lineNum,
				Message:     "Return statement may have inconsistent type — verify it matches the declared return type",
				Remediation: "Ensure all return paths return the same type as declared in the function signature",
			})
		}
	}
}

func (w *fileWalker) checkPythonConventionDuckTypingUnsafe(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	// Only flag if we actually have a method call on a variable
	hasMethodCall := strings.Contains(trimmed, ".append(") || strings.Contains(trimmed, ".read(") ||
		strings.Contains(trimmed, ".write(")
	if !hasMethodCall {
		return
	}

	// Check if there's an isinstance check on this line
	if strings.Contains(trimmed, "isinstance(") {
		return // isinstance check present, no flag
	}

	// Only flag if this looks like a variable access (not in a definition)
	if strings.HasPrefix(trimmed, "if ") || strings.HasPrefix(trimmed, "def ") {
		return
	}

	w.emitFinding(analysis.Finding{
		Rule:        "python_conventions.duck_typing_unsafe",
		Pillar:      "python_conventions",
		Severity:    analysis.SeverityAdvisory,
		Line:        lineNum,
		Message:     "Assuming attribute existence without isinstance() check — could fail at runtime",
		Remediation: "Add isinstance() check before calling methods on assumed types",
	})
}

// ── Async/Concurrency (4 rules) ───────────────────────────────────────────

func (w *fileWalker) checkPythonConventionUnclosedAsyncResource(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "await ") && !strings.Contains(trimmed, "async with") &&
		(strings.Contains(trimmed, "aiohttp.") || strings.Contains(trimmed, "asyncpg.") ||
			strings.Contains(trimmed, "asyncio.open") || strings.Contains(trimmed, "aiomysql.")) {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.unclosed_async_resource",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "Async resource (aiohttp/asyncpg/etc) not used with 'async with' — resource may leak",
			Remediation: "Use: async with resource as r: ... to ensure cleanup",
		})
	}
}

func (w *fileWalker) checkPythonConventionAsyncDeadlock(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "async def ") {
		return // header line
	}
	if strings.Contains(trimmed, "time.sleep(") || strings.Contains(trimmed, "requests.get(") ||
		strings.Contains(trimmed, "open(") && !strings.Contains(trimmed, "aio") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.async_deadlock",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Blocking call (sleep/requests/open) in async context — blocks the event loop",
			Remediation: "Use async alternatives: asyncio.sleep(), aiohttp.get(), aiofiles.open()",
		})
	}
}

func (w *fileWalker) checkPythonConventionTaskLeak(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "asyncio.create_task(") && !strings.Contains(trimmed, "await") &&
		!strings.Contains(trimmed, "gather(") && !strings.Contains(trimmed, ".add_done_callback(") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.task_leak",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "asyncio.create_task() called without tracking or callback — task may be abandoned",
			Remediation: "Track tasks: tasks.append(t) or use asyncio.gather() or add_done_callback()",
		})
	}
}

func (w *fileWalker) checkPythonConventionEventLoopMismatch(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "asyncio.new_event_loop") ||
		strings.Contains(trimmed, "asyncio.set_event_loop") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.event_loop_mismatch",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Creating/setting event loop manually — may conflict with running loop",
			Remediation: "Use asyncio.run() or the current running loop: asyncio.get_running_loop()",
		})
	}
}

// ── Exception Handling (4 rules) ──────────────────────────────────────────

func (w *fileWalker) checkPythonConventionBareExcept(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "except:") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.bare_except",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "bare except: catches all exceptions including SystemExit and KeyboardInterrupt",
			Remediation: "Catch specific exception types: except SpecificException:",
		})
	}
}

func (w *fileWalker) checkPythonConventionExceptionSwallowing(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.HasPrefix(trimmed, "except") && (strings.Contains(trimmed, ":") ||
		strings.Contains(trimmed, "pass")) {
		// This is very basic; a full check needs to track the except block
		if strings.HasPrefix(trimmed, "except") && strings.Contains(trimmed, ":") &&
			strings.Contains(trimmed, "pass") {
			w.emitFinding(analysis.Finding{
				Rule:        "python_conventions.exception_swallowing",
				Pillar:      "python_conventions",
				Severity:    analysis.SeverityMajor,
				Line:        lineNum,
				Message:     "Exception handler with only pass — silently ignores errors without logging",
				Remediation: "Log the exception or re-raise: except E: logger.error(...) or raise",
			})
		}
	}
}

func (w *fileWalker) checkPythonConventionExceptionChaining(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	// Skip if has proper exception chaining
	if strings.Contains(trimmed, " from ") {
		return
	}
	// Flag if raising an exception without "from" clause
	if strings.HasPrefix(trimmed, "raise ") && (strings.Contains(trimmed, "Error(") ||
		strings.Contains(trimmed, "Exception(")) {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.exception_chaining",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Raising exception without context — use 'raise NewEx() from original' to preserve traceback",
			Remediation: "Use: raise NewException(...) from e to chain exceptions",
		})
	}
}

func (w *fileWalker) checkPythonConventionFinallySideEffects(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.HasPrefix(trimmed, "finally:") {
		return // header, not the body
	}
	if (strings.Contains(trimmed, "open(") || strings.Contains(trimmed, ".write(") ||
		strings.Contains(trimmed, ".send(") || strings.Contains(trimmed, ".delete(")) &&
		// Very basic heuristic: if 'finally' is in parent context
		!strings.Contains(trimmed, "try:") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.finally_side_effects",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "I/O or mutation in finally block — may cause cascading failures during error handling",
			Remediation: "Move side effects to separate functions; keep finally for resource cleanup only",
		})
	}
}

// ── Import Organization (3 rules) ─────────────────────────────────────────

func (w *fileWalker) checkPythonConventionCircularImport(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if (strings.HasPrefix(trimmed, "from ") || strings.HasPrefix(trimmed, "import ")) &&
		strings.Contains(trimmed, "from . import ") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.circular_import",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Relative import in module — may create circular dependency",
			Remediation: "Restructure to break cycles; move imports inside functions if needed",
		})
	}
}

func (w *fileWalker) checkPythonConventionImportOrder(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if !strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, "from ") {
		return
	}
	// Very basic check: imports should come before code (not after a def/class)
	// This is overly simplistic but catches some cases
	isStdlib := strings.Contains(trimmed, "import sys") || strings.Contains(trimmed, "import os") ||
		strings.Contains(trimmed, "import json")
	isThirdParty := strings.Contains(trimmed, "import requests") || strings.Contains(trimmed, "import numpy") ||
		strings.Contains(trimmed, "import django")

	if isStdlib && isThirdParty {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.import_order",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Import order incorrect — PEP 8: stdlib first, then third-party, then local",
			Remediation: "Reorganize imports: stdlib → third-party → local (with blank lines between groups)",
		})
	}
}

func (w *fileWalker) checkPythonConventionUnusedImport(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if !strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, "from ") {
		return
	}
	// This is an extremely basic heuristic: just flag imports that look suspicious
	// A proper check requires whole-file analysis
	if strings.Contains(trimmed, "import ") && strings.Contains(trimmed, " as ") &&
		strings.Contains(trimmed, "_") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.unused_import",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "Import may be unused (imported but never referenced)",
			Remediation: "Remove unused imports or use them in the module",
		})
	}
}

// ── Memory & Resource Management (2 rules) ──────────────────────────────

func (w *fileWalker) checkPythonConventionResourceLeak(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "open(") && !strings.Contains(trimmed, "with ") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.resource_leak",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "File or resource opened without context manager — will not be closed",
			Remediation: "Use: with open(...) as f: ... to ensure resource cleanup",
		})
	}
	if (strings.Contains(trimmed, "requests.") || strings.Contains(trimmed, "socket.") ||
		strings.Contains(trimmed, "sqlite3.connect(")) && !strings.Contains(trimmed, "with ") &&
		!strings.Contains(trimmed, ".close()") {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.resource_leak",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "Resource (connection/socket) opened without context manager — may leak",
			Remediation: "Use context managers: with resource: ... or ensure explicit .close()",
		})
	}
}

func (w *fileWalker) checkPythonConventionUnboundedGrowth(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	// Flag unbounded collection growth
	hasAppend := strings.Contains(trimmed, ".append(")
	hasBounds := strings.Contains(trimmed, "maxlen") || strings.Contains(trimmed, "lru_cache") ||
		strings.Contains(trimmed, "limit")

	if hasAppend && !hasBounds {
		w.emitFinding(analysis.Finding{
			Rule:        "python_conventions.unbounded_growth",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "Data structure (list/dict) appended to without bounds — may grow unbounded in long-running services",
			Remediation: "Add size bounds: use collections.deque(maxlen=...) or LRU caches, or implement cleanup logic",
		})
	}
}
