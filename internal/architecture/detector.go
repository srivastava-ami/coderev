package architecture

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Summary is the architecture overview fed to the HTML report.
type Summary struct {
	Source      string // "doc" | "synthesised"
	DocFile     string // path to the arch doc if found
	Text        string // human-readable summary
	ProjectName string
	Nodes       []Node
	Edges       []Edge
}

type Node struct {
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Type  string   `json:"type"` // "app" | "lib" | "service" | "external"
	Tags  []string `json:"tags"`
}

type Edge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

// archDocCandidates lists filenames/patterns to look for, in priority order.
var archDocCandidates = []string{
	"architecture.md",
	"ARCHITECTURE.md",
	"arch.md",
	"docs/architecture.md",
	"docs/arch.md",
	"docs/one-pager.md",
	"docs/overview.md",
	"docs/README.md",
	"README.md",
}

// Detect looks for an architecture document in the target repo, then falls
// back to synthesising one from the NX project graph or directory structure.
func Detect(target string) Summary {
	for _, candidate := range archDocCandidates {
		path := filepath.Join(target, candidate)
		data, err := os.ReadFile(path)
		if err == nil {
			return Summary{Source: "doc", DocFile: path, Text: string(data), Nodes: []Node{}, Edges: []Edge{}}
		}
	}
	if s, ok := fromNXWorkspace(target); ok {
		return s
	}
	return synthesiseFromDirs(target)
}

// nxProject is the schema of a project.json file in an NX workspace.
type nxProject struct {
	Name         string   `json:"name"`
	ProjectType  string   `json:"projectType"` // "application" | "library"
	Tags         []string `json:"tags"`
	ImplicitDeps []string `json:"implicitDependencies"`
}

func readNXProject(path string) (nxProject, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nxProject{}, false
	}
	var proj nxProject
	if err := json.Unmarshal(data, &proj); err != nil || proj.Name == "" {
		return nxProject{}, false
	}
	return proj, true
}

func nxNodeType(proj nxProject) string {
	if proj.ProjectType == "application" {
		return "app"
	}
	return "lib"
}

// fromNXWorkspace reads project.json files and nx.json to build a graph.
func fromNXWorkspace(root string) (Summary, bool) {
	if _, err := os.Stat(filepath.Join(root, "nx.json")); err != nil {
		return Summary{}, false
	}

	var nodes []Node
	var edges []Edge

	_ = analysis.WalkIgnoring(root, func(path string, d fs.DirEntry) error {
		if d.Name() != "project.json" {
			return nil
		}
		proj, ok := readNXProject(path)
		if !ok {
			return nil
		}
		nodes = append(nodes, Node{ID: proj.Name, Label: proj.Name, Type: nxNodeType(proj), Tags: proj.Tags})
		for _, dep := range proj.ImplicitDeps {
			edges = append(edges, Edge{From: proj.Name, To: dep})
		}
		return nil
	})

	if len(nodes) == 0 {
		return Summary{}, false
	}
	return Summary{Source: "synthesised", Text: synthesiseNXText(nodes, edges), Nodes: nodes, Edges: edges}, true
}

// synthesiseFromDirs creates a minimal arch summary from the directory tree.
func synthesiseFromDirs(root string) Summary {
	entries, err := os.ReadDir(root)
	if err != nil {
		return Summary{Source: "synthesised", Text: "Unable to read project directory."}
	}

	var nodes []Node
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || e.Name() == "node_modules" {
			continue
		}
		nodes = append(nodes, Node{ID: e.Name(), Label: e.Name(), Type: guessType(e.Name())})
	}

	return Summary{Source: "synthesised", Text: synthesiseDirText(root, nodes), Nodes: nodes, Edges: []Edge{}}
}

func guessType(dirName string) string {
	switch dirName {
	case "apps", "app":
		return "app"
	case "libs", "lib", "packages", "pkg":
		return "lib"
	case "services", "service":
		return "service"
	default:
		return "lib"
	}
}

func tagsString(n Node) string {
	if len(n.Tags) == 0 {
		return ""
	}
	return " `" + strings.Join(n.Tags, "`, `") + "`"
}

func synthesiseNXText(nodes []Node, edges []Edge) string {
	var sb strings.Builder
	sb.WriteString("## Architecture Overview\n\n")
	sb.WriteString("*Synthesised from NX workspace configuration — no architecture document found.*\n\n")
	sb.WriteString("### Applications\n\n")
	for _, n := range filter(nodes, "app") {
		sb.WriteString("- **" + n.Label + "**" + tagsString(n) + "\n")
	}
	sb.WriteString("\n### Libraries\n\n")
	for _, n := range filter(nodes, "lib") {
		sb.WriteString("- **" + n.Label + "**" + tagsString(n) + "\n")
	}
	if len(edges) > 0 {
		sb.WriteString("\n### Dependencies\n\n")
		for _, e := range edges {
			sb.WriteString("- " + e.From + " → " + e.To + "\n")
		}
	}
	sb.WriteString("\n> Add a `docs/architecture.md` to replace this synthesised summary.")
	return sb.String()
}

func synthesiseDirText(root string, nodes []Node) string {
	var sb strings.Builder
	sb.WriteString("## Architecture Overview\n\n")
	sb.WriteString("*Synthesised from directory structure — no architecture document or NX workspace found.*\n\n")
	sb.WriteString("### Top-level modules\n\n")
	for _, n := range nodes {
		sb.WriteString("- **" + n.Label + "**\n")
	}
	sb.WriteString("\n> Add a `docs/architecture.md` to replace this synthesised summary.")
	return sb.String()
}

func filter(nodes []Node, t string) []Node {
	var out []Node
	for _, n := range nodes {
		if n.Type == t {
			out = append(out, n)
		}
	}
	return out
}

// DetectWithGraph is like Detect but uses graph.json (from graphJSONPath) to derive
// file-level architecture nodes and call edges when no architecture doc is found.
func DetectWithGraph(target, graphJSONPath string) Summary {
	for _, candidate := range archDocCandidates {
		path := filepath.Join(target, candidate)
		data, err := os.ReadFile(path)
		if err == nil {
			return Summary{Source: "doc", DocFile: path, Text: string(data), Nodes: []Node{}, Edges: []Edge{}}
		}
	}
	if s, ok := fromNXWorkspace(target); ok {
		return s
	}
	if graphJSONPath != "" {
		data, err := os.ReadFile(graphJSONPath)
		if err == nil {
			if s, ok := fromGraphJSON(data); ok {
				return s
			}
		}
	}
	return synthesiseFromDirs(target)
}

