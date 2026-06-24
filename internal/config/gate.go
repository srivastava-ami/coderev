package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func DefaultGateConfig() analysis.GateConfig {
	return analysis.GateConfig{
		Blockers:   0,
		Majors:     5,
		Advisories: 10,
		Total:      20,
	}
}

func LoadGate(path string) (*analysis.GateConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading gate config: %w", err)
	}
	var gc analysis.GateConfig
	if _, err := toml.Decode(string(data), &gc); err != nil {
		return nil, fmt.Errorf("parsing gate config: %w", err)
	}
	return &gc, nil
}
