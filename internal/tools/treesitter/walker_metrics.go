package treesitter

import (
	"fmt"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Fallback complexity thresholds used when the standards config leaves a value
// unset (zero). They mirror the defaults shipped in the embedded standards.
const (
	defaultMaxCyclomatic       = 8  // max cyclomatic complexity before a finding
	defaultCyclomaticHardBlock = 12 // cyclomatic complexity that hard-blocks
	defaultCyclomaticAdvisory  = 5  // cyclomatic complexity that raises an advisory
	defaultMaxCognitive        = 10 // max cognitive complexity before a finding
	defaultMaxFunctionLines    = 30 // max function length in lines
	defaultMaxParams           = 3  // max function parameters before a finding
	functionLengthAdvisoryBand = 10 // lines a function may exceed max and stay advisory
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

// testSev returns advisory when the file is a test/spec file, otherwise the given
// severity. Prevents long describe/it blocks and test-mock complexity from blocking PRs.
func (w *fileWalker) testSev(sev analysis.Severity) analysis.Severity {
	if isTestFile(w.file) {
		return analysis.SeverityAdvisory
	}
	return sev
}

func (w *fileWalker) checkCyclomatic(s *functionScope, loc string) {
	maxCC := w.stds.Complexity.Cyclomatic.MaxValue
	if maxCC == 0 {
		maxCC = defaultMaxCyclomatic
	}
	hardBlock := w.stds.Complexity.Cyclomatic.HardBlockAt
	if hardBlock == 0 {
		hardBlock = defaultCyclomaticHardBlock
	}
	advisory := w.stds.Complexity.Cyclomatic.AdvisoryAt
	if advisory == 0 {
		advisory = defaultCyclomaticAdvisory
	}

	switch {
	case s.cyclomatic >= hardBlock:
		w.emitFinding(analysis.Finding{Rule: "complexity.cyclomatic", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
			Message:     fmt.Sprintf("%s: cyclomatic complexity %d exceeds hard block (%d)", loc, s.cyclomatic, hardBlock),
			Remediation: w.stds.Complexity.Cyclomatic.Remediation})
	case s.cyclomatic > maxCC:
		w.emitFinding(analysis.Finding{Rule: "complexity.cyclomatic", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
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
		maxCog = defaultMaxCognitive
	}
	if s.cognitive <= maxCog {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.cognitive", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
		Message:     fmt.Sprintf("%s: cognitive complexity %d exceeds max (%d)", loc, s.cognitive, maxCog),
		Remediation: "Flatten nesting with guard clauses; extract inner blocks to named helpers."})
}

func (w *fileWalker) checkFunctionLength(s *functionScope, lines int, loc string) {
	maxLines := w.stds.Complexity.Function.MaxLines
	if maxLines == 0 {
		maxLines = defaultMaxFunctionLines
	}
	if lines <= maxLines {
		return
	}
	sev := w.testSev(analysis.SeverityBlocker)
	if lines <= maxLines+functionLengthAdvisoryBand {
		sev = analysis.SeverityAdvisory
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.function_length", Pillar: "complexity", Severity: sev, Line: s.startLine,
		Message:     fmt.Sprintf("%s: %d lines (max %d)", loc, lines, maxLines),
		Remediation: w.stds.Complexity.Function.Remediation})
}

func (w *fileWalker) checkParams(s *functionScope, loc string) {
	maxParams := w.stds.Complexity.Parameters.MaxCount
	if maxParams == 0 {
		maxParams = defaultMaxParams
	}
	if s.params <= maxParams {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.parameter_count", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
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
	w.emitFinding(analysis.Finding{Rule: "complexity.nesting", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
		Message:     fmt.Sprintf("%s: nesting depth %d (max %d)", loc, s.maxNesting, maxNest),
		Remediation: w.stds.Complexity.Nesting.Remediation})
}

// maxReturnsAdvisory is the early-return count above which a function is flagged.
// 6 matches idiomatic Go guard-clause style; 4 was stricter than necessary and
// pushed readable early-return code toward deeper nesting.
const maxReturnsAdvisory = 6

func (w *fileWalker) checkMaxReturnCount(s *functionScope, loc string) {
	if s.returns <= maxReturnsAdvisory {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "complexity.max_return_count", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
		Message:     fmt.Sprintf("%s: %d return statements (max %d) — consider restructuring", loc, s.returns, maxReturnsAdvisory),
		Remediation: "Use a single return with a result variable, or extract the branches to named helpers."})
}

func (w *fileWalker) checkBooleanParamFlag(s *functionScope, loc string) {
	for _, raw := range s.paramNames {
		name, typeHint := splitParamNameType(raw)
		// If a type annotation is present and is NOT boolean, skip — not a flag param.
		// "hash: string", "count: number" must not fire.
		if typeHint != "" && !isBoolType(typeHint) {
			continue
		}
		if isBoolFlagName(name) {
			w.emitFinding(analysis.Finding{Rule: "complexity.boolean_param_flag", Pillar: "complexity", Severity: analysis.SeverityAdvisory, Line: s.startLine,
				Message:     fmt.Sprintf("%s: boolean flag parameter '%s' — flag arguments make callers hard to read", loc, raw),
				Remediation: "Replace flag params with two separate functions or an options object."})
			return
		}
	}
}

// splitParamNameType splits "name: TypeAnnotation" or "name = default" into
// (name, typeHint). typeHint is empty when there is no annotation (unknown type).
func splitParamNameType(raw string) (name, typeHint string) {
	if idx := strings.Index(raw, ":"); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+1:])
	}
	if idx := strings.Index(raw, "="); idx >= 0 {
		return strings.TrimSpace(raw[:idx]), ""
	}
	return strings.TrimSpace(raw), ""
}

// isBoolType returns true when the type annotation resolves to a boolean variant.
func isBoolType(t string) bool {
	t = strings.TrimSpace(t)
	return t == "boolean" || t == "boolean | undefined" || t == "boolean | null" ||
		strings.HasPrefix(t, "boolean ")
}

// isBoolFlagName returns true when the identifier follows a bool-flag naming
// convention with a camelCase boundary — the char after the prefix must be
// uppercase, which prevents "hash" matching "has" (h is lowercase).
func isBoolFlagName(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"is", "has", "should", "flag", "enable", "disable", "toggle", "show", "hide"} {
		if len(name) > len(prefix) && strings.HasPrefix(lower, prefix) {
			if name[len(prefix)] >= 'A' && name[len(prefix)] <= 'Z' {
				return true
			}
		}
	}
	return false
}
