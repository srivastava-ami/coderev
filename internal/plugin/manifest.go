package plugin

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Manifest struct {
	Name         string   `toml:"name"`
	Version      string   `toml:"version"`
	Description  string   `toml:"description"`
	Author       string   `toml:"author"`
	Repository   string   `toml:"repository"`
	Binary       string   `toml:"binary"`
	Capabilities []string `toml:"capabilities"`
	Languages    []string `toml:"languages"`
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading plugin manifest: %w", err)
	}
	var m Manifest
	if _, err := toml.Decode(string(data), &m); err != nil {
		return nil, fmt.Errorf("parsing plugin manifest: %w", err)
	}
	if m.Name == "" {
		return nil, fmt.Errorf("plugin manifest missing required field: name")
	}
	if m.Binary == "" {
		return nil, fmt.Errorf("plugin manifest missing required field: binary")
	}
	return &m, nil
}
