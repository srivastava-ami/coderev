package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// These tests cover the native port of the destructuring-default form of the
// owned semgrep rule security.secret_fallback_literal (walker_injection.go).
// Each case is a positive (must flag) or negative (must not flag) fixture.

// ── positive ───────────────────────────────────────────────────────────────

func TestInjectionDestructureSecretBlocker(t *testing.T) {
	src := `const { SIGNING_KEY = 'dev-key' } = process.env;`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "security.secret_fallback_literal") {
		t.Fatal("must flag destructuring default { SIGNING_KEY = 'dev-key' }")
	}
	for _, finding := range f {
		if finding.Rule == "security.secret_fallback_literal" &&
			finding.Severity != analysis.SeverityBlocker {
			t.Errorf("expected blocker, got %s", finding.Severity)
		}
	}
}

func TestInjectionDestructureMultiKey(t *testing.T) {
	// JWT_SECRET must flag; PORT (same statement) must not.
	src := `const { JWT_SECRET = 'x', PORT = '3000' } = process.env;`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if countRule(f, "security.secret_fallback_literal") != 1 {
		t.Fatalf("expected exactly 1 finding (JWT_SECRET only), got %d", countRule(f, "security.secret_fallback_literal"))
	}
}

func TestInjectionDestructureMultiline(t *testing.T) {
	src := `
const {
  API_KEY = 'changeme',
  PORT    = '3000',
} = process.env;
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "security.secret_fallback_literal") {
		t.Error("must flag multi-line destructuring default for API_KEY")
	}
	for _, finding := range f {
		if finding.Rule == "security.secret_fallback_literal" && finding.Line != 3 {
			t.Errorf("expected finding on line 3 (API_KEY), got line %d", finding.Line)
		}
	}
}

func TestInjectionDestructureRenamed(t *testing.T) {
	// Renamed destructuring: env var name is JWT_SECRET, local is secret.
	src := `const { JWT_SECRET: secret = 'dev' } = process.env;`
	if !hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must flag renamed destructuring { JWT_SECRET: secret = 'dev' }")
	}
}

func TestInjectionDestructureNodeEnvGuardAdvisory(t *testing.T) {
	src := `
if (process.env.NODE_ENV !== 'production') {
  const { JWT_SECRET = 'dev-secret' } = process.env;
}
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "security.secret_fallback_literal") {
		t.Fatal("must still fire inside NODE_ENV guard (advisory, not suppressed)")
	}
	for _, finding := range f {
		if finding.Rule == "security.secret_fallback_literal" &&
			finding.Severity == analysis.SeverityBlocker {
			t.Error("must be advisory inside NODE_ENV guard, not blocker")
		}
	}
}

// ── negative ───────────────────────────────────────────────────────────────

func TestInjectionDestructurePortNoFP(t *testing.T) {
	src := `const { PORT = '3000' } = process.env;`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag PORT — not a secret key name")
	}
}

func TestInjectionDestructureEmptyNoFP(t *testing.T) {
	src := `const { JWT_SECRET = '' } = process.env;`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag empty-string default")
	}
}

func TestInjectionDestructureNoDefaultNoFP(t *testing.T) {
	// Plain destructuring without a default is fine.
	src := `const { JWT_SECRET } = process.env;`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag destructuring without a literal default")
	}
}

func TestInjectionDestructureTestFileNoFP(t *testing.T) {
	src := `const { JWT_SECRET = 'test-secret' } = process.env;`
	if hasRule(findingsForPath(t, src, "auth.test.ts", analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag in test files")
	}
}

func TestInjectionDestructureNonEnvNoFP(t *testing.T) {
	// Destructuring from something other than process.env must not fire.
	src := `const { JWT_SECRET = 'dev' } = config;`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "security.secret_fallback_literal") {
		t.Error("must NOT flag destructuring from a non-process.env source")
	}
}
