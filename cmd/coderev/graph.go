package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/graph"
)

// defaultGraphDir is where the native code graph is written when neither the
// --output flag nor a [graph] output_dir in tool_config.toml is set. It is a
// coderev-owned dotdir — deliberately NOT graphify-out/, which belongs to the
// separate graphify tool.
const defaultGraphDir = ".coderev/graph"

var flagGraphOutput string

var cmdGraph = &cobra.Command{
	Use:   "graph [directory]",
	Short: "Build and export a native code graph",
	Long: `Builds a code graph from source files — nodes are files,
functions and types; edges are imports, calls and containment —
and writes coderev's native graph.json + graph.html to the output
directory (--output, else [graph] output_dir in tool_config.toml,
else ` + defaultGraphDir + `).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := "."
		if len(args) > 0 {
			target = args[0]
		}
		// Resolve output dir: --output flag > tool_config [graph] output_dir >
		// the default dotdir. Relative paths resolve against the target.
		dir := flagGraphOutput
		if dir == "" {
			cfgPath, _ := config.DiscoverToolConfig(target)
			if tc, err := config.LoadToolConfig(cfgPath); err == nil && tc.Graph.OutputDir != "" {
				dir = tc.Graph.OutputDir
			}
		}
		if dir == "" {
			dir = defaultGraphDir
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(target, dir)
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
	cmdGraph.Flags().StringVar(&flagGraphOutput, "output", "", "output directory (default: <target>/"+defaultGraphDir+")")
}
