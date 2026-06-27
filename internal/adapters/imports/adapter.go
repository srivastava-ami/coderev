// Package imports is a native (zero-dependency) ToolAdapter that builds an
// import-dependency graph over TS/JS/Go sources and reports circular
// dependencies via Tarjan's strongly-connected-components algorithm. It is the
// default provider for file_structure.circular_deps and nx_conventions.boundaries,
// replacing the node/npm-bound `madge` adapter.
//
// The graph it produces (see BuildGraph / Graph) is deliberately reusable: the
// v1.3.0 code graph builds on the same nodes-and-edges structure.
package imports

import (
	"context"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

const adapterName = "imports"

// Adapter implements analysis.ToolAdapter using only the Go standard library.
type Adapter struct{}

// New returns a ready-to-use native imports adapter.
func New() *Adapter { return &Adapter{} }

func (a *Adapter) Name() string { return adapterName }

// IsAvailable is always true: the adapter is pure Go with no external binaries.
func (a *Adapter) IsAvailable() bool { return true }

func (a *Adapter) Capabilities() []string {
	return []string{"file_structure.circular_deps", "nx_conventions.boundaries"}
}

// BuildGraph constructs the import dependency graph for the given run. It is the
// exported, reusable entry point: callers (this adapter, and the forthcoming
// v1.3.0 code graph) get the full nodes+edges structure without re-parsing.
//
//	func BuildGraph(req analysis.RunRequest) *Graph
func BuildGraph(req analysis.RunRequest) *Graph {
	b := NewBuilder(req.Target)
	for _, f := range req.Files {
		b.Add(f)
	}
	return b.Build()
}

// Run builds the import graph and emits one blocker finding per detected cycle.
func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	g := BuildGraph(req)
	return a.AnalyzeGraph(g), nil
}

// AnalyzeGraph performs cycle detection on an already-built Graph and returns
// one blocker finding per circular dependency. This allows callers to build the
// graph incrementally (via Builder) and analyse it separately.
func (a *Adapter) AnalyzeGraph(g *Graph) []analysis.Finding {
	cycles := g.Cycles()

	findings := make([]analysis.Finding, 0, len(cycles))
	for _, cycle := range cycles {
		findings = append(findings, a.cycleFinding(g, cycle))
	}
	return findings
}

// cycleFinding renders one SCC as a circular-dependency finding using the
// original (uncleaned) file paths for readability.
func (a *Adapter) cycleFinding(g *Graph, cycle []string) analysis.Finding {
	paths := make([]string, 0, len(cycle)+1)
	for _, id := range cycle {
		paths = append(paths, displayPath(g, id))
	}
	// Close the loop so the cycle reads back to its starting node.
	paths = append(paths, paths[0])

	return analysis.Finding{
		Rule:        "file_structure.circular_deps",
		Pillar:      "file_structure",
		Severity:    analysis.SeverityBlocker,
		File:        displayPath(g, cycle[0]),
		Message:     "circular dependency: " + strings.Join(paths, " → "),
		Remediation: "Break the cycle: introduce an abstraction/interface, invert a dependency, or move shared code to a leaf module.",
		Source:      adapterName,
	}
}

func displayPath(g *Graph, id string) string {
	if n := g.Nodes[id]; n != nil {
		return n.Path
	}
	return id
}
