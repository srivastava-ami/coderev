package secrets

import "regexp"

// patternRule is one high-confidence, named secret pattern. These match
// well-known credential formats with a distinctive shape, so they fire on a
// regex hit alone — no entropy check is needed.
type patternRule struct {
	id   string         // short rule id, becomes security.secrets.<id>
	desc string         // human description used in the finding message
	re   *regexp.Regexp // pattern; the matched substring is masked in the message
}

// patternRules are the structural detectors. Order is not significant; every
// rule is tried on every line.
var patternRules = []patternRule{
	{
		id:   "aws-access-key-id",
		desc: "AWS Access Key ID",
		// AKIA/ASIA/... prefix + 16 base32 chars. Covers long-term, STS, and
		// service-specific key id variants.
		re: regexp.MustCompile(`\b((?:AKIA|ASIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASCA)[A-Z0-9]{16})\b`),
	},
	{
		id:   "jwt",
		desc: "JSON Web Token",
		// header.payload.signature, where header and payload are base64url of a
		// JSON object beginning "{\"" -> starts "eyJ".
		re: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{6,}\.eyJ[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]{6,}\b`),
	},
	{
		id:   "private-key",
		desc: "PEM private key",
		re:   regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA |OPENSSH |PGP )?PRIVATE KEY-----`),
	},
	{
		id:   "github-token",
		desc: "GitHub token",
		// ghp_ (PAT), gho_ (OAuth), ghu_ (user-to-server), ghs_ (server), ghr_ (refresh).
		re: regexp.MustCompile(`\b(gh[pousr]_[A-Za-z0-9]{36,255})\b`),
	},
	{
		id:   "slack-token",
		desc: "Slack token",
		re:   regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`),
	},
}

// reAssignment captures `<key> = "<value>"` / `<key>: '<value>'` style
// assignments in common languages, including single, double, and backtick
// quotes. Group 1 is the key (identifier-ish), group 2 is the unquoted value.
// The value charset forbids quotes and whitespace, so multi-word strings and
// template literals with interpolation are not captured.
// Note: the {20,} length floor mirrors minGenericSecretLen in entropy.go;
// looksLikeSecret re-checks the same bound after capture.
var reAssignment = regexp.MustCompile(
	"(?i)([A-Za-z_][A-Za-z0-9_.\\-]*)\\s*[:=]\\s*[`'\"]([^`'\"\\s]{20,})[`'\"]",
)

// reSecretName gates the generic entropy detector: the assignment's key must
// look secret-ish. This keeps the generic rule deterministic and free of the
// false positives a blanket "scan every literal" approach would produce.
var reSecretName = regexp.MustCompile(
	`(?i)(secret|password|passwd|pwd|token|api[_-]?key|apikey|access[_-]?key|` +
		`private[_-]?key|client[_-]?secret|credential|signing[_-]?key|` +
		`encryption[_-]?key|auth[_-]?token)`,
)
