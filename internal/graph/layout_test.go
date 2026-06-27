package graph

import "testing"

func sampleGraph() *Graph {
	return &Graph{
		Nodes: []Node{
			{ID: "a.go", Label: "a.go", Kind: KindFile},
			{ID: "b.go", Label: "b.go", Kind: KindFile},
			{ID: "f1", Label: "f1", Kind: KindFunction},
			{ID: "T", Label: "T", Kind: KindType},
		},
		Edges: []Edge{
			{Source: "a.go", Target: "b.go", Relation: "imports"},
			{Source: "a.go", Target: "f1", Relation: "contains"},
			{Source: "f1", Target: "T", Relation: "uses"},
		},
	}
}

// TestComputeLayoutDeterministic is the core guarantee: same graph, identical
// positions every time — no randomness, no wall-clock.
func TestComputeLayoutDeterministic(t *testing.T) {
	g := sampleGraph()
	a := ComputeLayout(g)
	b := ComputeLayout(g)
	if len(a) != len(g.Nodes) {
		t.Fatalf("expected %d positions, got %d", len(g.Nodes), len(a))
	}
	for id, pa := range a {
		pb, ok := b[id]
		if !ok {
			t.Fatalf("node %q missing on second run", id)
		}
		if pa != pb {
			t.Errorf("non-deterministic layout for %q: %v != %v", id, pa, pb)
		}
	}
}

// TestComputeLayoutInputOrderIndependent: shuffling node order must not change
// any node's position (sorted-ID processing).
func TestComputeLayoutInputOrderIndependent(t *testing.T) {
	g1 := sampleGraph()
	g2 := sampleGraph()
	g2.Nodes[0], g2.Nodes[3] = g2.Nodes[3], g2.Nodes[0]
	// Reverse edge order too — float-addition order must not move any node.
	for i, j := 0, len(g2.Edges)-1; i < j; i, j = i+1, j-1 {
		g2.Edges[i], g2.Edges[j] = g2.Edges[j], g2.Edges[i]
	}
	a := ComputeLayout(g1)
	b := ComputeLayout(g2)
	for id, pa := range a {
		if pb := b[id]; pa != pb {
			t.Errorf("layout changed with input order for %q: %v != %v", id, pa, pb)
		}
	}
}

// TestComputeLayoutWithinBounds: every coordinate stays inside the padded viewport.
func TestComputeLayoutWithinBounds(t *testing.T) {
	pos := ComputeLayout(sampleGraph())
	for id, p := range pos {
		if p.X < layoutPad || p.X > layoutWidth-layoutPad ||
			p.Y < layoutPad || p.Y > layoutHeight-layoutPad {
			t.Errorf("node %q out of bounds: %v", id, p)
		}
	}
}

// TestComputeLayoutEmpty: no nodes, no panic, empty map.
func TestComputeLayoutEmpty(t *testing.T) {
	if got := ComputeLayout(&Graph{}); len(got) != 0 {
		t.Errorf("expected empty layout, got %d", len(got))
	}
}
