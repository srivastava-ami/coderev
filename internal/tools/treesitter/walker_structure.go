package treesitter

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Fallback file-structure length limits used when the standards config leaves a
// value unset (zero). They mirror the defaults shipped in the embedded standards.
const (
	defaultMaxFileLines       = 250 // max file length in lines before a blocker
	defaultFileLengthAdvisory = 150 // file length in lines that raises an advisory
	defaultMaxClassLines      = 120 // max class/type length in lines before a blocker
)

func (w *fileWalker) checkFileLength(root *sitter.Node) {
	lines := int(root.EndPoint().Row) + 1
	maxL := w.stds.FileStructure.FileLength.MaxLines
	if maxL == 0 {
		maxL = defaultMaxFileLines
	}
	advisory := w.stds.FileStructure.FileLength.AdvisoryAt
	if advisory == 0 {
		advisory = defaultFileLengthAdvisory
	}

	switch {
	case lines >= maxL:
		w.emitFinding(analysis.Finding{Rule: "file_structure.file_length", Pillar: "file_structure", Severity: analysis.SeverityAdvisory, Line: 1,
			Message:     fmt.Sprintf("file has %d lines (max %d) — split by concern", lines, maxL),
			Remediation: w.stds.FileStructure.FileLength.Remediation})
	case lines >= advisory:
		w.emitFinding(analysis.Finding{Rule: "file_structure.file_length", Pillar: "file_structure", Severity: analysis.SeverityAdvisory, Line: 1,
			Message:     fmt.Sprintf("file has %d lines (advisory threshold %d)", lines, advisory),
			Remediation: w.stds.FileStructure.FileLength.Remediation})
	}
}

func (w *fileWalker) checkClassLength(node *sitter.Node) {
	start := int(node.StartPoint().Row) + 1
	end := int(node.EndPoint().Row) + 1
	lines := end - start + 1

	maxL := w.stds.FileStructure.ClassLength.MaxLines
	if maxL == 0 {
		maxL = defaultMaxClassLines
	}
	if lines <= maxL {
		return
	}
	w.emitFinding(analysis.Finding{Rule: "file_structure.class_length", Pillar: "file_structure", Severity: analysis.SeverityBlocker, Line: start,
		Message:     fmt.Sprintf("class/type '%s' has %d lines (max %d)", w.className(node), lines, maxL),
		Remediation: w.stds.FileStructure.ClassLength.Remediation})
}
