package treesitter

import (
	"fmt"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func (w *fileWalker) emitFunctionFindings(s *functionScope, lines int) {
	loc := fmt.Sprintf("function '%s'", s.name)
	w.checkCyclomatic(s, loc)
	w.checkCognitive(s, loc)
	w.checkFunctionLength(s, lines, loc)
	w.checkParams(s, loc)
	w.checkNesting(s, loc)
	w.checkMaxReturnCount(s, loc)
	w.checkBooleanParamFlag(s, loc)
}

func (w *fileWalker) checkCyclomatic(s *functionScope, loc string) {
	maxCC := w.stds.Complexity.Cyclomatic.MaxValue
	if maxCC == 0 {
		maxCC = 8
	}
	hardBlock := w.stds.Complexity.Cyclomatic.HardBlockAt
	if hardBlock == 0 {
		hardBlock = 12
	}
	advisory := w.stds.Complexity.Cyclomatic.AdvisoryAt
	if advisory == 0 {
		advisory = 5
	}

	switch {
	case s.cyclomatic >= hardBlock:
		w.emitFinding(analysis.Finding{Rule: "complexity.cyclomatic", Pillar: "complexity", Severity: analysis.SeverityBlocker, Line: s.startLine,
			Message:     fmt.Sprintf("%s: cyclomatic complexity %d exceeds hard block (%d)", loc, s.cyclomatic, hardBlock),
			Remediation: w.stds.Complexity.Cyclomatic.Remediation})
	case s.cyclomatic > maxCC:
		w.emitFinding(analysis.Finding{Rule: "complexity.cyclomatic", Pillar: "complexity", Severity: analysis.SeverityBlocker, Line: s.startLine,
			Message:     fmt.Sprintf("%s: cyclomatic complexity %d exceeds max (%d)", loc, s.cyclomatic, maxCC),
			Remediation: w.stds.Complexity.Cyclomatic.Remediation})
	case s.cyclomatic > advisory:
		w.emitFinding(analysis.Finding{Rule: "complexity.cyclomatic", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
			Message:     fmt.Sprintf("%s: cyclomatic complexity %d (advisory at %d)", loc, s.cyclomatic, advisory),
			Remediation: w.stds.Complexity.Cyclomatic.Remediation})
	}
}

func (w *fileWalker) checkCognitive(s *functionScope, loc string) {
	maxCog := w.stds.Complexity.Cognitive.MaxValue
	if maxCog == 0 {
		maxCog = 10
	}
	if s.cognitive <= maxCog {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.cognitive", Pillar: "complexity", Severity: analysis.SeverityBlocker, Line: s.startLine,
		Message:     fmt.Sprintf("%s: cognitive complexity %d exceeds max (%d)", loc, s.cognitive, maxCog),
		Remediation: "Flatten nesting with guard clauses; extract inner blocks to named helpers."})
}

func (w *fileWalker) checkFunctionLength(s *functionScope, lines int, loc string) {
	maxLines := w.stds.Complexity.Function.MaxLines
	if maxLines == 0 {
		maxLines = 30
	}
	if lines <= maxLines {
		return
	}
	sev := analysis.SeverityBlocker
	if lines <= maxLines+10 {
		sev = analysis.SeverityAdvisory
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.function_length", Pillar: "complexity", Severity: sev, Line: s.startLine,
		Message:     fmt.Sprintf("%s: %d lines (max %d)", loc, lines, maxLines),
		Remediation: w.stds.Complexity.Function.Remediation})
}

func (w *fileWalker) checkParams(s *functionScope, loc string) {
	maxParams := w.stds.Complexity.Parameters.MaxCount
	if maxParams == 0 {
		maxParams = 3
	}
	if s.params <= maxParams {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.parameter_count", Pillar: "complexity", Severity: analysis.SeverityBlocker, Line: s.startLine,
		Message:     fmt.Sprintf("%s: %d parameters (max %d) — introduce a parameter object", loc, s.params, maxParams),
		Remediation: w.stds.Complexity.Parameters.Remediation})
}

func (w *fileWalker) checkNesting(s *functionScope, loc string) {
	maxNest := w.stds.Complexity.Nesting.MaxDepth
	if maxNest == 0 {
		maxNest = 2
	}
	if s.maxNesting <= maxNest {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.nesting", Pillar: "complexity", Severity: analysis.SeverityBlocker, Line: s.startLine,
		Message:     fmt.Sprintf("%s: nesting depth %d (max %d)", loc, s.maxNesting, maxNest),
		Remediation: w.stds.Complexity.Nesting.Remediation})
}

func (w *fileWalker) checkMaxReturnCount(s *functionScope, loc string) {
	maxRet := 4
	if s.returns <= maxRet {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.max_return_count", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
		Message:     fmt.Sprintf("%s: %d return statements (max %d) — consider restructuring", loc, s.returns, maxRet),
		Remediation: "Use a single return with a result variable, or extract the branches to named helpers."})
}

func (w *fileWalker) checkBooleanParamFlag(s *functionScope, loc string) {
	for _, name := range s.paramNames {
		if isBoolFlagName(name) {
			w.emitFinding(analysis.Finding{Rule: "complexity.boolean_param_flag", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
				Message:     fmt.Sprintf("%s: boolean flag parameter '%s' — flag arguments make callers hard to read", loc, name),
				Remediation: "Replace flag params with two separate functions or an options object."})
			return
		}
	}
}

func isBoolFlagName(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"is", "has", "should", "flag", "enable", "disable", "toggle", "show", "hide"} {
		if strings.HasPrefix(lower, prefix) && len(name) > len(prefix) {
			return true
		}
	}
	return false
}
