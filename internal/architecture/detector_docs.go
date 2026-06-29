package architecture

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type archDocResult struct {
	files       []ArchDocFile
	primaryPath string
	primaryHTML string
	primaryText string
}

func scanMarkdownDocs(target string) archDocResult {
	var docFiles []ArchDocFile
	_ = analysis.WalkIgnoring(target, func(path string, d fs.DirEntry) error {
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		html := MarkdownToHTML(string(data))
		rel, _ := filepath.Rel(target, path)
		docFiles = append(docFiles, ArchDocFile{Path: rel, Name: d.Name(), HTML: html, Content: string(data)})
		return nil
	})
	var primaryPath, primaryHTML, primaryText string
	for _, candidate := range archDocCandidates {
		path := filepath.Join(target, candidate)
		data, err := os.ReadFile(path)
		if err == nil {
			primaryPath = path
			primaryText = string(data)
			primaryHTML = MarkdownToHTML(primaryText)
			break
		}
	}
	if primaryPath == "" && len(docFiles) > 0 {
		primaryPath = filepath.Join(target, docFiles[0].Path)
	}
	return archDocResult{files: docFiles, primaryPath: primaryPath, primaryHTML: primaryHTML, primaryText: primaryText}
}

func applyDocResult(s *Summary, dr archDocResult) {
	s.ArchDocFiles = dr.files
	if dr.primaryPath != "" {
		s.Source = "doc"
		s.DocFile = dr.primaryPath
		s.Text = dr.primaryText
		s.ArchDocHTML = dr.primaryHTML
	}
}
