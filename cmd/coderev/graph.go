package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/graph"
)

var flagGraphOutput string

var cmdGraph = &cobra.Command{
	Use:   "graph [directory]",
	Short: "Build and export a native code graph",
	Long: `Builds a code graph from source files — nodes are files,
functions and types; edges are imports, calls and containment —
and writes graphify-compatible output to the specified directory
(default: graphify-out/).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		dir := flagGraphOutput
		if dir == "" {
			dir = filepath.Join(target, "graphify-out")
		}
		fmt.Fprintf(os.Stderr, "building code graph for %s ...\n", target)

		g, err := graph.Build(target)
		if err != nil {
			return fmt.Errorf("build graph: %w", err)
		}

		graph.ComputeMetrics(g)
		fmt.Fprintf(os.Stderr, "  nodes: %d  edges: %d\n", len(g.Nodes), len(g.Edges))

		if err := graph.ExportAll(g, dir); err != nil {
			return fmt.Errorf("export graph: %w", err)
		}
		fmt.Fprintf(os.Stderr, "  wrote %s/graph.json + graph.html\n", dir)
		return nil
	},
}

func init() {
	cmdGraph.Flags().StringVar(&flagGraphOutput, "output", "", "output directory (default: <target>/graphify-out)")
}
