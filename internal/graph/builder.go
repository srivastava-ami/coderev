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

// Build walks target, builds the code graph from source files.
func Build(target string) (*Graph, error) {
	var fis []analysis.FileInfo
	if err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "node_modules" || base == ".git" || base == "dist" || base == "build" ||
				base == ".nx" || base == "coverage" || base == ".cache" || base == "vendor" ||
				base == "__pycache__" || base == "target" || base == ".cargo" {
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

	if len(fis) == 0 {
		return &Graph{
			FanIn:      make(map[string]int),
			FanOut:     make(map[string]int),
			Centrality: make(map[string]float64),
		}, nil
	}

	// Reuse imports.BuildGraph for file-level import edges.
	req := analysis.RunRequest{Target: target, Files: fis}
	ig := imports.BuildGraph(req)

	g := &Graph{
		FanIn:      make(map[string]int),
		FanOut:     make(map[string]int),
		Centrality: make(map[string]float64),
	}

	// File nodes.
	for id, n := range ig.Nodes {
		g.Nodes = append(g.Nodes, Node{
			ID:         id,
			Label:      filepath.Base(n.Path),
			Kind:       KindFile,
			SourceFile: n.Path,
		})
	}

	// Import edges.
	for from, targets := range ig.Edges {
		for to := range targets {
			g.Edges = append(g.Edges, Edge{Source: from, Target: to, Relation: "imports"})
		}
	}

	// Function / type extraction via tree-sitter.
	parser := sitter.NewParser()
	allFuncs := make(map[string]funcExtract)

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
		// Use TSX grammar for .tsx files.
		if ext == ".tsx" {
			cfg = tsxCfg
		}
		parser.SetLanguage(cfg.grammar)
		tree := parser.Parse(nil, fi.Content)
		if tree == nil {
			continue
		}
		extractDeclarations(g, tree.RootNode(), fi, cfg, fileID, allFuncs)
	}

	detectCalls(g, allFuncs)

	return g, nil
}

type funcExtract struct {
	fileID     string
	startBytes uint32
	endBytes   uint32
	content    []byte
}

// extractDeclarations walks the AST and registers function/type nodes.
func extractDeclarations(g *Graph, node *sitter.Node, fi analysis.FileInfo, cfg *langCfg, fileID string, funcs map[string]funcExtract) {
	t := node.Type()

	if contains(cfg.funcTypes, t) {
		name := funcName(node, fi.Content)
		if name != "" {
			nodeID := fileID + ":" + name
			addNodeUnique(g, Node{
				ID:         nodeID,
				Label:      name,
				Kind:       KindFunction,
				SourceFile: fi.Path,
			})
			g.Edges = append(g.Edges, Edge{Source: fileID, Target: nodeID, Relation: "contains"})
			funcs[nodeID] = funcExtract{
				fileID:     fileID,
				startBytes: node.StartByte(),
				endBytes:   node.EndByte(),
				content:    fi.Content,
			}
		}
	}

	if contains(cfg.typeTypes, t) {
		names := typeNames(node, fi.Content)
		for _, name := range names {
			if name == "" {
				continue
			}
			nodeID := fileID + ":" + name
			addNodeUnique(g, Node{
				ID:         nodeID,
				Label:      name,
				Kind:       KindType,
				SourceFile: fi.Path,
			})
			g.Edges = append(g.Edges, Edge{Source: fileID, Target: nodeID, Relation: "contains"})
		}
	}

	for i := 0; i < int(node.NamedChildCount()); i++ {
		extractDeclarations(g, node.NamedChild(i), fi, cfg, fileID, funcs)
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

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
