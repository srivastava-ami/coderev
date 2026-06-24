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
