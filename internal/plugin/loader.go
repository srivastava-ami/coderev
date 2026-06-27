package plugin

import (
	"io/fs"
	"os/exec"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type DiscoveredPlugin struct {
	Manifest Manifest
	ExecPath string
}

func DiscoverPlugins(dir string) ([]DiscoveredPlugin, error) {
	var plugins []DiscoveredPlugin
	err := analysis.WalkIgnoring(dir, func(path string, d fs.DirEntry) error {
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
