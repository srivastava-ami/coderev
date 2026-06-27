package graph

// NodeKind represents the classification of a code graph node.
type NodeKind string

const (
	KindFile     NodeKind = "file"
	KindFunction NodeKind = "function"
	KindType     NodeKind = "type"
)

// Node is a single vertex in the code graph.
type Node struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Kind       NodeKind `json:"kind"`
	SourceFile string   `json:"source_file"`
}

// Edge is a directed relationship between two nodes.
type Edge struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

// Graph is the full code graph with pre-computed metrics.
type Graph struct {
	Nodes []Node
	Edges []Edge

	FanIn      map[string]int
	FanOut     map[string]int
	Centrality map[string]float64
}
