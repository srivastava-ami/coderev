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
	fileFuncs, fileTypes := buildNodeIndex(g)
	fileImports := buildFileImports(g)
	dirFiles := buildDirFiles(fileFuncs, fileTypes)
	pkgs := buildPackages(dirFiles, fileImports, fileFuncs, fileTypes, root)
	flows := detectFlows(g, root)
	return pkgs, flows
}

func buildNodeIndex(g graphJSON) (map[string][]string, map[string][]string) {
	fileFuncs := map[string][]string{}
	fileTypes := map[string][]string{}
	for _, n := range g.Nodes {
		switch n.Kind {
		case "function":
			fileFuncs[n.SourceFile] = append(fileFuncs[n.SourceFile], n.Label)
		case "type":
			fileTypes[n.SourceFile] = append(fileTypes[n.SourceFile], n.Label)
		}
	}
	return fileFuncs, fileTypes
}

func buildFileImports(g graphJSON) map[string][]string {
	fileImports := map[string][]string{}
	for _, l := range g.Links {
		if l.Relation != "imports" {
			continue
		}
		sf, tf := l.Source, l.Target
		if idx := strings.LastIndex(sf, ":"); idx > 0 {
			sf = sf[:idx]
		}
		if idx := strings.LastIndex(tf, ":"); idx > 0 {
			tf = tf[:idx]
		}
		if sf != tf {
			fileImports[sf] = append(fileImports[sf], tf)
		}
	}
	return fileImports
}

func buildPackages(dirFiles, fileImports map[string][]string, fileFuncs, fileTypes map[string][]string, root string) []PackageInfo {
	var pkgs []PackageInfo
	for dir, files := range dirFiles {
		rel, _ := filepath.Rel(root, dir)
		importPath := filepath.ToSlash(rel)
		name := filepath.Base(dir)
		sort.Strings(files)
		fileSyms, allSyms := collectSymbols(files, fileFuncs, fileTypes, root)
		deps := collectDeps(files, fileImports, root, importPath)
		pkgs = append(pkgs, PackageInfo{
			ImportPath:      importPath,
			Name:            name,
			Layer:           nxsLayer(importPath),
			Deps:            deps,
			ExportedSymbols: allSyms,
			Files:           fileSyms,
		})
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ImportPath < pkgs[j].ImportPath })
	return pkgs
}

func collectSymbols(files []string, fileFuncs, fileTypes map[string][]string, root string) (map[string][]string, []string) {
	fileSyms := map[string][]string{}
	symSet := map[string]bool{}
	var allSyms []string
	for _, f := range files {
		syms := append(fileFuncs[f], fileTypes[f]...)
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
	return fileSyms, allSyms
}

func collectDeps(files []string, fileImports map[string][]string, root, importPath string) []string {
	depSet := map[string]bool{}
	for _, f := range files {
		for _, dep := range fileImports[f] {
			depRel, _ := filepath.Rel(root, dep)
			if depRel != importPath && !strings.HasPrefix(depRel, ".") {
				depSet[depRel] = true
			}
		}
	}
	var deps []string
	for d := range depSet {
		deps = append(deps, d)
	}
	sort.Strings(deps)
	return deps
}

type flowTracer struct {
	callers map[string][]string
	callees map[string][]string
	g       graphJSON
	root    string
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
	tracer := flowTracer{callers: callers, callees: callees, g: g, root: root}
	entryIDs := findEntryPoints(g)
	var flows []Flow
	for _, entry := range entryIDs {
		fname := flowName(entry)
		steps := traceFlow(entry, tracer)
		if len(steps) == 0 {
			continue
		}
		flows = append(flows, Flow{
			Name:        fname,
			Entry:       entry,
			Description: flowDescription(fname),
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

type flowState struct {
	tracer  flowTracer
	steps   *[]FlowStep
	visited map[string]bool
}

func traceFlow(entryID string, t flowTracer) []FlowStep {
	var steps []FlowStep
	fs := flowState{tracer: t, steps: &steps, visited: map[string]bool{}}
	fs.traceRecurse(entryID, 0)
	return steps
}

func (fs flowState) traceRecurse(nodeID string, depth int) {
	if depth > 8 || fs.visited[nodeID] {
		return
	}
	fs.visited[nodeID] = true
	var absFile, funcName string
	for _, n := range fs.tracer.g.Nodes {
		if n.ID == nodeID {
			absFile = n.SourceFile
			funcName = n.Label
			break
		}
	}
	if absFile == "" {
		return
	}
	relFile, _ := filepath.Rel(fs.tracer.root, absFile)
	relDir := filepath.Dir(relFile)
	layer := nxsLayer(relDir)
	label := componentLabel(PackageInfo{ImportPath: relDir})
	*fs.steps = append(*fs.steps, FlowStep{
		File:  relFile,
		Func:  funcName,
		Layer: layer,
		Label: label,
	})
	for _, callee := range fs.tracer.callees[nodeID] {
		fs.traceRecurse(callee, depth+1)
	}
}
