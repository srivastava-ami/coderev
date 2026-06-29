package architecture

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PackageInfo holds the deterministic architecture view of one Go package.
type PackageInfo struct {
	ImportPath string   // relative to module root, e.g. "internal/analysis"
	Name       string   // Go package name
	Layer      string   // "cmd" | "internal" | "pkg" | "root"
	DocSummary string   // first sentence of the package doc comment
	Deps       []string // intra-module imports only, sorted
}

// ScanGoPackages walks root and returns one PackageInfo per Go package.
func ScanGoPackages(root string) []PackageInfo {
	modPrefix := readModulePrefix(root)
	if modPrefix == "" {
		return nil
	}
	dirFiles := collectGoFiles(root)
	fset := token.NewFileSet()
	var pkgs []PackageInfo
	for dir, files := range dirFiles {
		if p, ok := parsePackageDir(fset, root, dir, files, modPrefix); ok {
			pkgs = append(pkgs, p)
		}
	}
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].ImportPath < pkgs[j].ImportPath })
	return pkgs
}

func collectGoFiles(root string) map[string][]string {
	dirFiles := map[string][]string{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return skipHidden(d)
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		dir := filepath.Dir(path)
		dirFiles[dir] = append(dirFiles[dir], path)
		return nil
	})
	return dirFiles
}

func skipHidden(d fs.DirEntry) error {
	if d != nil && d.IsDir() && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor") {
		return filepath.SkipDir
	}
	return nil
}

func parsePackageDir(fset *token.FileSet, root, dir string, files []string, modPrefix string) (PackageInfo, bool) {
	pkgs, err := parser.ParseDir(fset, dir, onlyNonTest, parser.ParseComments|parser.ImportsOnly)
	if err != nil || len(pkgs) == 0 {
		return PackageInfo{}, false
	}
	rel, _ := filepath.Rel(root, dir)
	importPath := filepath.ToSlash(rel)
	if importPath == "." {
		importPath = ""
	}
	for _, pkg := range pkgs {
		if pkg.Name == "main" && importPath != "cmd/coderev" && strings.Contains(importPath, "cmd/") {
			// skip subsidiary mains (none expected but be safe)
		}
		deps := collectDeps(pkg, modPrefix, importPath)
		return PackageInfo{
			ImportPath: importPath,
			Name:       pkg.Name,
			Layer:      classifyLayer(importPath),
			DocSummary: packageDocSummary(pkg),
			Deps:       deps,
		}, true
	}
	return PackageInfo{}, false
}

func onlyNonTest(fi os.FileInfo) bool {
	return !strings.HasSuffix(fi.Name(), "_test.go")
}

func collectDeps(pkg *ast.Package, modPrefix, selfPath string) []string {
	seen := map[string]bool{}
	for _, f := range pkg.Files {
		for _, imp := range f.Imports {
			raw := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(raw, modPrefix+"/") {
				continue
			}
			rel := strings.TrimPrefix(raw, modPrefix+"/")
			if rel != selfPath && !seen[rel] {
				seen[rel] = true
			}
		}
	}
	var deps []string
	for d := range seen {
		deps = append(deps, d)
	}
	sort.Strings(deps)
	return deps
}

func packageDocSummary(pkg *ast.Package) string {
	for _, f := range pkg.Files {
		if f.Doc == nil {
			continue
		}
		text := strings.TrimSpace(f.Doc.Text())
		if text == "" {
			continue
		}
		// Return up to the first sentence (period or newline).
		if i := strings.IndexAny(text, ".\n"); i >= 0 {
			return strings.TrimSpace(text[:i+1])
		}
		return text
	}
	return ""
}

func classifyLayer(importPath string) string {
	switch {
	case strings.HasPrefix(importPath, "cmd"):
		return "cmd"
	case strings.HasPrefix(importPath, "internal"):
		return "internal"
	case strings.HasPrefix(importPath, "pkg"):
		return "pkg"
	default:
		return "root"
	}
}

func readModulePrefix(root string) string {
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
