package graph

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	tsts "github.com/smacker/go-tree-sitter/typescript/typescript"
	tstsx "github.com/smacker/go-tree-sitter/typescript/tsx"

	"github.com/srivastava-ami/coderev/internal/adapters/imports"
	"github.com/srivastava-ami/coderev/internal/analysis"
)

type langCfg struct {
	grammar   *sitter.Language
	funcTypes []string
	typeTypes []string
}

var langConfigs = map[analysis.Language]*langCfg{
	analysis.LangGo: {
		grammar:   golang.GetLanguage(),
		funcTypes: []string{"function_declaration", "method_declaration"},
		typeTypes: []string{"type_declaration"},
	},
	analysis.LangTypeScript: {
		grammar:   tsts.GetLanguage(),
		funcTypes: []string{"function_declaration", "method_definition", "arrow_function"},
		typeTypes: []string{"class_declaration", "interface_declaration"},
	},
	analysis.LangJavaScript: {
		grammar:   javascript.GetLanguage(),
		funcTypes: []string{"function_declaration", "method_definition", "arrow_function"},
		typeTypes: []string{"class_declaration"},
	},
}

// tsx is identical to TypeScript but uses the TSX grammar.
var tsxCfg = &langCfg{
	grammar:   tstsx.GetLanguage(),
	funcTypes: []string{"function_declaration", "method_definition", "arrow_function"},
	typeTypes: []string{"class_declaration", "interface_declaration"},
}

func init() {
	langConfigs[analysis.LangUnknown] = nil
	// TSX uses the same language const as TypeScript but different grammar.
	// We register under a synthesized key so  parseFile can detect
	// .tsx files via analysis.ExtToLanguage[".tsx"] == LangTypeScript.
}

func cleanPath(p string) string { return filepath.Clean(p) }

func isSourceExt(ext string) bool {
	_, ok := analysis.ExtToLanguage[ext]
	return ok
}

func isIgnoredDir(base string) bool {
	return base == "node_modules" || base == ".git" || base == "dist" || base == "build" ||
		base == ".nx" || base == "coverage" || base == ".cache" || base == "vendor" ||
		base == "__pycache__" || base == "target" || base == ".cargo"
}

// Build walks target, builds the code graph from source files.
func Build(target string) (*Graph, error) {
	fis, err := walkSourceFiles(target)
	if err != nil {
		return nil, err
	}
	if len(fis) == 0 {
		return emptyGraph(), nil
	}
	g := buildImportGraph(target, fis)
	funcs := extractDeclarationsFromFiles(g, fis)
	detectCalls(g, funcs)
	return g, nil
}

type funcExtract struct {
	fileID     string
	startBytes uint32
	endBytes   uint32
	content    []byte
}

type declContext struct {
	g      *Graph
	fi     analysis.FileInfo
	cfg    *langCfg
	fileID string
	funcs  map[string]funcExtract
}

// extractDeclarations walks the AST and registers function/type nodes.
func extractDeclarations(ctx *declContext, node *sitter.Node) {
	t := node.Type()

	if contains(ctx.cfg.funcTypes, t) {
		name := funcName(node, ctx.fi.Content)
		if name != "" {
			nodeID := ctx.fileID + ":" + name
			addNodeUnique(ctx.g, Node{
				ID:         nodeID,
				Label:      name,
				Kind:       KindFunction,
				SourceFile: ctx.fi.Path,
			})
			ctx.g.Edges = append(ctx.g.Edges, Edge{Source: ctx.fileID, Target: nodeID, Relation: "contains"})
			ctx.funcs[nodeID] = funcExtract{
				fileID:     ctx.fileID,
				startBytes: node.StartByte(),
				endBytes:   node.EndByte(),
				content:    ctx.fi.Content,
			}
		}
	}

	if contains(ctx.cfg.typeTypes, t) {
		names := typeNames(node, ctx.fi.Content)
		for _, name := range names {
			if name == "" {
				continue
			}
			nodeID := ctx.fileID + ":" + name
			addNodeUnique(ctx.g, Node{
				ID:         nodeID,
				Label:      name,
				Kind:       KindType,
				SourceFile: ctx.fi.Path,
			})
			ctx.g.Edges = append(ctx.g.Edges, Edge{Source: ctx.fileID, Target: nodeID, Relation: "contains"})
		}
	}

	for i := 0; i < int(node.NamedChildCount()); i++ {
		extractDeclarations(ctx, node.NamedChild(i))
	}
}

// funcName extracts the name from a function/type declaration node.
func funcName(node *sitter.Node, src []byte) string {
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return nameNode.Content(src)
	}
	// Fallback: for arrow functions the name lives on the parent
	// variable_declarator.
	if node.Type() == "arrow_function" {
		parent := node.Parent()
		if parent != nil && parent.Type() == "variable_declarator" {
			if nameNode := parent.ChildByFieldName("name"); nameNode != nil {
				return nameNode.Content(src)
			}
		}
	}
	return ""
}

// typeNames extracts all type names from a type_declaration node (Go) or
// class/interface declaration (TS/JS).
func typeNames(node *sitter.Node, src []byte) []string {
	// TS/JS class/interface declarations have a direct "name" field.
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return []string{nameNode.Content(src)}
	}
	// Go type_declaration wraps one or more type_spec children, each with a "name" field.
	var out []string
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		if child.Type() == "type_spec" {
			if nameNode := child.ChildByFieldName("name"); nameNode != nil {
				out = append(out, nameNode.Content(src))
			}
		}
	}
	return out
}

// detectCalls searches function bodies for references to other functions.
func detectCalls(g *Graph, funcs map[string]funcExtract) {
	for callerID, fe := range funcs {
		body := strings.ToLower(string(fe.content[fe.startBytes:fe.endBytes]))
		for calleeID := range funcs {
			if callerID == calleeID {
				continue
			}
			parts := strings.Split(calleeID, ":")
			calleeName := strings.ToLower(parts[len(parts)-1])
			if strings.Contains(body, calleeName+"(") {
				g.Edges = append(g.Edges, Edge{Source: callerID, Target: calleeID, Relation: "calls"})
			}
		}
	}
}

func addNodeUnique(g *Graph, n Node) {
	for _, existing := range g.Nodes {
		if existing.ID == n.ID {
			return
		}
	}
	g.Nodes = append(g.Nodes, n)
}

func walkSourceFiles(target string) ([]analysis.FileInfo, error) {
	var fis []analysis.FileInfo
	if err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if isIgnoredDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := filepath.Ext(path)
		lang, ok := analysis.ExtToLanguage[ext]
		if !ok {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		fis = append(fis, analysis.FileInfo{
			Path:     path,
			Language: lang,
			Lines:    strings.Count(string(content), "\n") + 1,
			Content:  content,
		})
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walking target: %w", err)
	}
	return fis, nil
}

func emptyGraph() *Graph {
	return &Graph{
		FanIn:      make(map[string]int),
		FanOut:     make(map[string]int),
		Centrality: make(map[string]float64),
	}
}

func buildImportGraph(target string, fis []analysis.FileInfo) *Graph {
	req := analysis.RunRequest{Target: target, Files: fis}
	ig := imports.BuildGraph(req)

	g := &Graph{
		FanIn:      make(map[string]int),
		FanOut:     make(map[string]int),
		Centrality: make(map[string]float64),
	}

	for id, n := range ig.Nodes {
		g.Nodes = append(g.Nodes, Node{
			ID:         id,
			Label:      filepath.Base(n.Path),
			Kind:       KindFile,
			SourceFile: n.Path,
		})
	}

	for from, targets := range ig.Edges {
		for to := range targets {
			g.Edges = append(g.Edges, Edge{Source: from, Target: to, Relation: "imports"})
		}
	}

	return g
}

func extractDeclarationsFromFiles(g *Graph, fis []analysis.FileInfo) map[string]funcExtract {
	parser := sitter.NewParser()
	funcs := make(map[string]funcExtract)

	for _, fi := range fis {
		fileID := cleanPath(fi.Path)
		ext := filepath.Ext(fi.Path)
		cfg, ok := langConfigs[fi.Language]
		if !ok {
			continue
		}
		if cfg == nil {
			continue
		}
		if ext == ".tsx" {
			cfg = tsxCfg
		}
		parser.SetLanguage(cfg.grammar)
		tree := parser.Parse(nil, fi.Content)
		if tree == nil {
			continue
		}
		ctx := &declContext{
			g:      g,
			fi:     fi,
			cfg:    cfg,
			fileID: fileID,
			funcs:  funcs,
		}
		extractDeclarations(ctx, tree.RootNode())
	}

	return funcs
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
