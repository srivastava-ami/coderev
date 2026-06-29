package config

import (
	"os"
	"path/filepath"
)

const (
	coderevDir            = ".coderev"
	defaultToolConfigFile = "tool_config.toml"
	globalConfigDir       = "~/.config/coderev"
)

// DiscoverToolConfig looks for tool_config.toml in .coderev/ first (preferred),
// then the target root (backward compat), then the global config dir.
func DiscoverToolConfig(target string) (string, bool) {
	candidates := []string{
		filepath.Join(target, coderevDir, defaultToolConfigFile),
		filepath.Join(target, defaultToolConfigFile),
	}
	if cwd, err := os.Getwd(); err == nil && cwd != target {
		candidates = append(candidates,
			filepath.Join(cwd, coderevDir, defaultToolConfigFile),
			filepath.Join(cwd, defaultToolConfigFile),
		)
	}
	home, _ := os.UserHomeDir()
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".config", "coderev", defaultToolConfigFile))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return filepath.Join(target, coderevDir, defaultToolConfigFile), false
}
