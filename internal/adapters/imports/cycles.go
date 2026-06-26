package imports

import (
	"sort"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Node is a single vertex in the import graph. For TS/JS every source file is a
// node; for Go each file is still a node but edges are drawn at package (dir)
// granularity, so an import of package B links to every file in B.
type Node struct {
	ID       string            // canonical (cleaned) file path — the graph key
	Path     string            // original path as supplied by the runner
	Language analysis.Language
}

// Graph is a reusable import dependency graph: nodes (source files) plus
// directed edges (A imports B). It is intentionally exported and free of any
// finding/severity concerns so later consumers — notably the v1.3.0 code
// graph — can build on the same structure rather than re-parsing imports.
type Graph struct {
	Nodes map[string]*Node            // node ID -> node
	Edges map[string]map[string]bool  // node ID -> set of imported node IDs
}

// NewGraph returns an empty graph ready for AddNode/AddEdge.
func NewGraph() *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Edges: make(map[string]map[string]bool),
	}
}

// AddNode registers a file as a graph node. Re-adding an existing ID is a no-op.
func (g *Graph) AddNode(id, path string, lang analysis.Language) {
	if _, ok := g.Nodes[id]; ok {
		return
	}
	g.Nodes[id] = &Node{ID: id, Path: path, Language: lang}
}

// AddEdge records that node `from` imports node `to`. Self-edges are ignored.
func (g *Graph) AddEdge(from, to string) {
	if from == to {
		return
	}
	if _, ok := g.Nodes[from]; !ok {
		return
	}
	if _, ok := g.Nodes[to]; !ok {
		return
	}
	if g.Edges[from] == nil {
		g.Edges[from] = make(map[string]bool)
	}
	g.Edges[from][to] = true
}

// Successors returns the sorted list of node IDs that `id` imports. Sorting
// keeps cycle output deterministic.
func (g *Graph) Successors(id string) []string {
	out := make([]string, 0, len(g.Edges[id]))
	for to := range g.Edges[id] {
		out = append(out, to)
	}
	sort.Strings(out)
	return out
}

// Cycles returns every circular dependency in the graph as an ordered list of
// node IDs. It runs Tarjan's strongly-connected-components algorithm: any SCC
// with more than one member is a cycle. Output is deterministic.
func (g *Graph) Cycles() [][]string {
	t := &tarjan{
		graph:   g,
		index:   make(map[string]int),
		lowlink: make(map[string]int),
		onStack: make(map[string]bool),
	}

	// Visit nodes in a stable order for deterministic results.
	ids := make([]string, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		if _, seen := t.index[id]; !seen {
			t.strongConnect(id)
		}
	}
	return t.cycles
}

// tarjan holds the mutable state for one SCC traversal.
type tarjan struct {
	graph   *Graph
	counter int
	index   map[string]int
	lowlink map[string]int
	onStack map[string]bool
	stack   []string
	cycles  [][]string
}

func (t *tarjan) strongConnect(v string) {
	t.index[v] = t.counter
	t.lowlink[v] = t.counter
	t.counter++
	t.stack = append(t.stack, v)
	t.onStack[v] = true

	for _, w := range t.graph.Successors(v) {
		if _, seen := t.index[w]; !seen {
			t.strongConnect(w)
			if t.lowlink[w] < t.lowlink[v] {
				t.lowlink[v] = t.lowlink[w]
			}
		} else if t.onStack[w] {
			if t.index[w] < t.lowlink[v] {
				t.lowlink[v] = t.index[w]
			}
		}
	}

	// v is the root of an SCC: pop it off the stack.
	if t.lowlink[v] == t.index[v] {
		var scc []string
		for {
			w := t.stack[len(t.stack)-1]
			t.stack = t.stack[:len(t.stack)-1]
			t.onStack[w] = false
			scc = append(scc, w)
			if w == v {
				break
			}
		}
		if len(scc) > 1 {
			// Tarjan pops in reverse discovery order; reverse for readability.
			for i, j := 0, len(scc)-1; i < j; i, j = i+1, j-1 {
				scc[i], scc[j] = scc[j], scc[i]
			}
			t.cycles = append(t.cycles, scc)
		}
	}
}
