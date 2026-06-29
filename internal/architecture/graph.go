package architecture

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

type rawGraphJSON struct {
	Nodes []struct {
		ID         string `json:"id"`
		Label      string `json:"label"`
		SourceFile string `json:"source_file"`
	} `json:"nodes"`
	Links []struct {
		Source string `json:"source"`
		Target string `json:"target"`
	} `json:"links"`
}

func fromGraphJSON(data []byte) (Summary, bool) {
	var g rawGraphJSON
	if err := json.Unmarshal(data, &g); err != nil || len(g.Nodes) == 0 {
		return Summary{}, false
	}
	fileNodes, fileEdges := buildFileGraph(g)
	if len(fileNodes) == 0 {
		return Summary{}, false
	}
	text := synthesiseGraphText(fileNodes, fileEdges)
	return Summary{Source: "synthesised", Text: text, Nodes: fileNodes, Edges: fileEdges}, true
}

func buildFileGraph(g rawGraphJSON) ([]Node, []Edge) {
	fileOf := make(map[string]string, len(g.Nodes))
	seen := map[string]bool{}
	var nodes []Node
	for _, n := range g.Nodes {
		if n.SourceFile == "" {
			continue
		}
		fileOf[n.ID] = n.SourceFile
		if !seen[n.SourceFile] {
			seen[n.SourceFile] = true
			label := filepath.Base(n.SourceFile)
			nodes = append(nodes, Node{ID: n.SourceFile, Label: label, Type: "lib"})
		}
	}
	edgeSeen := map[string]bool{}
	var edges []Edge
	for _, l := range g.Links {
		src, dst := fileOf[l.Source], fileOf[l.Target]
		if src == "" || dst == "" || src == dst {
			continue
		}
		key := src + "→" + dst
		if !edgeSeen[key] {
			edgeSeen[key] = true
			edges = append(edges, Edge{From: src, To: dst})
		}
	}
	return nodes, edges
}

func synthesiseGraphText(nodes []Node, edges []Edge) string {
	var sb strings.Builder
	sb.WriteString("## Architecture Overview\n\n")
	sb.WriteString("*Derived from the code graph — no architecture document found.*\n\n")
	sb.WriteString("### Files\n\n")
	for _, n := range nodes {
		sb.WriteString("- **" + n.Label + "**\n")
	}
	if len(edges) > 0 {
		sb.WriteString("\n### Call dependencies\n\n")
		for _, e := range edges {
			sb.WriteString("- " + filepath.Base(e.From) + " → " + filepath.Base(e.To) + "\n")
		}
	}
	sb.WriteString("\n> Add a `docs/architecture.md` to replace this synthesised view.")
	return sb.String()
}
