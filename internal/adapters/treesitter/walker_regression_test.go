package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// findingsForPath is like findingsForSrc but lets the caller specify the file path,
// which matters for isTestFile checks.
func findingsForPath(t *testing.T, src string, path string, lang analysis.Language) []analysis.Finding {
	t.Helper()
	adapter := New(defaultStds())
	fi := analysis.FileInfo{Path: path, Language: lang, Content: []byte(src)}
	findings, err := adapter.analyseFile(fi)
	if err != nil {
		t.Fatalf("analyseFile %s: %v", path, err)
	}
	return findings
}

// ── Bug 1: floating-promise false positives ────────────────────────────────────

// clearSessionToken() is a bare call with no explicit async marker — must NOT fire.
func TestFloatingPromiseBareCallNoFP(t *testing.T) {
	src := `
async function logout() {
  clearSessionToken();
  signOut();
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "stability.no_floating_promise") {
		t.Error("bare () call should not fire stability.no_floating_promise (no Async/async/fetch marker)")
	}
}

// clearSessionTokenAsync() has "Async" in name — must fire.
func TestFloatingPromiseAsyncNamedCallFires(t *testing.T) {
	src := `
async function logout() {
  clearSessionTokenAsync();
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "stability.no_floating_promise") {
		t.Error("call with Async in name should fire stability.no_floating_promise")
	}
}

// Awaited async call must NOT fire.
func TestFloatingPromiseAwaitedNoFP(t *testing.T) {
	src := `
async function logout() {
  await clearSessionTokenAsync();
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "stability.no_floating_promise") {
		t.Error("awaited call must not fire stability.no_floating_promise")
	}
}

// ── Bug 2: boolean-param-flag false positives ─────────────────────────────────

// hash: string — "has" prefix but followed by 'h' (lowercase), not a flag param.
func TestBoolFlagFPHashParam(t *testing.T) {
	src := `
function computeHash(hash: string): string {
  return hash;
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if hasRule(findings, "complexity.boolean_param_flag") {
		t.Error("'hash: string' should not fire complexity.boolean_param_flag (type is string, not boolean)")
	}
}

// hasToken: boolean — legitimate flag param, must fire.
func TestBoolFlagLegitimateParam(t *testing.T) {
	src := `
function render(hasToken: boolean): string {
  return hasToken ? "yes" : "no";
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "complexity.boolean_param_flag") {
		t.Error("'hasToken: boolean' should fire complexity.boolean_param_flag")
	}
}

// ── Bug 2: complexity checks are advisory in test files ───────────────────────

func TestComplexityAdvisoryInTestFile(t *testing.T) {
	// A long function in a .spec.ts file should produce advisory, not blocker.
	src := `
function describe_suite() {
  const a1 = 1; const a2 = 2; const a3 = 3;
  const b1 = 1; const b2 = 2; const b3 = 3;
  const c1 = 1; const c2 = 2; const c3 = 3;
  const d1 = 1; const d2 = 2; const d3 = 3;
  const e1 = 1; const e2 = 2; const e3 = 3;
  const f1 = 1; const f2 = 2; const f3 = 3;
  const g1 = 1; const g2 = 2; const g3 = 3;
  const h1 = 1; const h2 = 2; const h3 = 3;
  const i1 = 1; const i2 = 2; const i3 = 3;
  const j1 = 1; const j2 = 2; const j3 = 3;
  const k1 = 1; const k2 = 2; const k3 = 3;
}
`
	findings := findingsForPath(t, src, "component.spec.ts", analysis.LangTypeScript)
	for _, f := range findings {
		if f.Rule == "complexity.function_length" && f.Severity == analysis.SeverityBlocker {
			t.Errorf("complexity.function_length in .spec.ts must be advisory, got blocker (line %d)", f.Line)
		}
	}
}

// ── Bug 3: magic numbers inside string literals ───────────────────────────────

func TestMagicNumbersInsideStringNoFP(t *testing.T) {
	files := []analysis.FileInfo{
		{
			Path:    "infra.ts",
			Content: []byte(`const instanceId = 'ps-tapjoy-portal-rg-00';`),
		},
		{
			Path:    "infra2.ts",
			Content: []byte(`const pad = '00' + c;`),
		},
		{
			Path:    "id.ts",
			Content: []byte(`const id: string = "ref-42-abc";`),
		},
	}
	findings := checkMagicNumbers(files, nil)
	for _, f := range findings {
		if f.Rule == "hardcoding.magic_numbers" {
			t.Errorf("string literal digits should not fire magic_numbers: %s (line %d: %s)", f.File, f.Line, f.Message)
		}
	}
}

// A real magic number outside a string must still fire.
func TestMagicNumbersOutsideStringFires(t *testing.T) {
	files := []analysis.FileInfo{
		{
			Path:    "timeout.ts",
			Content: []byte(`setTimeout(callback, 5000);`),
		},
	}
	findings := checkMagicNumbers(files, nil)
	if !hasRule(findings, "hardcoding.magic_numbers") {
		t.Error("bare numeric literal 5000 should fire hardcoding.magic_numbers")
	}
}

// ── Bug 4: isTestFile recognises .spec.tsx, .test.tsx, /e2e/ ─────────────────

func TestIsTestFileSpecTsx(t *testing.T) {
	if !isTestFile("src/components/Button.spec.tsx") {
		t.Error(".spec.tsx should be recognised as a test file")
	}
}

func TestIsTestFileTestTsx(t *testing.T) {
	if !isTestFile("src/components/Button.test.tsx") {
		t.Error(".test.tsx should be recognised as a test file")
	}
}

func TestIsTestFileE2EDir(t *testing.T) {
	if !isTestFile("e2e/auth/login.ts") {
		t.Error("file in /e2e/ directory should be recognised as a test file")
	}
}

func TestIsTestFileProductionNotFlagged(t *testing.T) {
	if isTestFile("src/components/Button.tsx") {
		t.Error("production .tsx file must not be recognised as a test file")
	}
}

// no_any in .spec.tsx must be advisory, not blocker.
func TestAnyTypeAdvisoryInSpecTsx(t *testing.T) {
	src := `
const mock = jest.fn() as any;
`
	findings := findingsForPath(t, src, "Button.spec.tsx", analysis.LangTypeScript)
	for _, f := range findings {
		if f.Rule == "type_safety.no_any" && f.Severity == analysis.SeverityBlocker {
			t.Error("type_safety.no_any in .spec.tsx should be advisory, not blocker")
		}
	}
}
