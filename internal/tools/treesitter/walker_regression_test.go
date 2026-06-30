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

func TestSecretFallbackJWTBlocker(t *testing.T) {
	src := `const secret = process.env.JWT_SECRET ?? 'local-dev-secret-change-me';`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "security.secret_fallback_literal") {
		t.Fatal("must flag JWT_SECRET ?? literal")
	}
	for _, finding := range f {
		if finding.Rule == "security.secret_fallback_literal" && finding.Severity != analysis.SeverityBlocker {
			t.Errorf("expected blocker, got %s", finding.Severity)
		}
	}
}

func TestSecretFallbackAPIKeyOr(t *testing.T) {
	src := `const s = process.env.API_KEY || "changeme";`
	if !hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must flag API_KEY || literal")
	}
}

func TestSecretFallbackBracketNotation(t *testing.T) {
	src := `process.env["DB_PASSWORD"] ?? "postgres";`
	if !hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must flag bracket notation DB_PASSWORD")
	}
}

func TestSecretFallbackEmptyNoFP(t *testing.T) {
	src := `const s = process.env.JWT_SECRET ?? '';`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag empty string fallback")
	}
}

func TestSecretFallbackPortNoFP(t *testing.T) {
	src := `const p = process.env.PORT ?? '3000';`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag PORT — not a secret key name")
	}
}

func TestSecretFallbackLogLevelNoFP(t *testing.T) {
	src := `const l = process.env.LOG_LEVEL || 'info';`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag LOG_LEVEL — not a secret key name")
	}
}

func TestSecretFallbackTestFileNoFP(t *testing.T) {
	src := `const secret = process.env.JWT_SECRET ?? 'test-secret';`
	if hasRule(findingsForPath(t, src, "auth.test.ts", analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag in test files")
	}
}

func TestSecretFallbackNodeEnvGuardAdvisory(t *testing.T) {
	src := `
if (process.env.NODE_ENV !== 'production') {
  secret = process.env.JWT_SECRET ?? 'dev-secret';
}
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "security.secret_fallback_literal") {
		t.Error("must still fire inside NODE_ENV guard (advisory, not suppressed)")
	}
	for _, finding := range f {
		if finding.Rule == "security.secret_fallback_literal" && finding.Severity == analysis.SeverityBlocker {
			t.Error("must be advisory inside NODE_ENV guard, not blocker")
		}
	}
}

// ── Go Convention Rules (20 new rules for production Go) ──────────────────────

// TestGoGoroutineLeakDetected flags bare "go " statement.
func TestGoGoroutineLeakDetected(t *testing.T) {
	src := `
func Worker(tasks chan Task) {
	go processTask(tasks)
}
`
	findings := findingsForPath(t, src, "worker.go", analysis.LangGo)
	if !hasRule(findings, "go_conventions.goroutine_leak") {
		t.Error("must flag goroutine spawned without tracking")
	}
}

// TestGoDeadlockChannelReceive detects bare channel receive without timeout.
func TestGoDeadlockChannelReceive(t *testing.T) {
	src := `
func WaitForSignal(ch chan struct{}) {
	<-ch
}
`
	findings := findingsForPath(t, src, "signal.go", analysis.LangGo)
	if !hasRule(findings, "go_conventions.deadlock_pattern") {
		t.Error("must flag channel receive without timeout")
	}
}

// TestGoDeferPanicDetected flags defer containing panic.
func TestGoDeferPanicDetected(t *testing.T) {
	src := `
func Cleanup() {
	defer panic("cleanup failed")
}
`
	findings := findingsForPath(t, src, "cleanup.go", analysis.LangGo)
	if !hasRule(findings, "go_conventions.defer_panic") {
		t.Error("must flag defer with panic")
	}
}

// TestGoUncheckedErrorDetected flags explicit error ignore.
func TestGoUncheckedErrorDetected(t *testing.T) {
	src := `
func WriteFile(filename string, data []byte) {
	_ = os.WriteFile(filename, data, 0o644)
}
`
	findings := findingsForPath(t, src, "file.go", analysis.LangGo)
	if !hasRule(findings, "go_conventions.unchecked_error") {
		t.Error("must flag explicit error ignore with _ =")
	}
}

// TestGoInterfaceBloatDetected flags interface type declaration.
func TestGoInterfaceBloatDetected(t *testing.T) {
	src := `
type Reader interface {
	Read(p []byte) (n int, err error)
	ReadByte() (byte, error)
	ReadRune() (rune, int, error)
	ReadFull(p []byte) (n int, err error)
}
`
	findings := findingsForPath(t, src, "reader.go", analysis.LangGo)
	if !hasRule(findings, "go_conventions.interface_bloat") {
		t.Error("must flag interface declaration (triggers manual review)")
	}
}

// TestGoUnclosedBodyDetected flags response body access without close.
func TestGoUnclosedBodyDetected(t *testing.T) {
	src := `
func FetchData(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	return body, err
}
`
	findings := findingsForPath(t, src, "fetch.go", analysis.LangGo)
	if !hasRule(findings, "go_conventions.unclosed_body") {
		t.Error("must flag resp.Body access without close")
	}
}

// TestGoFileDescriptorLeakDetected flags os.Open without close.
func TestGoFileDescriptorLeakDetected(t *testing.T) {
	src := `
func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}
`
	findings := findingsForPath(t, src, "file.go", analysis.LangGo)
	if !hasRule(findings, "go_conventions.file_descriptor_leak") {
		t.Error("must flag os.Open without visible close")
	}
}

// TestGoNilSliceIterationDetected flags bare range without nil check.
func TestGoNilSliceIterationDetected(t *testing.T) {
	src := `
func ProcessSlice(items []Item) {
	for _, item := range items {
		item.Process()
	}
}
`
	findings := findingsForPath(t, src, "process.go", analysis.LangGo)
	if !hasRule(findings, "go_conventions.nil_slice_iteration") {
		t.Error("must flag range iteration (triggers nil check review)")
	}
}

// TestGoPanicInLibFires verifies existing panic rule still works.
func TestGoPanicInLibFires(t *testing.T) {
	src := `
func ValidateInput(val int) string {
	if val < 0 {
		panic("invalid value")
	}
	return fmt.Sprintf("%d", val)
}
`
	findings := findingsForPath(t, src, "validate.go", analysis.LangGo)
	if !hasRule(findings, "go.panic_in_lib") {
		t.Error("must flag panic in library code")
	}
}

// TestGoContextTODOFires verifies existing context.TODO rule still works.
func TestGoContextTODOFires(t *testing.T) {
	src := `
func QueryDatabase(query string) ([]Row, error) {
	return db.Query(context.TODO(), query)
}
`
	findings := findingsForPath(t, src, "db.go", analysis.LangGo)
	if !hasRule(findings, "go.context_todo") {
		t.Error("must flag context.TODO() placeholder")
	}
}

// TestGoDeferInLoopFires verifies existing defer-in-loop rule still works.
func TestGoDeferInLoopFires(t *testing.T) {
	src := `
func ProcessFiles(files []string) error {
	for _, f := range files {
		handle, err := os.Open(f)
		if err != nil {
			return err
		}
		defer handle.Close()
		process(handle)
	}
	return nil
}
`
	findings := findingsForPath(t, src, "files.go", analysis.LangGo)
	if !hasRule(findings, "go.defer_in_loop") {
		t.Error("must flag defer inside loop")
	}
}
