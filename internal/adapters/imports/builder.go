package imports

import "github.com/srivastava-ami/coderev/internal/analysis"

// fileData is the lightweight per-file record kept by Builder after Add
// extracts import specifiers from the file's content. Content is NOT retained.
type fileData struct {
	path        string
	language    analysis.Language
	importSpecs []string
}

// Builder incrementally accumulates per-file import data and produces an import
// dependency Graph on Build(). Files are added one at a time via Add(); each
// Add parses the file's import specifiers and discards the raw content, so the
// builder never holds the full content of every file simultaneously.
type Builder struct {
	target string
	files  []fileData
}

// NewBuilder returns a builder scoped to the given analysis target directory.
func NewBuilder(target string) *Builder {
	return &Builder{target: target}
}

// Add extracts import specifiers from f and stores only the lightweight
// metadata needed for Build. f.Content is referenced during Add and NOT
// retained after this method returns.
func (b *Builder) Add(f analysis.FileInfo) {
	b.files = append(b.files, fileData{
		path:        f.Path,
		language:    f.Language,
		importSpecs: extractImports(f),
	})
}

// Build resolves all accumulated import specifiers to graph edges and returns
// the complete import-dependency Graph.
func (b *Builder) Build() *Graph {
	r := newResolver(b.target, b.files)
	g := NewGraph()
	for _, f := range b.files {
		g.AddNode(clean(f.path), f.path, f.language)
	}
	for _, f := range b.files {
		from := clean(f.path)
		for _, spec := range f.importSpecs {
			targets, ok := r.resolve(f.path, spec, f.language)
			if !ok {
				continue
			}
			for _, to := range targets {
				g.AddEdge(from, to)
			}
		}
	}
	return g
}
