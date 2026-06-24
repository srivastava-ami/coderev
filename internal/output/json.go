package output

import (
	"encoding/json"
	"io"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/quality"
)

type GateOutput struct {
	Summary  GateSummary              `json:"summary"`
	Findings []analysis.Finding       `json:"findings"`
	Warnings []analysis.AdapterWarning `json:"warnings,omitempty"`
	Gate     quality.GateResult       `json:"gate"`
}

type GateSummary struct {
	FilesScanned  int `json:"files_scanned"`
	TotalFindings int `json:"total_findings"`
	Blockers      int `json:"blockers"`
	Majors        int `json:"majors"`
	Advisories    int `json:"advisories"`
}

func WriteJSON(result analysis.RunResult, w io.Writer, gateResult quality.GateResult) error {
	bySev := map[string]int{"blocker": 0, "major": 0, "advisory": 0}
	for _, f := range result.Findings {
		bySev[string(f.Severity)]++
	}
	out := GateOutput{
		Summary: GateSummary{
			FilesScanned:  len(result.Files),
			TotalFindings: len(result.Findings),
			Blockers:      bySev["blocker"],
			Majors:        bySev["major"],
			Advisories:    bySev["advisory"],
		},
		Findings: result.Findings,
		Warnings: result.Warnings,
		Gate:    gateResult,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
