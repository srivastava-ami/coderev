package report

import (
	"fmt"
	"strings"

	"github.com/srivastava-ami/coderev/internal/architecture"
)

func writeArchSection(b *strings.Builder, r Report) {
	if len(r.Architecture.Nodes) == 0 && r.Architecture.Text == "" {
		return
	}
	b.WriteString("## Architecture\n\n")
	if r.Architecture.Text != "" {
		fmt.Fprintf(b, "%s\n\n", r.Architecture.Text)
	}
	if len(r.Architecture.Nodes) > 0 {
		writeMermaidGraph(b, r.Architecture)
	}
}

func writeMermaidGraph(b *strings.Builder, arch architecture.Summary) {
	b.WriteString("```mermaid\ngraph LR\n")
	nodeIDs := buildNodeIDs(arch.Nodes)
	for _, n := range arch.Nodes {
		label := n.Label
		if label == "" {
			label = n.ID
		}
		fmt.Fprintf(b, "    %s[\"%s\"]\n", nodeIDs[n.ID], label)
	}
	for _, e := range arch.Edges {
		writeMermaidEdge(b, nodeIDs, e)
	}
	b.WriteString("```\n\n")
}

func writeMermaidEdge(b *strings.Builder, nodeIDs map[string]string, e architecture.Edge) {
	from := nodeIDs[e.From]
	if from == "" {
		from = mermaidID(e.From)
	}
	to := nodeIDs[e.To]
	if to == "" {
		to = mermaidID(e.To)
	}
	if e.Label != "" {
		fmt.Fprintf(b, "    %s -->|%s| %s\n", from, e.Label, to)
	} else {
		fmt.Fprintf(b, "    %s --> %s\n", from, to)
	}
}

func buildNodeIDs(nodes []architecture.Node) map[string]string {
	m := make(map[string]string, len(nodes))
	for _, n := range nodes {
		m[n.ID] = mermaidID(n.ID)
	}
	return m
}
