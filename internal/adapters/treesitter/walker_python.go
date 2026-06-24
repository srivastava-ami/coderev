package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func (w *fileWalker) checkPythonPrint(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "print(") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.fmt_print",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityAdvisory,
			Line:        lineNum,
			Message:     "print() call in production code — use structured logging",
			Remediation: "Replace with logger.debug() or appropriate logging.",
		})
	}
}

func (w *fileWalker) checkPythonBareExcept(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "except:") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.no_bare_except",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "bare except: catches all exceptions including SystemExit/KeyboardInterrupt",
			Remediation: "Catch specific exception types: except SpecificError:",
		})
	}
}

func (w *fileWalker) checkPythonEvalExec(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	for _, pat := range []string{"eval(", "exec("} {
		if strings.Contains(trimmed, pat) {
			w.emitFinding(analysis.Finding{
				Rule:        "python.no_eval_exec",
				Pillar:      "python_conventions",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "eval()/exec() of dynamic code — remote code execution risk",
				Remediation: "Use ast.literal_eval() for trusted data or a proper parser.",
			})
			return
		}
	}
}

func (w *fileWalker) checkPythonSQLStringConcat(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	upper := strings.ToUpper(trimmed)
	hasSQL := false
	for _, kw := range []string{"SELECT ", "INSERT ", "UPDATE ", "DELETE "} {
		if strings.Contains(upper, kw) {
			hasSQL = true
			break
		}
	}
	if !hasSQL && !strings.Contains(trimmed, "sql +") && !strings.Contains(trimmed, "query +") {
		return
	}
	if !strings.Contains(trimmed, "+") && !strings.Contains(trimmed, `f"`) {
		return
	}
	w.emitFinding(analysis.Finding{
		Rule:        "python.sql_injection",
		Pillar:      "python_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        lineNum,
		Message:     "possible SQL injection — use parameterized queries, not string formatting",
		Remediation: "Use parameterized queries: cursor.execute('SELECT * FROM t WHERE id = %s', (id,))",
	})
}

func (w *fileWalker) checkPythonSubprocess(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "os.system(") || strings.Contains(trimmed, "os.popen(") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.no_subprocess_shell",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "subprocess with shell=True enables shell injection attacks",
			Remediation: "Remove shell=True and pass args as a list, or use shlex.quote() on inputs.",
		})
		return
	}
	if strings.Contains(trimmed, "subprocess.") && strings.Contains(trimmed, "shell=True") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.no_subprocess_shell",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityBlocker,
			Line:        lineNum,
			Message:     "subprocess with shell=True enables shell injection attacks",
			Remediation: "Remove shell=True and pass args as a list, or use shlex.quote() on inputs.",
		})
	}
}

func (w *fileWalker) checkPythonMutableDefault(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	for _, pat := range []string{"=[]", "={}", "=set()"} {
		if strings.Contains(trimmed, pat) && strings.HasPrefix(trimmed, "def ") {
			w.emitFinding(analysis.Finding{
				Rule:        "python.no_mutable_default",
				Pillar:      "python_conventions",
				Severity:    analysis.SeverityBlocker,
				Line:        lineNum,
				Message:     "mutable default argument — the same list/dict/set is shared across all calls",
				Remediation: "Use None as default and create the mutable inside the function: if x is None: x = []",
			})
			return
		}
	}
}

func (w *fileWalker) checkPythonWildcardImport(line string, lineNum int) {
	trimmed, skip := w.pythonGuard(line)
	if skip {
		return
	}
	if strings.Contains(trimmed, "import *") {
		w.emitFinding(analysis.Finding{
			Rule:        "python.no_wildcard_import",
			Pillar:      "python_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "wildcard import (from module import *) pollutes the namespace",
			Remediation: "Import only what you need: from module import SpecificName",
		})
	}
}
