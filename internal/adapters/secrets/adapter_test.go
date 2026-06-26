package secrets

import (
	"context"
	"strings"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// findingsFor runs the native adapter over a single in-memory file.
func findingsFor(t *testing.T, path, src string) []analysis.Finding {
	t.Helper()
	a := New()
	req := analysis.RunRequest{
		Files: []analysis.FileInfo{{
			Path:     path,
			Language: analysis.LangTypeScript,
			Content:  []byte(src),
		}},
	}
	got, err := a.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	return got
}

func hasSecretRule(findings []analysis.Finding, suffix string) bool {
	want := "security.secrets." + suffix
	for _, f := range findings {
		if f.Rule == want {
			return true
		}
	}
	return false
}

func hasAnySecret(findings []analysis.Finding) bool {
	for _, f := range findings {
		if strings.HasPrefix(f.Rule, "security.secrets.") {
			return true
		}
	}
	return false
}

// ── Port / contract ───────────────────────────────────────────────────────

func TestAdapterContract(t *testing.T) {
	a := New()
	if a.Name() != "secrets" {
		t.Errorf("Name() = %q, want secrets", a.Name())
	}
	if !a.IsAvailable() {
		t.Error("IsAvailable() must always be true for the native scanner")
	}
	caps := a.Capabilities()
	if len(caps) != 1 || caps[0] != "security.secrets" {
		t.Errorf("Capabilities() = %v, want [security.secrets]", caps)
	}
	// Implements the port.
	var _ analysis.ToolAdapter = New()
}

func TestEveryFindingIsBlockerWithProvenance(t *testing.T) {
	f := findingsFor(t, "config.ts", `const k = "AKIAIOSFODNN7EXAMPLE";`)
	if len(f) == 0 {
		t.Fatal("expected at least one finding")
	}
	for _, finding := range f {
		if finding.Severity != analysis.SeverityBlocker {
			t.Errorf("severity = %s, want blocker", finding.Severity)
		}
		if finding.Source != "secrets" {
			t.Errorf("source = %q, want secrets", finding.Source)
		}
		if finding.Pillar != "security" {
			t.Errorf("pillar = %q, want security", finding.Pillar)
		}
		if finding.Line != 1 {
			t.Errorf("line = %d, want 1", finding.Line)
		}
	}
}

// ── Positive fixtures (must detect) ───────────────────────────────────────

func TestDetectsAWSAccessKeyID(t *testing.T) {
	f := findingsFor(t, "infra.ts", `const id = "AKIAIOSFODNN7EXAMPLE";`)
	if !hasSecretRule(f, "aws-access-key-id") {
		t.Fatalf("must detect AWS access key id; got %+v", f)
	}
}

func TestDetectsAWSSecretViaEntropy(t *testing.T) {
	src := `aws_secret_access_key = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"`
	if !hasAnySecret(findingsFor(t, "creds.txt", src)) {
		t.Fatal("must detect a 40-char AWS secret access key")
	}
}

func TestDetectsJWT(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJzdWIiOiIxMjM0NTY3ODkwIn0." +
		"dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	f := findingsFor(t, "auth.ts", `const token = "`+jwt+`";`)
	if !hasSecretRule(f, "jwt") {
		t.Fatalf("must detect JWT; got %+v", f)
	}
}

func TestDetectsPEMPrivateKey(t *testing.T) {
	src := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
	if !hasSecretRule(findingsFor(t, "key.pem", src), "private-key") {
		t.Fatal("must detect PEM private key header")
	}
}

func TestDetectsGitHubToken(t *testing.T) {
	src := `const t = "ghp_0123456789abcdefghijABCDEFGHIJ012345";`
	if !hasSecretRule(findingsFor(t, "ci.js", src), "github-token") {
		t.Fatal("must detect GitHub token")
	}
}

func TestDetectsGenericHighEntropyToken(t *testing.T) {
	src := `const apiKey = "x8Kf3jQ9pLm2Wz7vN4tR6yB1cD5eH0gAaSdFgHj";`
	if !hasSecretRule(findingsFor(t, "client.ts", src), "generic-high-entropy") {
		t.Fatal("must detect generic high-entropy secret behind a secret-ish key")
	}
}

// ── Negative fixtures (must NOT flag) ─────────────────────────────────────

func TestIgnoresUUID(t *testing.T) {
	// Secret-ish key name, but the value is a UUID -> must not flag.
	src := `const apiKey = "550e8400-e29b-41d4-a716-446655440000";`
	if hasAnySecret(findingsFor(t, "ids.ts", src)) {
		t.Fatal("must NOT flag a UUID value")
	}
}

func TestIgnoresSHA256TestHash(t *testing.T) {
	src := `const tokenHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";`
	if hasAnySecret(findingsFor(t, "fixtures.ts", src)) {
		t.Fatal("must NOT flag a sha256 hex digest (test hash)")
	}
}

func TestIgnoresMD5TestHash(t *testing.T) {
	src := `const secret = "d41d8cd98f00b204e9800998ecf8427e";`
	if hasAnySecret(findingsFor(t, "fixtures.ts", src)) {
		t.Fatal("must NOT flag an md5 hex digest")
	}
}

func TestIgnoresLowEntropyValue(t *testing.T) {
	src := `const password = "passwordpasswordpassword";`
	if hasAnySecret(findingsFor(t, "weak.ts", src)) {
		t.Fatal("must NOT flag a low-entropy repetitive string")
	}
}

func TestIgnoresNonSecretKeyName(t *testing.T) {
	// High-entropy value, but the key is not secret-ish -> generic rule skips.
	src := `const requestId = "x8Kf3jQ9pLm2Wz7vN4tR6yB1cD5eH0gAaSdFgHj";`
	if hasAnySecret(findingsFor(t, "trace.ts", src)) {
		t.Fatal("must NOT flag a high-entropy value behind a non-secret key name")
	}
}

func TestIgnoresEmptyAndShortValues(t *testing.T) {
	for _, src := range []string{
		`const secret = "";`,
		`const token = "short";`,
		`const apiKey = "abc123";`,
	} {
		if hasAnySecret(findingsFor(t, "x.ts", src)) {
			t.Errorf("must NOT flag empty/short value: %s", src)
		}
	}
}
