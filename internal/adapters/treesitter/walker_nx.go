package treesitter

import (
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// checkDeepImport flags relative imports that cross lib boundaries.
// Pattern: import from '../../' (two or more levels up) in a non-test file.
func (w *fileWalker) checkDeepImport(line string, lineNum int) {
	if w.lang != analysis.LangTypeScript && w.lang != analysis.LangJavaScript {
		return
	}
	if isTestFile(w.file) {
		return
	}
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "import ") && !strings.Contains(trimmed, "require(") {
		return
	}
	if strings.Contains(line, `'../../`) || strings.Contains(line, `"../../`) {
		w.emitFinding(analysis.Finding{
			Rule:        "nx_conventions.no_deep_import",
			Pillar:      "nx_conventions",
			Severity:    analysis.SeverityMajor,
			Line:        lineNum,
			Message:     "deep relative import crosses library boundary — use the library's public path alias",
			Remediation: "Import via @scope/lib-name instead of a relative path (configure tsconfig paths).",
		})
	}
}
