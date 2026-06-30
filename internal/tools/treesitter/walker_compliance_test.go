package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// ── PCI DSS Encryption Tests ────────────────────────────────────────────

func TestPCIDSSHTTPDetection(t *testing.T) {
	src := `const url = "http://api.example.com/data";`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.pci_dss_encryption") {
		t.Error("must detect http:// (non-HTTPS) connection")
	}
	for _, finding := range f {
		if finding.Rule == "compliance.pci_dss_encryption" &&
			finding.Severity != analysis.SeverityBlocker {
			t.Errorf("HTTP detection must be blocker, got %s", finding.Severity)
		}
	}
}

func TestPCIDSSHTTPSNotFlagged(t *testing.T) {
	src := `const url = "https://api.example.com/data";`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.pci_dss_encryption") {
		t.Error("must NOT flag HTTPS")
	}
}

func TestPCIDSSHTTPInComment(t *testing.T) {
	src := `// See http://example.com for docs`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.pci_dss_encryption") {
		t.Error("must NOT flag http:// in comment")
	}
}

func TestPCIDSSUnencryptedSocket(t *testing.T) {
	src := `const conn = net.Dial("tcp", "example.com:8080");`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.pci_dss_encryption") {
		t.Error("must detect unencrypted TCP socket")
	}
}

func TestPCIDSSWebSocketUnsecure(t *testing.T) {
	src := `const ws = new WebSocket("ws://api.example.com");`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.pci_dss_encryption") {
		t.Error("must detect ws:// (unencrypted WebSocket)")
	}
}

func TestPCIDSSWebSocketSecure(t *testing.T) {
	src := `const ws = new WebSocket("wss://api.example.com");`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.pci_dss_encryption") {
		t.Error("must NOT flag wss:// (secure WebSocket)")
	}
}

// ── HIPAA Audit Logging Tests ──────────────────────────────────────────

func TestHIPAASensitiveOpWithoutLogging(t *testing.T) {
	src := `
function updatePatientRecord(data) {
  const result = db.update("patients", data);
  return result;
}
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.hipaa_audit_logging") {
		t.Error("must detect sensitive operation (updatePatientRecord) without audit logging")
	}
}

func TestHIPAASensitiveOpWithLogging(t *testing.T) {
	src := `
function updatePatientRecord(data) {
  const result = db.update("patients", data);
  logger.audit("Patient record updated", { patientId: data.id });
  return result;
}
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.hipaa_audit_logging") {
		t.Error("must NOT flag if audit logging is present")
	}
}

func TestHIPAADeleteOperationWithoutLogging(t *testing.T) {
	src := `
async function deleteUser(userId) {
  await db.delete("users", { id: userId });
}
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.hipaa_audit_logging") {
		t.Error("must detect delete operation without logging")
	}
}

func TestHIPAAConsoleLog(t *testing.T) {
	src := `
function createRecord(data) {
  db.create("records", data);
  console.log("Record created:", data);
}
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.hipaa_audit_logging") {
		t.Error("console.log counts as audit logging")
	}
}

// ── SOC2 Access Control Tests ──────────────────────────────────────────

func TestSOC2PublicEndpointWithoutAuth(t *testing.T) {
	src := `
app.get("/api/users", (req, res) => {
  res.json(users);
});
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.soc2_access_control") {
		t.Error("must detect public endpoint without authorization")
	}
}

func TestSOC2PublicEndpointWithAuth(t *testing.T) {
	src := `
app.get("/api/users", (req, res) => {
  if (!req.user) return res.status(401).send("Unauthorized");
  res.json(users);
});
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.soc2_access_control") {
		t.Error("must NOT flag endpoint with auth check")
	}
}

func TestSOC2DecoratorAuth(t *testing.T) {
	src := `
@Authorized
@Get("/api/admin")
getAdminPanel() {
  return { data: "admin" };
}
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.soc2_access_control") {
		t.Error("must recognize @Authorized decorator")
	}
}

func TestSOC2PublicStaticUnsafe(t *testing.T) {
	src := `
public static String API_KEY = "secret-key-12345";
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.soc2_access_control") {
		t.Error("must detect public static member without access control")
	}
}

// ── Data Classification Tests ──────────────────────────────────────────

func TestDataClassPIIEmail(t *testing.T) {
	src := `const email = userData.email;`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect PII (email) without classification marker")
	}
	for _, finding := range f {
		if finding.Rule == "compliance.data_classification" &&
			finding.Severity != analysis.SeverityBlocker {
			t.Errorf("PII detection must be blocker, got %s", finding.Severity)
		}
	}
}

func TestDataClassPIIWithMarker(t *testing.T) {
	src := `
/* @pii */
const email = userData.email;
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.data_classification") {
		t.Error("must NOT flag PII with @pii marker")
	}
}

func TestDataClassPIISSN(t *testing.T) {
	src := `
function validateSSN(ssn) {
  return ssn.match(/^\d{3}-\d{2}-\d{4}$/);
}
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect SSN (PII) without marker")
	}
}

func TestDataClassFinancialCardNumber(t *testing.T) {
	src := `const cardNumber = req.body.cardNumber;`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect financial data (cardNumber) without marker")
	}
}

func TestDataClassFinancialWithMarker(t *testing.T) {
	src := `
// @financial
const cardNumber = encryptedData.decrypt();
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.data_classification") {
		t.Error("must NOT flag financial data with @financial in comment")
	}
}

func TestDataClassFinancialBankAccount(t *testing.T) {
	src := `
const accountNumber = patient.bank_account;
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect bank_account (financial) without marker")
	}
}

func TestDataClassHealthDiagnosis(t *testing.T) {
	src := `const diagnosis = patient.diagnosis;`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect health data (diagnosis) without marker")
	}
}

func TestDataClassHealthWithMarker(t *testing.T) {
	src := `
/* @health */
const diagnosis = medicalRecord.diagnosis;
`
	if hasRule(findingsForSrc(t, src, analysis.LangTypeScript), "compliance.data_classification") {
		t.Error("must NOT flag health data with @health marker")
	}
}

func TestDataClassHealthMedication(t *testing.T) {
	src := `
function prescribeMedication(medication) {
  return db.save("prescriptions", { medication });
}
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect medication (health data) without marker")
	}
}

func TestDataClassHealthAllergy(t *testing.T) {
	src := `const patientAllergies = patient.allergy;`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect allergy (health data) without marker")
	}
}

func TestDataClassPIIPhone(t *testing.T) {
	src := `const contact = { phone: userData.phone };`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect phone (PII) without marker")
	}
}

func TestDataClassPIIAddress(t *testing.T) {
	src := `
function updateProfile(address) {
  return db.update("users", { address });
}
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect address (PII) without marker")
	}
}

func TestDataClassFinancialSalary(t *testing.T) {
	src := `const salary = employee.salary;`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect salary (financial data) without marker")
	}
}

func TestDataClassPIIFirstLastName(t *testing.T) {
	src := `
const user = {
  firstName: data.firstName,
  lastName: data.lastName
};
`
	f := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(f, "compliance.data_classification") {
		t.Error("must detect firstName/lastName (PII) without markers")
	}
}
