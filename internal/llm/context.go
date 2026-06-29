package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type DiffHunk struct {
	File    string
	Header  string
	Content string
}

type GraphNeighbor struct {
	ID      string
	File    string
	Label   string
	Callers []string
	Callees []string
}

type ReviewContext struct {
	BaseRef   string
	Hunks     []DiffHunk
	Findings  []analysis.Finding
	Neighbors []GraphNeighbor
}

const reviewSystemInstruction = `You are a senior software engineer reviewing a pull request.
Identify logical issues, missing edge cases, incorrect assumptions, and correctness
concerns that static analysis cannot catch. Be specific: reference file paths and
line numbers. Do NOT repeat findings already listed in <findings>. Do NOT suggest
cosmetic or style changes. Do NOT suggest fixes — describe each issue so the author
can reason about it.
`

const reviewOutputInstruction = `
Write the review grouped by file. For each issue state: file:line, the concern,
and why it matters. If no logical issues are found, say so explicitly.
`

type graphNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	SourceFile string `json:"source_file"`
}

type graphLink struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

type graphData struct {
	Nodes []graphNode `json:"nodes"`
	Links []graphLink `json:"links"`
}

func ParseDiff(data []byte) ([]DiffHunk, error) {
	var hunks []DiffHunk
	sc := bufio.NewScanner(bytes.NewReader(data))
	var curFile string
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				curFile = strings.TrimPrefix(parts[3], "b/")
			}
		case strings.HasPrefix(line, "@@") && curFile != "":
			hunks = append(hunks, DiffHunk{File: curFile, Header: line})
		case len(hunks) > 0:
			hunks[len(hunks)-1].Content += line + "\n"
		}
	}
	return hunks, sc.Err()
}

// GraphNeighborhood loads graphJSON, finds nodes whose source_file is in
// changedFiles, and returns nodes reachable within hops steps via calls/imports
// edges (both directions). Hard cap: 60 nodes.
func GraphNeighborhood(graphJSON []byte, changedFiles []string, hops int) ([]GraphNeighbor, error) {
	var gd graphData
	if err := json.Unmarshal(graphJSON, &gd); err != nil {
		return nil, err
	}
	changed := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		changed[f] = true
	}
	nodeByID := make(map[string]graphNode, len(gd.Nodes))
	var seeds []string
	for _, n := range gd.Nodes {
		nodeByID[n.ID] = n
		if changed[n.SourceFile] {
			seeds = append(seeds, n.ID)
		}
	}
	if len(seeds) == 0 {
		return nil, nil
	}
	visited := bfsNeighbors(gd.Links, seeds, hops)
	return buildNeighborList(gd.Links, nodeByID, visited), nil
}

// bfsNeighbors walks calls/imports edges outward from seeds for hops steps,
// capping the visited set at 60 nodes.
func bfsNeighbors(links []graphLink, seeds []string, hops int) map[string]bool {
	visited := make(map[string]bool, len(seeds))
	queue := make([]string, 0, len(seeds))
	for _, s := range seeds {
		if !visited[s] {
			visited[s] = true
			queue = append(queue, s)
		}
	}
	for hop := 0; hop < hops && len(queue) > 0 && len(visited) < 60; hop++ {
		queue = bfsStep(links, queue, visited)
	}
	return visited
}

// bfsStep expands one BFS layer: returns newly discovered node IDs and
// records them in visited. Stays within the 60-node cap.
func bfsStep(links []graphLink, queue []string, visited map[string]bool) []string {
	var next []string
	for _, id := range queue {
		for _, l := range links {
			if l.Relation != "calls" && l.Relation != "imports" {
				continue
			}
			nb := neighborID(l, id)
			if nb != "" && !visited[nb] && len(visited) < 60 {
				visited[nb] = true
				next = append(next, nb)
			}
		}
	}
	return next
}

func neighborID(l graphLink, id string) string {
	if l.Source == id {
		return l.Target
	}
	if l.Target == id {
		return l.Source
	}
	return ""
}

// buildNeighborList assembles GraphNeighbor values for every visited node ID.
func buildNeighborList(links []graphLink, nodeByID map[string]graphNode, visited map[string]bool) []GraphNeighbor {
	result := make([]GraphNeighbor, 0, len(visited))
	for id := range visited {
		n := nodeByID[id]
		var callers, callees []string
		for _, l := range links {
			if l.Relation != "calls" && l.Relation != "imports" {
				continue
			}
			if l.Target == id {
				callers = append(callers, l.Source)
			}
			if l.Source == id {
				callees = append(callees, l.Target)
			}
		}
		result = append(result, GraphNeighbor{ID: n.ID, File: n.SourceFile, Label: n.Label, Callers: callers, Callees: callees})
	}
	return result
}

func AssemblePrompt(ctx ReviewContext) string {
	var b strings.Builder
	b.WriteString(reviewSystemInstruction)
	b.WriteString("\n")
	if len(ctx.Neighbors) > 0 {
		b.WriteString("<graph_context>\n")
		for _, n := range ctx.Neighbors {
			fmt.Fprintf(&b, "  node %s (%s)\n    callers: %s\n    callees: %s\n",
				n.Label, n.File, strings.Join(n.Callers, ", "), strings.Join(n.Callees, ", "))
		}
		b.WriteString("</graph_context>\n")
	}
	if len(ctx.Findings) > 0 {
		b.WriteString("<findings>\n")
		for _, f := range ctx.Findings {
			fmt.Fprintf(&b, "  [%s] %s:%d %s\n", f.Severity, f.File, f.Line, f.Message)
		}
		b.WriteString("</findings>\n")
	}
	if len(ctx.Hunks) > 0 {
		fmt.Fprintf(&b, "<diff base=%q>\n", ctx.BaseRef)
		for _, h := range ctx.Hunks {
			fmt.Fprintf(&b, "--- %s %s\n%s", h.File, h.Header, h.Content)
		}
		b.WriteString("</diff>\n")
	}
	b.WriteString(reviewOutputInstruction)
	return b.String()
}
