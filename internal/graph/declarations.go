package graph

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

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
