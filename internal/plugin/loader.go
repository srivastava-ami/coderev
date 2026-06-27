package plugin

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type DiscoveredPlugin struct {
	Manifest Manifest
	ExecPath string
}

var skipPluginDirs = map[string]bool{".git": true, "node_modules": true, "target": true, "__pycache__": true}

func DiscoverPlugins(dir string) ([]DiscoveredPlugin, error) {
	var plugins []DiscoveredPlugin
	err := filepath.WalkDir(dir, walkPluginFn(&plugins, analysis.NewIgnorer(dir)))
	return plugins, err
}

func walkPluginFn(plugins *[]DiscoveredPlugin, ig *analysis.Ignorer) func(string, os.DirEntry, error) error {
	return func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipPluginDirs[d.Name()] || ig.SkipDir(path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if ig.SkipFile(path) {
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
