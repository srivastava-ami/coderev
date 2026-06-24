package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Load parses a standards TOML file into an analysis.Standards struct.
func Load(path string) (analysis.Standards, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return analysis.Standards{}, fmt.Errorf("reading standards file: %w", err)
	}
	return loadTOML(data)
}

func loadTOML(data []byte) (analysis.Standards, error) {
	var s analysis.Standards
	if _, err := toml.Decode(string(data), &s); err != nil {
		return analysis.Standards{}, fmt.Errorf("parsing standards TOML: %w", err)
	}
	return s, nil
}
