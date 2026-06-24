package plugin

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DiscoveredPlugin struct {
	Manifest Manifest
	ExecPath string
}

func DiscoverPlugins(dir string) ([]DiscoveredPlugin, error) {
	var plugins []DiscoveredPlugin
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			skipDirs := []string{".git", "node_modules", "target", "__pycache__"}
			for _, skip := range skipDirs {
				if d.Name() == skip {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "-plugin.toml") {
			return nil
		}
		m, err := LoadManifest(path)
		if err != nil {
			return nil
		}
		execPath, err := exec.LookPath(m.Binary)
		if err != nil {
			return nil
		}
		plugins = append(plugins, DiscoveredPlugin{Manifest: *m, ExecPath: execPath})
		return nil
	})
	return plugins, err
}
