package config

import (
	"os"
	"path/filepath"
)

const (
	defaultToolConfigFile = "tool_config.toml"
	defaultOutputFile     = "coderev-report.html"
	globalConfigDir       = "~/.config/coderev"
)

// DiscoverToolConfig walks the same search path for the tool adapter config.
func DiscoverToolConfig(target string) (string, bool) {
	return discoverFile(target, defaultToolConfigFile)
}

func discoverFile(target, name string) (string, bool) {
	candidates := []string{
		filepath.Join(target, name),
	}

	if cwd, err := os.Getwd(); err == nil && cwd != target {
		candidates = append(candidates, filepath.Join(cwd, name))
	}

	home, err := os.UserHomeDir()
	if err == nil {
		candidates = append(candidates, filepath.Join(home, ".config", "coderev", name))
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}
