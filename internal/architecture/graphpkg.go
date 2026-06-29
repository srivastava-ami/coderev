package architecture

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

type graphJSON struct {
	Nodes []graphJSONNode `json:"nodes"`
	Links []graphJSONLink `json:"links"`
}

type graphJSONNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	SourceFile string `json:"source_file"`
}

type graphJSONLink struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Relation string `json:"relation"`
}

// FlowStep describes one step in a use-case flow across layers.
type FlowStep struct {
	File       string `json:"file"`
	Func       string `json:"func"`
	Layer      string `json:"layer"`
	Label      string `json:"label"`
	Subsequent string `json:"subsequent"`
}

// Flow describes one traced use-case from entry point through layers.
type Flow struct {
	Name        string     `json:"name"`
	Entry       string     `json:"entry"`
	Description string     `json:"description"`
	Steps       []FlowStep `json:"steps"`
}

// PackagesFromGraph reads graph.json and builds per-package, per-file info.
func PackagesFromGraph(data []byte, root string) ([]PackageInfo, []Flow) {
	var g graphJSON
	if err := json.Unmarshal(data, &g); err != nil || len(g.Nodes) == 0 {
		return nil, nil
	}

	fileFuncs := map[string][]string{}
	fileTypes := map[string][]string{}
	funcID := map[string]string{}

	for _, n := range g.Nodes {
		switch n.Kind {
		case "function":
			fileFuncs[n.SourceFile] = append(fileFuncs[n.SourceFile], n.Label)
			funcID[n.ID] = n.SourceFile
		case "type":
			fileTypes[n.SourceFile] = append(fileTypes[n.SourceFile], n.Label)
		}
	}

	fileImports := map[string][]string{}
	for _, l := range g.Links {
		if l.Relation == "imports" {
			sf := l.Source
			if idx := strings.LastIndex(sf, ":"); idx > 0 {
				sf = sf[:idx]
			}
			tf := l.Target
			if idx := strings.LastIndex(tf, ":"); idx > 0 {
				tf = tf[:idx]
			}
			if sf != tf {
				fileImports[sf] = append(fileImports[sf], tf)
			}
		}
	}

	dirFiles := map[string][]string{}
	for f := range fileFuncs {
		dir := filepath.Dir(f)
		dirFiles[dir] = append(dirFiles[dir], f)
	}
	for f := range fileTypes {
		dir := filepath.Dir(f)
		found := false
		for _, existing := range dirFiles[dir] {
			if existing == f {
				found = true
				break
			}
		}
		if !found {
			dirFiles[dir] = append(dirFiles[dir], f)
		}
	}

	var pkgs []PackageInfo
	for dir, files := range dirFiles {
		rel, _ := filepath.Rel(root, dir)
		importPath := filepath.ToSlash(rel)

		var allSyms []string
		symSet := map[string]bool{}
		fileSyms := map[string][]string{}

		sort.Strings(files)
		for _, f := range files {
			syms := fileFuncs[f]
			syms = append(syms, fileTypes[f]...)
			sort.Strings(syms)

			relFile, _ := filepath.Rel(root, f)
			fileSyms[relFile] = syms

			for _, s := range syms {
				if !symSet[s] {
					symSet[s] = true
					allSyms = append(allSyms, s)
				}
			}
		}
		sort.Strings(allSyms)

		depSet := map[string]bool{}
		var deps []string
		for _, f := range files {
			for _, dep := range fileImports[f] {
				depRel, _ := filepath.Rel(root, dep)
				if depRel != importPath && !strings.HasPrefix(depRel, ".") {
					depSet[depRel] = true
				}
			}
		}
		for d := range depSet {
			deps = append(deps, d)
		}
		sort.Strings(deps)

		name := filepath.Base(dir)
		pkg := PackageInfo{
			ImportPath:      importPath,
			Name:            name,
			Layer:           nxsLayer(importPath),
			Deps:            deps,
			ExportedSymbols: allSyms,
			Files:           fileSyms,
		}
		pkgs = append(pkgs, pkg)
	}

	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ImportPath < pkgs[j].ImportPath })

	flows := detectFlows(g, root)
	return pkgs, flows
}

func detectFlows(g graphJSON, root string) []Flow {
	callers := map[string][]string{}
	callees := map[string][]string{}
	for _, l := range g.Links {
		if l.Relation == "calls" {
			callees[l.Source] = append(callees[l.Source], l.Target)
			callers[l.Target] = append(callers[l.Target], l.Source)
		}
	}

	entryIDs := findEntryPoints(g)
	var flows []Flow
	for _, entry := range entryIDs {
		flowName := flowName(entry)
		steps := traceFlow(entry, callers, callees, g, root)
		if len(steps) == 0 {
			continue
		}
		flows = append(flows, Flow{
			Name:        flowName,
			Entry:       entry,
			Description: flowDescription(flowName),
			Steps:       steps,
		})
	}
	return flows
}

func findEntryPoints(g graphJSON) []string {
	var entries []string
	callersOf := map[string]int{}
	for _, l := range g.Links {
		if l.Relation == "calls" {
			callersOf[l.Target]++
		}
	}
	for _, n := range g.Nodes {
		if n.Kind != "function" {
			continue
		}
		if callersOf[n.ID] == 0 && isEntryCandidate(n.Label) {
			entries = append(entries, n.ID)
		}
	}
	sort.Strings(entries)
	return entries
}

func isEntryCandidate(name string) bool {
	entryNames := []string{"main", "run", "runGraph", "runConfig", "runAsk", "stdRun",
		"runReview", "runFullGraphReview", "cmdSetup", "cmdInstallHooks", "cmdInstallDeps",
		"cmdPlugin", "cmdGraph", "cmdConfig", "cmdAsk"}
	for _, e := range entryNames {
		if name == e {
			return true
		}
	}
	return false
}

func flowName(entryID string) string {
	parts := strings.Split(entryID, ":")
	name := parts[len(parts)-1]
	return name
}

func flowDescription(name string) string {
	switch name {
	case "main":
		return "CLI bootstrap and subcommand dispatch"
	case "stdRun":
		return "Full code scan: analysis → graph → architecture → report"
	case "runGraph":
		return "Code graph construction from tree-sitter ASTs"
	case "runReview":
		return "Diff-anchored LLM review with graph neighborhood context"
	case "runFullGraphReview":
		return "Full-codebase LLM review using graph context"
	default:
		return ""
	}
}

func traceFlow(entryID string, callers, callees map[string][]string, g graphJSON, root string) []FlowStep {
	visited := map[string]bool{}
	var steps []FlowStep
	traceRecurse(entryID, callers, callees, g, &steps, visited, root, 0)
	return steps
}

func traceRecurse(nodeID string, callers, callees map[string][]string, g graphJSON, steps *[]FlowStep, visited map[string]bool, root string, depth int) {
	if depth > 8 || visited[nodeID] {
		return
	}
	visited[nodeID] = true

	var absFile, funcName string
	for _, n := range g.Nodes {
		if n.ID == nodeID {
			absFile = n.SourceFile
			funcName = n.Label
			break
		}
	}
	if absFile == "" {
		return
	}

	relFile, _ := filepath.Rel(root, absFile)
	relDir := filepath.Dir(relFile)
	layer := nxsLayer(relDir)
	label := componentLabel(PackageInfo{ImportPath: relDir})

	*steps = append(*steps, FlowStep{
		File:  relFile,
		Func:  funcName,
		Layer: layer,
		Label: label,
	})

	for _, callee := range callees[nodeID] {
		traceRecurse(callee, callers, callees, g, steps, visited, root, depth+1)
	}
}
