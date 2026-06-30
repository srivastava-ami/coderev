package graph

import (
	"path/filepath"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/srivastava-ami/coderev/internal/tools/imports"
	"github.com/srivastava-ami/coderev/internal/tools/treesitter"
	"github.com/srivastava-ami/coderev/internal/analysis"
)

type langCfg struct {
	grammar   *sitter.Language
	funcTypes []string
	typeTypes []string
}

var langConfigs = map[analysis.Language]*langCfg{
	analysis.LangGo: {
		grammar:   treesitter.GrammarFor(analysis.LangGo),
		funcTypes: []string{"function_declaration", "method_declaration"},
		typeTypes: []string{"type_declaration"},
	},
	analysis.LangTypeScript: {
		grammar:   treesitter.GrammarFor(analysis.LangTypeScript),
		funcTypes: []string{"function_declaration", "method_definition", "arrow_function"},
		typeTypes: []string{"class_declaration", "interface_declaration"},
	},
	analysis.LangJavaScript: {
		grammar:   treesitter.GrammarFor(analysis.LangJavaScript),
		funcTypes: []string{"function_declaration", "method_definition", "arrow_function"},
		typeTypes: []string{"class_declaration"},
	},
}

// tsx is identical to TypeScript but uses the TSX grammar.
var tsxCfg = &langCfg{
	grammar:   treesitter.TSXGrammar(),
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
	fis, err := analysis.CollectSourceFiles(target)
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

func addNodeUnique(g *Graph, n Node) {
	for _, existing := range g.Nodes {
		if existing.ID == n.ID {
			return
		}
	}
	g.Nodes = append(g.Nodes, n)
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
