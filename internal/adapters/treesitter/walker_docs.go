package treesitter

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func (w *fileWalker) checkComment(node *sitter.Node) {
	text := node.Content(w.src)
	line := int(node.StartPoint().Row) + 1

	if looksLikeCode(text) {
		w.emitFinding(analysis.Finding{Rule: "documentation.no_comment_tombstones", Pillar: "documentation", Severity: analysis.SeverityBlocker, Line: line,
			Message:     "commented-out code detected — delete it, git is the undo stack",
			Remediation: "Remove the commented-out block entirely."})
		return
	}

	if strings.Contains(strings.ToUpper(text), "TODO") && !todoHasTicket(text) {
		w.emitFinding(analysis.Finding{Rule: "documentation.todo_format", Pillar: "documentation", Severity: analysis.SeverityMajor, Line: line,
			Message:     "TODO without ticket reference — use TODO(#<issue>)",
			Remediation: "Convert to TODO(#<issue>) or create a ticket and link it."})
	}
}

func looksLikeCode(comment string) bool {
	inner := stripCommentMarkers(comment)
	// Must START with a code keyword — avoids flagging prose that contains "function" or "return" as words.
	for _, tok := range []string{"function ", "function(", "const ", "let ", "var ", "return ", "import ", "export ", "class ", "def "} {
		if strings.HasPrefix(inner, tok) {
			return true
		}
	}
	// Highly distinctive operator tokens are safe to check anywhere in the text.
	for _, tok := range []string{":= ", "->", "() {", "=> {"} {
		if strings.Contains(inner, tok) {
			return true
		}
	}
	return false
}

func stripCommentMarkers(s string) string {
	s = strings.TrimPrefix(s, "//")
	s = strings.TrimPrefix(s, "/*")
	s = strings.TrimSuffix(s, "*/")
	s = strings.TrimPrefix(s, "#")
	return strings.TrimSpace(s)
}

func todoHasTicket(text string) bool {
	upper := strings.ToUpper(text)
	idx := strings.Index(upper, "TODO")
	if idx < 0 {
		return false
	}
	after := text[idx+4:]
	return strings.HasPrefix(after, "(#") || strings.Contains(after, "#")
}
