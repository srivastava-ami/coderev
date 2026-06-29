package graph

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed graph_template.html
var graphHTMLTemplate string

// sortedGraph returns copies of the graph's nodes and edges in a canonical order
// (nodes by ID; edges by source, then target, then relation). Exports use this so
// output is byte-for-byte deterministic regardless of the builder's internal map
// iteration order.
func sortedGraph(g *Graph) ([]Node, []Edge) {
	ns := make([]Node, len(g.Nodes))
	copy(ns, g.Nodes)
	sort.Slice(ns, func(i, j int) bool { return ns[i].ID < ns[j].ID })
	es := make([]Edge, len(g.Edges))
	copy(es, g.Edges)
	sort.Slice(es, func(i, j int) bool {
		if es[i].Source != es[j].Source {
			return es[i].Source < es[j].Source
		}
		if es[i].Target != es[j].Target {
			return es[i].Target < es[j].Target
		}
		return es[i].Relation < es[j].Relation
	})
	return ns, es
}

// jsonGraph is coderev's native code-graph JSON structure.
type jsonGraph struct {
	Directed bool       `json:"directed"`
	Nodes    []jsonNode `json:"nodes"`
	Links    []jsonLink `json:"links"`
}

type jsonNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	SourceFile string `json:"source_file"`
}

type jsonLink struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

// ExportJSON writes coderev's native graph.json to dir.
func ExportJSON(g *Graph, dir string) error {
	snodes, sedges := sortedGraph(g)
	gf := jsonGraph{Directed: true}
	for _, n := range snodes {
		gf.Nodes = append(gf.Nodes, jsonNode{
			ID:         n.ID,
			Label:      n.Label,
			Kind:       string(n.Kind),
			SourceFile: n.SourceFile,
		})
	}
	for _, e := range sedges {
		gf.Links = append(gf.Links, jsonLink{
			Source:   e.Source,
			Target:   e.Target,
			Relation: e.Relation,
		})
	}

	data, err := json.MarshalIndent(gf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal graph.json: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.WriteFile(filepath.Join(dir, "graph.json"), data, 0o644)
}

// buildHTMLPayload serialises sorted function/type nodes and edges into JSON.
func buildHTMLPayload(g *Graph, pos map[string]Position) ([]byte, []byte) {
	snodes, sedges := sortedGraph(g)
	type jsNode struct {
		ID         string  `json:"id"`
		Label      string  `json:"label"`
		Group      string  `json:"group"`
		SourceFile string  `json:"source_file"`
		X          float64 `json:"x"`
		Y          float64 `json:"y"`
	}
	type jsEdge struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	nodes := make([]jsNode, 0, len(snodes))
	for _, n := range snodes {
		p := pos[n.ID]
		nodes = append(nodes, jsNode{ID: n.ID, Label: n.Label, Group: string(n.Kind), SourceFile: n.SourceFile, X: p.X, Y: p.Y})
	}
	edges := make([]jsEdge, 0, len(sedges))
	for _, e := range sedges {
		edges = append(edges, jsEdge{Source: e.Source, Target: e.Target})
	}
	nodesJSON, _ := json.Marshal(nodes)
	edgesJSON, _ := json.Marshal(edges)
	return nodesJSON, edgesJSON
}

// buildFilePayload aggregates function nodes into file-level nodes and edges.
func buildFilePayload(g *Graph, pos map[string]Position) ([]byte, []byte) {
	snodes, sedges := sortedGraph(g)
	type fileMeta struct{ sumX, sumY float64; count, fnCount int }
	meta := map[string]*fileMeta{}
	fnToFile := map[string]string{}
	for _, n := range snodes {
		p := pos[n.ID]
		m := meta[n.SourceFile]
		if m == nil {
			m = &fileMeta{}
			meta[n.SourceFile] = m
		}
		m.sumX += p.X; m.sumY += p.Y; m.count++
		if n.Kind == KindFunction || n.Kind == KindType {
			m.fnCount++
		}
		fnToFile[n.ID] = n.SourceFile
	}
	files := make([]string, 0, len(meta))
	for f := range meta {
		files = append(files, f)
	}
	sort.Strings(files)
	type fileNode struct {
		ID      string  `json:"id"`
		Label   string  `json:"label"`
		Kind    string  `json:"kind"`
		X       float64 `json:"x"`
		Y       float64 `json:"y"`
		FnCount int     `json:"fn_count"`
	}
	fnodes := make([]fileNode, 0, len(files))
	for _, f := range files {
		m := meta[f]
		n := float64(m.count)
		fnodes = append(fnodes, fileNode{ID: f, Label: filepath.Base(f),
			Kind: "file", X: m.sumX / n, Y: m.sumY / n, FnCount: m.fnCount})
	}
	type fileEdge struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	seen := map[string]bool{}
	var fedges []fileEdge
	for _, e := range sedges {
		sf, tf := fnToFile[e.Source], fnToFile[e.Target]
		if sf == "" || tf == "" || sf == tf {
			continue
		}
		if key := sf + "|" + tf; !seen[key] {
			seen[key] = true
			fedges = append(fedges, fileEdge{Source: sf, Target: tf})
		}
	}
	fnodesJSON, _ := json.Marshal(fnodes)
	fedgesJSON, _ := json.Marshal(fedges)
	return fnodesJSON, fedgesJSON
}

// ExportGraphHTML writes a fully self-contained HTML page into dir/graph.html.
func ExportGraphHTML(g *Graph, dir string) error {
	pos := ComputeLayout(g)
	nodesJSON, edgesJSON := buildHTMLPayload(g, pos)
	filesJSON, fileLinksJSON := buildFilePayload(g, pos)
	html := graphHTMLTemplate
	html = strings.ReplaceAll(html, "__NODES_JSON__", string(nodesJSON))
	html = strings.ReplaceAll(html, "__LINKS_JSON__", string(edgesJSON))
	html = strings.ReplaceAll(html, "__FILES_JSON__", string(filesJSON))
	html = strings.ReplaceAll(html, "__FILE_LINKS_JSON__", string(fileLinksJSON))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return os.WriteFile(filepath.Join(dir, "graph.html"), []byte(html), 0o644)
}

// ExportAll writes both graph.json and graph.html into dir.
func ExportAll(g *Graph, dir string) error {
	if err := ExportJSON(g, dir); err != nil {
		return err
	}
	return ExportGraphHTML(g, dir)
}
