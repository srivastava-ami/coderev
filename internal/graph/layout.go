package graph

import (
	"math"
	"sort"
)

// Position is a 2-D coordinate in the layout viewport.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Layout viewport — the SVG coordinate space the HTML renders into.
const (
	layoutWidth  = 1600.0
	layoutHeight = 1000.0
	layoutPad    = 60.0
	layoutIters  = 300
)

// ComputeLayout produces a DETERMINISTIC 2-D placement of the graph's nodes
// using a fixed-seed, fixed-iteration Fruchterman-Reingold force model. There is
// no randomness, no wall-clock, and no external dependency: the same graph
// always yields byte-identical positions. Nodes are processed in sorted-ID order
// so input ordering cannot affect the result.
func ComputeLayout(g *Graph) map[string]Position {
	n := len(g.Nodes)
	pos := make(map[string]Position, n)
	if n == 0 {
		return pos
	}

	// Stable node ordering for determinism.
	ids := make([]string, n)
	for i, nd := range g.Nodes {
		ids[i] = nd.ID
	}
	sort.Strings(ids)
	idx := make(map[string]int, n)
	for i, id := range ids {
		idx[id] = i
	}

	// Deterministic initial placement: a golden-angle (sunflower) spiral around
	// the centre. No PRNG — position is a pure function of the node's index.
	px := make([]float64, n)
	py := make([]float64, n)
	cx, cy := layoutWidth/2, layoutHeight/2
	maxR := math.Min(layoutWidth, layoutHeight)/2 - layoutPad
	const goldenAngle = 2.399963229728653 // radians
	for i := 0; i < n; i++ {
		r := math.Sqrt(float64(i)+0.5) / math.Sqrt(float64(n)) * maxR
		a := float64(i) * goldenAngle
		px[i] = cx + r*math.Cos(a)
		py[i] = cy + r*math.Sin(a)
	}

	// Edges in index space. Sorted so the attractive-force accumulation order is
	// fixed: floating-point addition is not associative, so a varying edge order
	// (the builder emits edges from maps) would otherwise drift positions and
	// break determinism.
	type pair struct{ a, b int }
	edges := make([]pair, 0, len(g.Edges))
	for _, e := range g.Edges {
		ai, ok1 := idx[e.Source]
		bi, ok2 := idx[e.Target]
		if ok1 && ok2 && ai != bi {
			edges = append(edges, pair{ai, bi})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].a != edges[j].a {
			return edges[i].a < edges[j].a
		}
		return edges[i].b < edges[j].b
	})

	k := math.Sqrt(layoutWidth * layoutHeight / float64(n)) // ideal edge length
	temp := layoutWidth / 10.0
	cool := temp / float64(layoutIters+1)
	dispX := make([]float64, n)
	dispY := make([]float64, n)

	for it := 0; it < layoutIters; it++ {
		for i := range dispX {
			dispX[i], dispY[i] = 0, 0
		}
		// Repulsion between every pair.
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				dx := px[i] - px[j]
				dy := py[i] - py[j]
				dist := math.Hypot(dx, dy)
				if dist < 0.01 {
					// Deterministic separation for coincident nodes.
					dx, dy, dist = 0.01, 0, 0.01
				}
				f := k * k / dist
				ux, uy := dx/dist, dy/dist
				dispX[i] += ux * f
				dispY[i] += uy * f
				dispX[j] -= ux * f
				dispY[j] -= uy * f
			}
		}
		// Attraction along edges.
		for _, e := range edges {
			dx := px[e.a] - px[e.b]
			dy := py[e.a] - py[e.b]
			dist := math.Hypot(dx, dy)
			if dist < 0.01 {
				dist = 0.01
			}
			f := dist * dist / k
			ux, uy := dx/dist, dy/dist
			dispX[e.a] -= ux * f
			dispY[e.a] -= uy * f
			dispX[e.b] += ux * f
			dispY[e.b] += uy * f
		}
		// Apply, capped by temperature, clamped to the viewport.
		for i := 0; i < n; i++ {
			d := math.Hypot(dispX[i], dispY[i])
			if d < 0.01 {
				continue
			}
			lim := math.Min(d, temp)
			px[i] += dispX[i] / d * lim
			py[i] += dispY[i] / d * lim
			px[i] = math.Max(layoutPad, math.Min(layoutWidth-layoutPad, px[i]))
			py[i] = math.Max(layoutPad, math.Min(layoutHeight-layoutPad, py[i]))
		}
		temp -= cool
	}

	for id, i := range idx {
		pos[id] = Position{X: round1(px[i]), Y: round1(py[i])}
	}
	return pos
}

// round1 rounds to one decimal so exported coordinates are stable and compact.
func round1(v float64) float64 { return math.Round(v*10) / 10 }
