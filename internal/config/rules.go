package config

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed rules
var RulesFS embed.FS

// ListPatternRules returns a list of all available TOML rule files.
func ListPatternRules() ([]string, error) {
	var files []string
	err := fs.WalkDir(RulesFS, "rules", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".toml") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
