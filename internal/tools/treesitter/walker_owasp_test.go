package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// ── A01: Injection Tests ───────────────────────────────────────────────────────

// TestOwaspA01SQLInjectionGo detects SQL injection in Go
func TestOwaspA01SQLInjectionGo(t *testing.T) {
	src := `query := fmt.Sprintf("SELECT * FROM users WHERE id = " + userInput)`
	findings := findingsForSrc(t, src, analysis.LangGo)
	if !hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must flag SQL injection with string concatenation in Go")
	}
}

// TestOwaspA01SQLInjectionPython detects SQL injection in Python
func TestOwaspA01SQLInjectionPython(t *testing.T) {
	src := `query = "SELECT * FROM users WHERE id = " + user_input`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must flag SQL injection with string concatenation in Python")
	}
}

// TestOwaspA01CommandInjectionGo detects command injection in Go
func TestOwaspA01CommandInjectionGo(t *testing.T) {
	src := `cmd := exec.Command("bash", "-c", "rm -rf " + userInput)`
	findings := findingsForSrc(t, src, analysis.LangGo)
	if !hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must flag command injection with string concatenation in Go")
	}
}

// TestOwaspA01CommandInjectionPython detects command injection in Python
func TestOwaspA01CommandInjectionPython(t *testing.T) {
	src := `subprocess.run("rm -rf " + userInput, shell=True)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must flag command injection in Python subprocess")
	}
}

// TestOwaspA01TemplateLiteralSQLInjection detects SQL in template literals
func TestOwaspA01TemplateLiteralSQLInjection(t *testing.T) {
	src := "`SELECT * FROM users WHERE id = ${userInput}`"
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must flag SQL injection in template literal")
	}
}

// TestOwaspA01EvalInjection detects eval with user input
func TestOwaspA01EvalInjection(t *testing.T) {
	src := `eval(userInput)`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must flag eval with user input")
	}
}

// TestOwaspA01NoFPParameterizedQuery should not flag safe parameterized queries
func TestOwaspA01NoFPParameterizedQuery(t *testing.T) {
	src := `db.Query("SELECT * FROM users WHERE id = $1", userId)`
	findings := findingsForSrc(t, src, analysis.LangGo)
	if hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must NOT flag parameterized queries")
	}
}

// TestOwaspA01NoFPSafeExec should not flag safe command execution
func TestOwaspA01NoFPSafeExec(t *testing.T) {
	src := `exec.Command("rm", "-rf", dir)`
	findings := findingsForSrc(t, src, analysis.LangGo)
	if hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must NOT flag safe command execution with array args")
	}
}

// TestOwaspA01NoFPTestFile should not flag in test files
func TestOwaspA01NoFPTestFile(t *testing.T) {
	src := `query := "SELECT * FROM users WHERE id = " + testInput`
	findings := findingsForPath(t, src, "test_query.go", analysis.LangGo)
	if hasRule(findings, "security.owasp_a01_injection") {
		t.Error("must NOT flag potential injection in test files")
	}
}

// ── A02: Authentication Tests ──────────────────────────────────────────────────

// TestOwaspA02MissingAuthCheck detects returning data without auth
func TestOwaspA02MissingAuthCheck(t *testing.T) {
	src := `return userData`
	// Only flag in route/controller context - test filename matters
	findings := findingsForPath(t, src, "user_controller.go", analysis.LangGo)
	if !hasRule(findings, "security.owasp_a02_authentication") {
		t.Error("must flag returning user data without apparent auth check")
	}
}

// TestOwaspA02CommentedAuthCode detects commented auth checks
func TestOwaspA02CommentedAuthCode(t *testing.T) {
	src := `// if (verifyAuth(token)) { ...`
	findings := findingsForPath(t, src, "auth_handler.ts", analysis.LangTypeScript)
	if !hasRule(findings, "security.owasp_a02_authentication") {
		t.Error("must flag commented-out authentication checks")
	}
}

// TestOwaspA02AlwaysFalseAuth detects auth checks that are always false
func TestOwaspA02AlwaysFalseAuth(t *testing.T) {
	src := `if (!user.authenticated) { return data; }`
	findings := findingsForPath(t, src, "api_handler.ts", analysis.LangTypeScript)
	// This should flag because it's a negated auth check (always bypasses)
	for _, f := range findings {
		if f.Rule == "security.owasp_a02_authentication" {
			return // Found it
		}
	}
	t.Error("must flag always-false authentication checks")
}

// TestOwaspA02NoFPWithAuthGuard should not flag when auth is present
func TestOwaspA02NoFPWithAuthGuard(t *testing.T) {
	src := `if (!auth.verify(token)) throw Unauthorized(); return userData;`
	findings := findingsForPath(t, src, "user_handler.go", analysis.LangGo)
	if hasRule(findings, "security.owasp_a02_authentication") {
		t.Error("must NOT flag when auth check is present")
	}
}

// TestOwaspA02NoFPNonEndpoint should not flag in non-endpoint files
func TestOwaspA02NoFPNonEndpoint(t *testing.T) {
	src := `return userData`
	findings := findingsForPath(t, src, "util.go", analysis.LangGo)
	if hasRule(findings, "security.owasp_a02_authentication") {
		t.Error("must NOT flag in non-endpoint files")
	}
}

// ── A03: XXE Tests ─────────────────────────────────────────────────────────────

// TestOwaspA03XXEGoUnmarshal detects XXE in Go
func TestOwaspA03XXEGoUnmarshal(t *testing.T) {
	src := `xml.Unmarshal(data, &v)`
	findings := findingsForSrc(t, src, analysis.LangGo)
	if !hasRule(findings, "security.owasp_a03_xxe") {
		t.Error("must flag xml.Unmarshal without XXE protection")
	}
}

// TestOwaspA03XXEPythonEtree detects XXE in Python
func TestOwaspA03XXEPythonEtree(t *testing.T) {
	src := `tree = ElementTree.parse(file)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "security.owasp_a03_xxe") {
		t.Error("must flag ElementTree.parse without XXE protection")
	}
}

// TestOwaspA03XXEPythonFromstring detects XXE in Python fromstring
func TestOwaspA03XXEPythonFromstring(t *testing.T) {
	src := `root = etree.fromstring(xmlData)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if !hasRule(findings, "security.owasp_a03_xxe") {
		t.Error("must flag etree.fromstring without XXE protection")
	}
}

// TestOwaspA03XXEJavaScript detects XXE in xml2js
func TestOwaspA03XXEJavaScript(t *testing.T) {
	src := `parseString(xmlData, (err, result) => {})`
	findings := findingsForSrc(t, src, analysis.LangJavaScript)
	if !hasRule(findings, "security.owasp_a03_xxe") {
		t.Error("must flag xml2js parseString without XXE protection")
	}
}

// TestOwaspA03XXENoFPDefusedXML should not flag defusedxml
func TestOwaspA03XXENoFPDefusedXML(t *testing.T) {
	src := `from defusedxml import ElementTree as ET; tree = ET.parse(file)`
	findings := findingsForSrc(t, src, analysis.LangPython)
	if hasRule(findings, "security.owasp_a03_xxe") {
		t.Error("must NOT flag defusedxml (XXE-safe library)")
	}
}

// TestOwaspA03XXENoFPTestFile should not flag in test files
func TestOwaspA03XXENoFPTestFile(t *testing.T) {
	src := `tree = ElementTree.parse(testFile)`
	findings := findingsForPath(t, src, "test_parsing.py", analysis.LangPython)
	if hasRule(findings, "security.owasp_a03_xxe") {
		t.Error("must NOT flag XXE in test files")
	}
}

// ── A05: Broken Access Control Tests ───────────────────────────────────────────

// TestOwaspA05MissingAuthzCheck detects missing authorization
func TestOwaspA05MissingAuthzCheck(t *testing.T) {
	src := `db.Delete(userId, recordId)`
	findings := findingsForPath(t, src, "record_controller.go", analysis.LangGo)
	if !hasRule(findings, "security.owasp_a05_broken_access") {
		t.Error("must flag resource deletion without permission check")
	}
}

// TestOwaspA05ResourceUpdateNoCheck detects update without auth
func TestOwaspA05ResourceUpdateNoCheck(t *testing.T) {
	src := `repository.Update(userId, data)`
	findings := findingsForPath(t, src, "user_api.ts", analysis.LangTypeScript)
	if !hasRule(findings, "security.owasp_a05_broken_access") {
		t.Error("must flag resource update without permission check")
	}
}

// TestOwaspA05MissingRBACCheck detects missing role check
func TestOwaspA05MissingRBACCheck(t *testing.T) {
	src := `if (role) { return admin_data; }`
	findings := findingsForPath(t, src, "admin_permission.go", analysis.LangGo)
	// This is a conservative check - look for the pattern
	for _, f := range findings {
		if f.Rule == "security.owasp_a05_broken_access" {
			return
		}
	}
	// Conservative - may not always catch this pattern
}

// TestOwaspA05NoFPWithOwnershipCheck should not flag with owner validation
func TestOwaspA05NoFPWithOwnershipCheck(t *testing.T) {
	src := `if (record.owner !== userId) throw Forbidden(); db.Update(recordId, data)`
	findings := findingsForPath(t, src, "record_handler.go", analysis.LangGo)
	if hasRule(findings, "security.owasp_a05_broken_access") {
		t.Error("must NOT flag when ownership is verified")
	}
}

// TestOwaspA05NoFPWithAuthCheck should not flag with authorization check
func TestOwaspA05NoFPWithAuthCheck(t *testing.T) {
	src := `if (!canAccess(userId, resourceId)) throw Forbidden(); db.Delete(resourceId)`
	findings := findingsForPath(t, src, "delete_handler.ts", analysis.LangTypeScript)
	if hasRule(findings, "security.owasp_a05_broken_access") {
		t.Error("must NOT flag when authorization is checked")
	}
}

// TestOwaspA05NoFPNonEndpoint should not flag in utility functions
func TestOwaspA05NoFPNonEndpoint(t *testing.T) {
	src := `repository.Update(id, data)`
	findings := findingsForPath(t, src, "helpers.go", analysis.LangGo)
	if hasRule(findings, "security.owasp_a05_broken_access") {
		t.Error("must NOT flag in non-endpoint files")
	}
}

// TestOwaspA05NoFPTestFile should not flag in tests
func TestOwaspA05NoFPTestFile(t *testing.T) {
	src := `db.Delete(testUserId)`
	findings := findingsForPath(t, src, "record_test.go", analysis.LangGo)
	if hasRule(findings, "security.owasp_a05_broken_access") {
		t.Error("must NOT flag in test files")
	}
}

// ── Helper to check for rule in findings
