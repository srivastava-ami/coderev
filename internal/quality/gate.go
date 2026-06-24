package quality

import (
	"fmt"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/config"
)

type GateResult struct {
	Passed     bool   `json:"passed"`
	Blockers   int    `json:"blockers"`
	Majors     int    `json:"majors"`
	Advisories int    `json:"advisories"`
	Total      int    `json:"total"`
	Message    string `json:"message"`
}

func Evaluate(findings []analysis.Finding, gate config.GateConfig) GateResult {
	blockers, majors, advisories := 0, 0, 0
	for _, f := range findings {
		switch f.Severity {
		case analysis.SeverityBlocker:
			blockers++
		case analysis.SeverityMajor:
			majors++
		case analysis.SeverityAdvisory:
			advisories++
		}
	}
	total := len(findings)

	return GateResult{
		Passed:     blockers <= gate.Blockers && majors <= gate.Majors && advisories <= gate.Advisories && total <= gate.Total,
		Blockers:   blockers,
		Majors:     majors,
		Advisories: advisories,
		Total:      total,
		Message:    gateMessage(blockers, majors, advisories, total, gate),
	}
}

func gateMessage(blockers, majors, advisories, total int, gate config.GateConfig) string {
	switch {
	case blockers > gate.Blockers:
		return fmt.Sprintf("%d blocker(s) exceed threshold of %d", blockers, gate.Blockers)
	case majors > gate.Majors:
		return fmt.Sprintf("%d major(s) exceed threshold of %d", majors, gate.Majors)
	case advisories > gate.Advisories:
		return fmt.Sprintf("%d advisory(s) exceed threshold of %d", advisories, gate.Advisories)
	case total > gate.Total:
		return fmt.Sprintf("%d total finding(s) exceed threshold of %d", total, gate.Total)
	}
	return ""
}
