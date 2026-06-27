package analysis

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// WalkIgnoring walks target, skipping any directory or file excluded by the root
// .gitignore (and builtin noise dirs), and calls fn for each surviving file. It
// is the single ignore-aware walk: all file discovery routes through it, so the
// skip logic — and the .gitignore policy — lives in exactly one place.
func WalkIgnoring(target string, fn func(path string, d fs.DirEntry) error) error {
	ig := NewIgnorer(target)
	return filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if ig.SkipDir(path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if ig.SkipFile(path) {
			return nil
		}
		return fn(path, d)
	})
}

// CollectSourceFiles returns every recognised source file under target as a
// FileInfo, honouring .gitignore via WalkIgnoring. Shared by the scanner and the
// code graph.
func CollectSourceFiles(target string) ([]FileInfo, error) {
	var files []FileInfo
	err := WalkIgnoring(target, func(path string, _ fs.DirEntry) error {
		lang, ok := ExtToLanguage[filepath.Ext(path)]
		if !ok {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		files = append(files, FileInfo{
			Path:     path,
			Language: lang,
			Lines:    strings.Count(string(content), "\n") + 1,
			Content:  content,
		})
		return nil
	})
	return files, err
}
