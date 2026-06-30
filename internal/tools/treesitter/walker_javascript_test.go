package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// JavaScript/TypeScript Convention Tests

// Type Safety Tests

func TestJSAnyTypeUsage_TypeScript(t *testing.T) {
	src := `const x: any = getValue();`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "javascript_conventions.any_type_usage") {
		t.Error("must flag bare 'any' type usage")
	}
}

func TestJSAnyTypeUsage_JavaScript_NoFP(t *testing.T) {
	src := `const x = getValue();`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(findings, "javascript_conventions.any_type_usage") {
		t.Error("must not flag non-TS code")
	}
}

func TestJSTypeCoercion_LooseEquality(t *testing.T) {
	src := `if (x == y) { }`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(findings, "javascript_conventions.type_coercion") {
		t.Error("must flag loose equality (==)")
	}
}

func TestJSTypeCoercion_StrictEquality_NoFP(t *testing.T) {
	src := `if (x === y) { }`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(findings, "javascript_conventions.type_coercion") {
		t.Error("must not flag strict equality (===)")
	}
}

func TestJSOptionalChainingOveruse(t *testing.T) {
	src := `const name = "John"?.length;`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "javascript_conventions.optional_chaining_overuse") {
		t.Error("must flag optional chaining on literal")
	}
}

func TestJSNullCoalescingWithCall(t *testing.T) {
	src := `const data = missing ?? JSON.parse(source);`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(findings, "javascript_conventions.null_coalescing_correct") {
		t.Error("must flag null coalescing with error-prone function")
	}
}

func TestJSTypeAssertionUnsafe(t *testing.T) {
	src := `const x = obj as any;`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "javascript_conventions.type_assertion_unsafe") {
		t.Error("must flag unsafe type assertion 'as any'")
	}
}

func TestJSTypeAssertionUnsafe_DoubleAs(t *testing.T) {
	src := `const x = obj as unknown as MyType;`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "javascript_conventions.type_assertion_unsafe") {
		t.Error("must flag double type assertion 'as unknown as'")
	}
}

// Promises & Async Tests

func TestJSUnhandledPromise(t *testing.T) {
	src := `.then(() => { console.log('done'); })`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(findings, "javascript_conventions.unhandled_promise") {
		t.Error("must flag .then() without .catch()")
	}
}

func TestJSUnhandledPromise_WithCatch_NoFP(t *testing.T) {
	src := `.then(() => {}).catch(e => console.error(e))`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(findings, "javascript_conventions.unhandled_promise") {
		t.Error("must not flag .then() with .catch()")
	}
}

func TestJSAsyncAwaitChaining(t *testing.T) {
	src := `const data = await promise.then(x => x);`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(findings, "javascript_conventions.async_await_chain") {
		t.Error("must flag mixing await with .then()")
	}
}

func TestJSPromiseRaceHazard(t *testing.T) {
	src := `const winner = await Promise.race([p1, p2]);`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(findings, "javascript_conventions.promise_race_hazard") {
		t.Error("must flag Promise.race() usage")
	}
}

func TestJSCallbackHell(t *testing.T) {
	src := `.then(a => a).then(b => b).then(c => c).then(d => d)`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(findings, "javascript_conventions.callback_hell") {
		t.Error("must flag deeply nested .then() calls")
	}
}

// Security & Data Flow Tests

func TestJSNullCoalescingCorrect_SafeCall_NoFP(t *testing.T) {
	src := `const data = missing ?? fallback;`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(findings, "javascript_conventions.null_coalescing_correct") {
		t.Error("must not flag null coalescing with simple fallback")
	}
}

func TestJSTypeAssertion_Simple_NoFP(t *testing.T) {
	src := `const x = obj as MyType;`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "javascript_conventions.type_assertion_unsafe") {
		t.Error("must not flag legitimate type assertions")
	}
}

// Comment lines should not trigger

func TestJSConventions_CommentSkip(t *testing.T) {
	src := `// This code has any but it's in a comment`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "javascript_conventions.any_type_usage") {
		t.Error("must skip comment lines")
	}
}

func TestJSConventions_LooseEqualitySkipInComment(t *testing.T) {
	src := `// if (x == y) { }`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if hasRule(findings, "javascript_conventions.type_coercion") {
		t.Error("must skip comment lines for equality check")
	}
}
