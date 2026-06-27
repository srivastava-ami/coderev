package graph

// ComputeMetrics calculates fan-in, fan-out, and a simple degree
// centrality for every node in the graph.  It must be called after the
// graph is fully built.
func ComputeMetrics(g *Graph) {
	for id := range g.FanIn {
		delete(g.FanIn, id)
	}
	for id := range g.FanOut {
		delete(g.FanOut, id)
	}
	for id := range g.Centrality {
		delete(g.Centrality, id)
	}

	// Initialise counts for every node.
	for _, n := range g.Nodes {
		g.FanIn[n.ID] = 0
		g.FanOut[n.ID] = 0
		g.Centrality[n.ID] = 0
	}

	// Count incoming and outgoing edges.
	for _, e := range g.Edges {
		g.FanOut[e.Source] = g.FanOut[e.Source] + 1
		g.FanIn[e.Target] = g.FanIn[e.Target] + 1
	}

	n := len(g.Nodes)
	if n <= 1 {
		return
	}
	maxDegree := float64(2 * (n - 1)) // worst-case complete directed graph

	for _, node := range g.Nodes {
		deg := float64(g.FanIn[node.ID] + g.FanOut[node.ID])
		if maxDegree > 0 {
			g.Centrality[node.ID] = deg / maxDegree
		}
	}
}
