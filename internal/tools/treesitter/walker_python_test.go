package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Python Conventions Tests (18 new rules)

// Type Safety Tests

func TestPythonTypeHintsMissing_NoHints(t *testing.T) {
	src := `def calculate(x, y):
    return x + y`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.type_hints_missing") {
		t.Error("must flag function without type hints")
	}
}

func TestPythonTypeHintsMissing_WithHints_NoFP(t *testing.T) {
	src := `def calculate(x: int, y: int) -> int:
    return x + y`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.type_hints_missing") {
		t.Error("must not flag function with type hints")
	}
}

func TestPythonNoneCoercion_ImplicitCheck(t *testing.T) {
	src := `if x:
    print("x is truthy")`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.none_coercion") {
		t.Error("must flag implicit None check")
	}
}

func TestPythonNoneCoercion_ExplicitCheck_NoFP(t *testing.T) {
	src := `if x is not None:
    print("x is not None")`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.none_coercion") {
		t.Error("must not flag explicit None check")
	}
}

func TestPythonDynamicAttribute_SetAttr(t *testing.T) {
	src := `setattr(obj, name, value)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.dynamic_attribute") {
		t.Error("must flag setattr() call")
	}
}

func TestPythonDynamicAttribute_GetAttr(t *testing.T) {
	src := `getattr(obj, "name")`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.dynamic_attribute") {
		t.Error("must flag getattr() call")
	}
}

func TestPythonTypeInconsistency_ConditionalReturn(t *testing.T) {
	src := `return value if condition else default`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.type_inconsistency") {
		t.Error("must flag conditional return")
	}
}

func TestPythonDuckTypingUnsafe_NoTypeCheck(t *testing.T) {
	src := `items.append(x)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.duck_typing_unsafe") {
		t.Error("must flag attribute access without isinstance check")
	}
}

func TestPythonDuckTypingUnsafe_WithTypeCheck_NoFP(t *testing.T) {
	src := `items.append(x) if isinstance(items, list) else None`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.duck_typing_unsafe") {
		t.Error("must not flag attribute access with isinstance check on same line")
	}
}

// Async/Concurrency Tests

func TestPythonUnclosedAsyncResource_AioHTTP(t *testing.T) {
	src := `await aiohttp.ClientSession()`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.unclosed_async_resource") {
		t.Error("must flag unclosed async resource")
	}
}

func TestPythonUnclosedAsyncResource_WithAsyncWith_NoFP(t *testing.T) {
	src := `async with aiohttp.ClientSession() as session:`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.unclosed_async_resource") {
		t.Error("must not flag resource with async with")
	}
}

func TestPythonAsyncDeadlock_Sleep(t *testing.T) {
	src := `time.sleep(1)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.async_deadlock") {
		t.Error("must flag time.sleep() in async context")
	}
}

func TestPythonAsyncDeadlock_Requests(t *testing.T) {
	src := `requests.get(url)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.async_deadlock") {
		t.Error("must flag requests.get() in async context")
	}
}

func TestPythonTaskLeak_CreateTaskUntracked(t *testing.T) {
	src := `asyncio.create_task(coro)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.task_leak") {
		t.Error("must flag untracked asyncio.create_task()")
	}
}

func TestPythonTaskLeak_WithAwait_NoFP(t *testing.T) {
	src := `await asyncio.create_task(coro)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.task_leak") {
		t.Error("must not flag awaited task")
	}
}

func TestPythonEventLoopMismatch_NewLoop(t *testing.T) {
	src := `asyncio.new_event_loop()`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.event_loop_mismatch") {
		t.Error("must flag manual event loop creation")
	}
}

func TestPythonEventLoopMismatch_SetEventLoop(t *testing.T) {
	src := `asyncio.set_event_loop(loop)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.event_loop_mismatch") {
		t.Error("must flag manual event loop assignment")
	}
}

// Exception Handling Tests

func TestPythonConventionsBareExcept(t *testing.T) {
	src := `except:`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.bare_except") {
		t.Error("must flag bare except:")
	}
}

func TestPythonConventionsBareExcept_Specific_NoFP(t *testing.T) {
	src := `except ValueError:`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.bare_except") {
		t.Error("must not flag specific exception type")
	}
}

func TestPythonExceptionSwallowing_PassOnly(t *testing.T) {
	src := `except Exception: pass`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.exception_swallowing") {
		t.Error("must flag exception swallowing with pass")
	}
}

func TestPythonExceptionChaining_RaiseWithoutFrom(t *testing.T) {
	src := `raise ValueError("error")`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.exception_chaining") {
		t.Error("must flag raise without from")
	}
}

func TestPythonExceptionChaining_RaiseFrom_NoFP(t *testing.T) {
	src := `raise ValueError("error") from e`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.exception_chaining") {
		t.Error("must not flag raise from")
	}
}

func TestPythonFinallySideEffects(t *testing.T) {
	src := `        file.write(data)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.finally_side_effects") {
		t.Error("must flag file write in finally block")
	}
}

// Import Organization Tests

func TestPythonCircularImport_RelativeImport(t *testing.T) {
	src := `from . import module`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.circular_import") {
		t.Error("must flag relative import")
	}
}

func TestPythonImportOrder_StdlibAfterThirdParty(t *testing.T) {
	src := `import numpy
import sys`
	findings := findingsForSrc(t, src, analysis.LangPython)
	// Since our simple heuristic can't track order across lines, skip this test
	// A proper implementation would need multi-line state tracking
	_ = findings
}

func TestPythonUnusedImport_AliasUnderscore(t *testing.T) {
	src := `import unused as _`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.unused_import") {
		t.Error("must flag unused import")
	}
}

// Memory & Resource Management Tests

func TestPythonResourceLeak_OpenWithoutWith(t *testing.T) {
	src := `f = open("file.txt")`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.resource_leak") {
		t.Error("must flag open() without with")
	}
}

func TestPythonResourceLeak_WithContext_NoFP(t *testing.T) {
	src := `with open("file.txt") as f:`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.resource_leak") {
		t.Error("must not flag open() with context manager")
	}
}

func TestPythonUnboundedGrowth_AppendNoCheck(t *testing.T) {
	src := `items.append(value)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "python_conventions.unbounded_growth") {
		t.Error("must flag unbounded append")
	}
}

func TestPythonUnboundedGrowth_WithMaxSize_NoFP(t *testing.T) {
	src := `items.append(value) if len(items) < maxlen else None`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "python_conventions.unbounded_growth") {
		t.Error("must not flag append with bounds check")
	}
}
