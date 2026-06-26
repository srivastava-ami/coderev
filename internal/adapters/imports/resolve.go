package imports

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// tsExtensions are the candidate extensions tried (in order) when resolving a
// TS/JS import specifier that omits its extension.
var tsExtensions = []string{".ts", ".tsx", ".d.ts", ".js", ".jsx", ".mjs", ".cjs"}

// import-extraction regexes.
var (
	// `import x from 'spec'`, `export { y } from 'spec'`
	reFromImport = regexp.MustCompile(`(?:import|export)\b[^;'"]*?\bfrom\s*['"]([^'"]+)['"]`)
	// bare side-effect import: `import 'spec'`
	reBareImport = regexp.MustCompile(`(?m)^\s*import\s+['"]([^'"]+)['"]`)
	// `require('spec')` and dynamic `import('spec')`
	reCallImport = regexp.MustCompile(`(?:require|import)\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	// Go single import: `import "spec"` or `import alias "spec"`
	reGoSingle = regexp.MustCompile(`(?m)^\s*import\s+(?:[\w.]+\s+)?"([^"]+)"`)
	// Go grouped import block: `import ( ... )`
	reGoBlock = regexp.MustCompile(`(?s)import\s*\(\s*(.*?)\s*\)`)
	// individual quoted path inside a Go import block, possibly with alias.
	reGoBlockEntry = regexp.MustCompile(`(?m)^\s*(?:[\w.]+\s+)?"([^"]+)"`)
)

// resolver turns import specifiers into the set of internal node IDs they point
// at. It is built once per BuildGraph call and reads go.mod / tsconfig.json from
// the analysis target so package and alias imports resolve correctly.
type resolver struct {
	target    string
	index     map[string]*analysis.FileInfo // canonical path to file
	goModule  string                        // module path from go.mod (may be "")
	tsBaseURL string                        // absolute baseUrl from tsconfig
	tsPaths   map[string][]string           // tsconfig compilerOptions.paths
}

func newResolver(req analysis.RunRequest) *resolver {
	r := &resolver{
		target: req.Target,
		index:  make(map[string]*analysis.FileInfo, len(req.Files)),
	}
	for i := range req.Files {
		f := &req.Files[i]
		r.index[clean(f.Path)] = f
	}
	r.loadGoModule()
	r.loadTSConfig()
	return r
}

func clean(p string) string { return filepath.Clean(p) }

// extractImports returns the raw import specifiers found in a source file.
func extractImports(f analysis.FileInfo) []string {
	src := string(f.Content)
	switch f.Language {
	case analysis.LangGo:
		return extractGoImports(src)
	case analysis.LangTypeScript, analysis.LangJavaScript:
		return extractTSImports(src)
	default:
		return nil
	}
}

func extractTSImports(src string) []string {
	var specs []string
	for _, re := range []*regexp.Regexp{reFromImport, reBareImport, reCallImport} {
		for _, m := range re.FindAllStringSubmatch(src, -1) {
			specs = append(specs, m[1])
		}
	}
	return specs
}

func extractGoImports(src string) []string {
	var specs []string
	for _, m := range reGoSingle.FindAllStringSubmatch(src, -1) {
		specs = append(specs, m[1])
	}
	for _, block := range reGoBlock.FindAllStringSubmatch(src, -1) {
		for _, m := range reGoBlockEntry.FindAllStringSubmatch(block[1], -1) {
			specs = append(specs, m[1])
		}
	}
	return specs
}

// resolve maps a single import specifier from `fromPath` to the internal node
// IDs it references. ok is false for external/unresolvable imports (e.g. npm
// packages or Go stdlib), which are intentionally excluded from the graph.
func (r *resolver) resolve(fromPath, spec string, lang analysis.Language) (ids []string, ok bool) {
	switch lang {
	case analysis.LangGo:
		return r.resolveGo(spec)
	case analysis.LangTypeScript, analysis.LangJavaScript:
		return r.resolveTS(fromPath, spec)
	default:
		return nil, false
	}
}

// resolveTS handles relative imports, tsconfig path aliases, then gives up on
// bare package specifiers.
func (r *resolver) resolveTS(fromPath, spec string) ([]string, bool) {
	if strings.HasPrefix(spec, ".") {
		base := filepath.Dir(fromPath)
		if id, ok := r.resolveTSFile(filepath.Join(base, spec)); ok {
			return []string{id}, true
		}
		return nil, false
	}
	if id, ok := r.resolveAlias(spec); ok {
		return []string{id}, true
	}
	return nil, false
}

// resolveAlias matches a specifier against tsconfig compilerOptions.paths.
func (r *resolver) resolveAlias(spec string) (string, bool) {
	for pattern, targets := range r.tsPaths {
		matched, ok := matchAlias(pattern, spec)
		if !ok {
			continue
		}
		for _, t := range targets {
			candidate := strings.Replace(t, "*", matched, 1)
			full := candidate
			if r.tsBaseURL != "" {
				full = filepath.Join(r.tsBaseURL, candidate)
			}
			if id, ok := r.resolveTSFile(full); ok {
				return id, true
			}
		}
	}
	return "", false
}

// matchAlias matches `spec` against a tsconfig path pattern. For a wildcard
// pattern ("@app/*") it returns the captured tail; for an exact pattern it
// returns "".
func matchAlias(pattern, spec string) (string, bool) {
	if i := strings.Index(pattern, "*"); i >= 0 {
		prefix := pattern[:i]
		suffix := pattern[i+1:]
		if strings.HasPrefix(spec, prefix) && strings.HasSuffix(spec, suffix) {
			return spec[len(prefix) : len(spec)-len(suffix)], true
		}
		return "", false
	}
	return "", pattern == spec
}

// resolveTSFile tries a base path with every candidate extension plus an
// index.* file, returning the first match present in the file index.
func (r *resolver) resolveTSFile(base string) (string, bool) {
	base = clean(base)
	// Exact path (already has an extension).
	if _, ok := r.index[base]; ok {
		return base, true
	}
	for _, ext := range tsExtensions {
		if id := clean(base + ext); r.index[id] != nil {
			return id, true
		}
	}
	for _, ext := range tsExtensions {
		if id := clean(filepath.Join(base, "index"+ext)); r.index[id] != nil {
			return id, true
		}
	}
	return "", false
}

// resolveGo maps a Go import path to every indexed .go file in the matching
// package directory. External imports (stdlib, other modules) return false.
func (r *resolver) resolveGo(spec string) ([]string, bool) {
	if r.goModule == "" {
		return nil, false
	}
	if spec != r.goModule && !strings.HasPrefix(spec, r.goModule+"/") {
		return nil, false
	}
	sub := strings.TrimPrefix(strings.TrimPrefix(spec, r.goModule), "/")
	dir := clean(filepath.Join(r.target, sub))

	var ids []string
	for id, f := range r.index {
		if f.Language == analysis.LangGo && clean(filepath.Dir(id)) == dir {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, false
	}
	return ids, true
}

// loadGoModule reads the module path from <target>/go.mod, if present.
func (r *resolver) loadGoModule() {
	data, err := os.ReadFile(filepath.Join(r.target, "go.mod"))
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			r.goModule = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			return
		}
	}
}

// tsConfig is the minimal slice of tsconfig.json we need for alias resolution.
type tsConfig struct {
	CompilerOptions struct {
		BaseURL string              `json:"baseUrl"`
		Paths   map[string][]string `json:"paths"`
	} `json:"compilerOptions"`
}

// loadTSConfig reads <target>/tsconfig.json (tolerating // and /* */ comments)
// and records baseUrl + paths for alias resolution.
func (r *resolver) loadTSConfig() {
	path := filepath.Join(r.target, "tsconfig.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg tsConfig
	if err := json.Unmarshal(stripJSONComments(data), &cfg); err != nil {
		return
	}
	r.tsPaths = cfg.CompilerOptions.Paths
	if base := cfg.CompilerOptions.BaseURL; base != "" {
		r.tsBaseURL = clean(filepath.Join(r.target, base))
	} else {
		r.tsBaseURL = r.target
	}
}

// JSONC comment stripping lives in jsonc.go.
