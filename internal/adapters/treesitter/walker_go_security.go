package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// checkGoSQLStringConcat flags SQL queries built with fmt.Sprintf or string concatenation.
func (w *fileWalker) checkGoSQLStringConcat(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip || isTestFile(w.file) {
		return
	}
	msg := goSQLMessage(trimmed)
	if msg == "" {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "go.sql_string_concat",
		Pillar:      "security",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     msg,
		Remediation: "Use parameterised queries: db.Query(q, args...). Never format user input into SQL.",
	})
}

func goSQLMessage(trimmed string) string {
	if strings.Contains(trimmed, "fmt.Sprintf(") && goLineHasSQLKeyword(trimmed) {
		return "SQL query built with fmt.Sprintf — SQL injection vector"
	}
	if goLineHasSQLConcat(trimmed) {
		return "SQL query assembled by string concatenation — SQL injection vector"
	}
	return ""
}

func goLineHasSQLKeyword(trimmed string) bool {
	upper := strings.ToUpper(trimmed)
	for _, kw := range []string{"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "WHERE ", "FROM "} {
		if strings.Contains(upper, kw) {
			return true
		}
	}
	return false
}

// goSQLKeywords are SQL fragments that, when string-concatenated onto a query,
// indicate hand-built SQL. They are stored WITHOUT the `+ "` concatenation
// prefix (which is added at match time) so this detector's own pattern list
// does not trip the sql_string_concat check on this very file.
var goSQLKeywords = []string{"SELECT", "INSERT", "UPDATE", "DELETE", " WHERE ", " AND ", " OR "}

func goLineHasSQLConcat(trimmed string) bool {
	upper := strings.ToUpper(trimmed)
	for _, kw := range goSQLKeywords {
		if strings.Contains(upper, `+ "`+kw) {
			return true
		}
	}
	return false
}

// checkGoFmtErrorfNoFormat flags fmt.Errorf calls where the format string is a variable.
func (w *fileWalker) checkGoFmtErrorfNoFormat(line string, lineNum int) {
	trimmed, skip := w.goGuard(line)
	if skip {
		return
	}
	idx := strings.Index(trimmed, "fmt.Errorf(")
	if idx < 0 {
		return
	}
	after := strings.TrimSpace(trimmed[idx+len("fmt.Errorf("):])
	if after == "" || after[0] == '"' || after[0] == '`' {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "go.fmt_errorf_no_format",
		Pillar:      "security",
		Severity:    analysis.SeverityMajor,
		Line:        lineNum,
		Message:     "fmt.Errorf called with a variable as format string — use fmt.Errorf(\"%s\", err) or errors.New(msg)",
		Remediation: "Pass a string literal format: fmt.Errorf(\"%s\", variable) or errors.New(variable.Error()).",
	})
}

// checkGoIOCopyNoLimit flags io.Copy calls in files that import net/http without a size guard.
func (w *fileWalker) checkGoIOCopyNoLimit(lines []string) {
	if w.lang != analysis.LangGo {
		return
	}
	src := string(w.src)
	if !strings.Contains(src, `"net/http"`) {
		return
	}
	if strings.Contains(src, "io.LimitReader") {
		return
	}
	for i, line := range lines {
		if strings.Contains(line, "io.Copy(") && !strings.HasPrefix(strings.TrimSpace(line), "//") {
			w.emitFinding(analysis.Finding{
				Rule:        "go.io_copy_no_limit",
				Pillar:      "stability",
				Severity:    analysis.SeverityMajor,
				Line:        i + 1,
				Message:     "io.Copy from HTTP response body without io.LimitReader — unbounded write can exhaust disk",
				Remediation: "Wrap the reader: io.Copy(dst, io.LimitReader(resp.Body, maxBytes)).",
			})
		}
	}
}
