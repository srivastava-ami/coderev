package treesitter

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// rustGuard returns (trimmed, skip=true) when the line is not Rust or is a comment.
func (w *fileWalker) rustGuard(line string) (string, bool) {
	if w.lang != analysis.LangRust {
		return "", true
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
		return "", true
	}
	return trimmed, false
}

// emitFinding appends a finding, auto-filling File/Source/Snippet and
// enriching Tags/Standards from the rule registry if not already set.
// jsTSGuard returns (trimmed, skip=true) when the line should not be checked.
// Callers that only apply to JS/TS and skip comment lines use this as their single guard.
func (w *fileWalker) jsTSGuard(line string) (string, bool) {
	if w.lang != analysis.LangTypeScript && w.lang != analysis.LangJavaScript {
		return "", true
	}
	t := strings.TrimSpace(line)
	return t, strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*")
}

// pythonGuard returns (trimmed, skip=true) when the line is not Python or is a
// comment or string literal.
func (w *fileWalker) pythonGuard(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "\"") || strings.HasPrefix(trimmed, "'") {
		return "", true
	}
	if w.lang != analysis.LangPython {
		return "", true
	}
	return trimmed, false
}

// codeLineSkip returns true when the line is a comment or import — for checks
// that apply to all languages and share this identical preamble.
func codeLineSkip(line string) bool {
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "*") {
		return true
	}
	return strings.HasPrefix(t, "import ") || strings.HasPrefix(t, "from ")
}

func (w *fileWalker) emitFinding(f analysis.Finding) {
	f.File = w.file
	f.Source = "treesitter"
	f.Snippet = w.snippetAt(f.Line)
	if meta, ok := analysis.RuleRegistry[f.Rule]; ok && len(f.Tags) == 0 {
		f.Tags = meta.Tags
		f.Standards = meta.Standards
	}
	w.findings = append(w.findings, f)
}

func (w *fileWalker) snippetAt(line int) string {
	lines := strings.Split(string(w.src), "\n")
	if line <= 0 || line > len(lines) {
		return ""
	}
	start := max(0, line-2)
	end := min(len(lines), line+2)
	return strings.Join(lines[start:end], "\n")
}

func (w *fileWalker) fnName(node *sitter.Node) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "property_identifier" {
			return child.Content(w.src)
		}
	}
	return "<anonymous>"
}

func (w *fileWalker) className(node *sitter.Node) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "type_identifier" {
			return child.Content(w.src)
		}
	}
	return "<anonymous>"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
