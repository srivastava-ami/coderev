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

var skipPluginDirs = map[string]bool{".git": true, "node_modules": true, "target": true, "__pycache__": true}

func DiscoverPlugins(dir string) ([]DiscoveredPlugin, error) {
	var plugins []DiscoveredPlugin
	err := filepath.WalkDir(dir, walkPluginFn(&plugins))
	return plugins, err
}

func walkPluginFn(plugins *[]DiscoveredPlugin) func(string, os.DirEntry, error) error {
	return func(path string, d os.DirEntry, err error) error {
		if err != nil || (d.IsDir() && skipPluginDirs[d.Name()]) {
			return filepath.SkipDir
		}
		if d.IsDir() {
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
		*plugins = append(*plugins, DiscoveredPlugin{Manifest: *m, ExecPath: execPath})
		return nil
	}
}
