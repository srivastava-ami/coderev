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
	"unicode"
)

// PackageInfo holds the deterministic architecture view of one Go package.
type PackageInfo struct {
	ImportPath      string   // relative to module root, e.g. "internal/analysis"
	Name            string   // Go package name
	Layer           string   // NXS layer: governance|orchestration|execution|persistence|surface
	DocSummary      string   // first sentence of the package doc comment
	Deps            []string // intra-module imports only, sorted
	ExportedSymbols []string // exported func/type names, sorted
	ExternalLibs    []string // external (non-stdlib, non-module) import paths, sorted
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
	for dir := range dirFiles {
		if p, ok := parsePackageDir(fset, root, dir, modPrefix); ok {
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

func parsePackageDir(fset *token.FileSet, root, dir, modPrefix string) (PackageInfo, bool) {
	pkgs, err := parser.ParseDir(fset, dir, onlyNonTest, parser.ParseComments)
	if err != nil || len(pkgs) == 0 {
		return PackageInfo{}, false
	}
	rel, _ := filepath.Rel(root, dir)
	importPath := filepath.ToSlash(rel)
	if importPath == "." {
		importPath = ""
	}
	for _, pkg := range pkgs {
		deps, extLibs := collectImports(pkg, modPrefix, importPath)
		return PackageInfo{
			ImportPath:      importPath,
			Name:            pkg.Name,
			Layer:           nxsLayer(importPath),
			DocSummary:      packageDocSummary(pkg),
			Deps:            deps,
			ExportedSymbols: collectExported(pkg),
			ExternalLibs:    extLibs,
		}, true
	}
	return PackageInfo{}, false
}

func onlyNonTest(fi os.FileInfo) bool {
	return !strings.HasSuffix(fi.Name(), "_test.go")
}

func collectImports(pkg *ast.Package, modPrefix, selfPath string) (intra, external []string) {
	intraSeen, extSeen := map[string]bool{}, map[string]bool{}
	for _, f := range pkg.Files {
		classifyFileImports(f, modPrefix, selfPath, intraSeen, extSeen)
	}
	return sortedKeys(intraSeen), sortedKeys(extSeen)
}

func classifyFileImports(f *ast.File, modPrefix, selfPath string, intra, ext map[string]bool) {
	for _, imp := range f.Imports {
		raw := strings.Trim(imp.Path.Value, `"`)
		if strings.HasPrefix(raw, modPrefix+"/") {
			rel := strings.TrimPrefix(raw, modPrefix+"/")
			if rel != selfPath {
				intra[rel] = true
			}
			continue
		}
		if isExternalImport(raw) {
			ext[raw] = true
		}
	}
}

func isExternalImport(path string) bool {
	first := strings.SplitN(path, "/", 2)[0]
	return strings.Contains(first, ".")
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func collectExported(pkg *ast.Package) []string {
	seen := map[string]bool{}
	for _, f := range pkg.Files {
		collectFileExported(f, seen)
	}
	return sortedKeys(seen)
}

func collectFileExported(f *ast.File, seen map[string]bool) {
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name != nil && isExported(d.Name.Name) {
				seen[d.Name.Name] = true
			}
		case *ast.GenDecl:
			collectGenDeclExported(d, seen)
		}
	}
}

func collectGenDeclExported(d *ast.GenDecl, seen map[string]bool) {
	for _, spec := range d.Specs {
		if ts, ok := spec.(*ast.TypeSpec); ok && isExported(ts.Name.Name) {
			seen[ts.Name.Name] = true
		}
	}
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
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
		if i := strings.IndexAny(text, ".\n"); i >= 0 {
			return strings.TrimSpace(text[:i+1])
		}
		return text
	}
	return ""
}

// nxsLayer maps a Go import path to an NXS architecture layer.
// Heuristic — teams can edit the generated architecture.toml to correct assignments.
func nxsLayer(importPath string) string {
	switch {
	case strings.HasPrefix(importPath, "cmd"):
		return "orchestration"
	case matchesAny(importPath, "internal/config", "internal/quality"):
		return "governance"
	case matchesAny(importPath, "internal/adapters", "internal/baseline",
		"internal/toolmgr", "internal/plugin", "internal/github", "internal/llm"):
		return "persistence"
	case matchesAny(importPath, "internal/report", "internal/output"):
		return "surface"
	default:
		return "execution"
	}
}

func matchesAny(path string, prefixes ...string) bool {
	for _, p := range prefixes {
		if path == p || strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
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
