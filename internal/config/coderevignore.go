package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"
)

//go:embed default_coderevignore.txt
var defaultCoderevIgnoreTxt string

// IgnoreList holds compiled patterns for filtering LLM review context.
type IgnoreList struct {
	patterns []string
}

// LoadIgnoreList reads the ignore file at path. Returns an empty list (no error)
// when the file does not exist yet.
func LoadIgnoreList(path string) (IgnoreList, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return IgnoreList{}, nil
		}
		return IgnoreList{}, err
	}
	return IgnoreList{patterns: parseIgnoreList(string(data))}, nil
}

// WriteDefaultIgnoreList writes embedded defaults to path. No-ops if file exists.
func WriteDefaultIgnoreList(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(defaultCoderevIgnoreTxt), 0o644)
}

// Matches reports whether path is excluded by any pattern in the list.
func (il IgnoreList) Matches(path string) bool {
	for _, p := range il.patterns {
		if matchesPattern(p, path) {
			return true
		}
	}
	return false
}

func parseIgnoreList(content string) []string {
	var patterns []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

func matchesPattern(pattern, path string) bool {
	base := filepath.Base(path)
	if m, _ := filepath.Match(pattern, base); m {
		return true
	}
	if m, _ := filepath.Match(pattern, filepath.ToSlash(path)); m {
		return true
	}
	if strings.HasSuffix(pattern, "/") {
		dir := strings.TrimSuffix(pattern, "/")
		for _, part := range strings.Split(filepath.ToSlash(path), "/") {
			if part == dir {
				return true
			}
		}
	}
	return false
}
