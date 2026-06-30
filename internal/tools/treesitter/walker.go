package treesitter

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// lineScanner provides line-based access to source code.
type lineScanner struct {
	Lines []string
}

// fileWalker performs all AST-based checks on a single parsed file.
type fileWalker struct {
	def                 *LangDef
	src                 []byte
	file                string
	lang                analysis.Language
	stds                analysis.Standards
	matcher             *PatternMatcher  // TOML rule matcher (Phase A)
	isMain              bool             // Go package main — stdout output is legitimate, not a logging bypass
	findings            []analysis.Finding
	scanner             *lineScanner
	fileHasSetHook      bool // Rust: tracks if panic::set_hook found in main.rs
	lastErrorStructLine int  // Rust: tracks last Error struct definition line
}

func newFileWalker(def *LangDef, fi analysis.FileInfo, stds analysis.Standards, matcher *PatternMatcher) *fileWalker {
	lines := strings.Split(string(fi.Content), "\n")
	return &fileWalker{
		def:     def,
		src:     fi.Content,
		file:    fi.Path,
		lang:    fi.Language,
		stds:    stds,
		matcher: matcher,
		isMain:  fi.Language == analysis.LangGo && isGoMainPackage(fi.Content),
		scanner: &lineScanner{Lines: lines},
	}
}

// walk is the entry point: runs all checks against the root node.
func (w *fileWalker) walk(root *sitter.Node) []analysis.Finding {
	w.checkFileLength(root)
	w.walkNode(root, 0, nil)
	w.checkPatterns()
	return w.findings
}

func (w *fileWalker) walkNode(node *sitter.Node, depth int, parentFn *functionScope) {
	t := node.Type()
	if contains(w.def.FunctionTypes, t) {
		w.checkFunction(node)
		return // checkFunction recurses into the function body
	}
	if contains(w.def.ClassTypes, t) {
		w.checkClassLength(node)
	}
	if contains(w.def.CommentTypes, t) {
		w.checkComment(node)
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		w.walkNode(node.Child(i), depth+1, parentFn)
	}
}

// functionScope accumulates metrics as we walk inside a function.
type functionScope struct {
	name        string
	startLine   int
	endLine     int
	cyclomatic  int
	cognitive   int
	maxNesting  int
	params      int
	returns     int      // count of return statements
	paramNames  []string // raw parameter name strings for boolean flag check
}

func (w *fileWalker) checkFunction(fnNode *sitter.Node) {
	scope := &functionScope{
		name:       w.fnName(fnNode),
		startLine:  int(fnNode.StartPoint().Row) + 1,
		endLine:    int(fnNode.EndPoint().Row) + 1,
		cyclomatic: 1, // base complexity
	}
	for i := 0; i < int(fnNode.ChildCount()); i++ {
		child := fnNode.Child(i)
		if child.Type() == w.def.ParameterType {
			w.collectParams(scope, child)
		}
	}
	// nestDepth=-1 so the function body block itself counts as depth 0,
	// making nesting relative to the function interior, not the function node.
	w.walkFunctionBody(fnNode, scope, -1)
	scope.endLine = int(fnNode.EndPoint().Row) + 1
	w.emitFunctionFindings(scope, scope.endLine-scope.startLine+1)
}

func (w *fileWalker) collectParams(scope *functionScope, paramNode *sitter.Node) {
	scope.params = int(paramNode.NamedChildCount())
	for j := 0; j < int(paramNode.NamedChildCount()); j++ {
		scope.paramNames = append(scope.paramNames, paramNode.NamedChild(j).Content(w.src))
	}
}

func (w *fileWalker) walkFunctionBody(node *sitter.Node, scope *functionScope, nestDepth int) {
	t := node.Type()

	if contains(w.def.CommentTypes, t) {
		w.checkComment(node)
	}
	if contains(w.def.BranchTypes, t) {
		scope.cyclomatic++
		scope.cognitive += 1 + max(0, nestDepth)
	}
	if t == "return_statement" {
		scope.returns++
	}
	if contains(w.def.BlockTypes, t) {
		nestDepth++
		if nestDepth > scope.maxNesting {
			scope.maxNesting = nestDepth
		}
	}

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if contains(w.def.FunctionTypes, child.Type()) {
			w.checkFunction(child)
			continue
		}
		w.walkFunctionBody(child, scope, nestDepth)
	}
}
