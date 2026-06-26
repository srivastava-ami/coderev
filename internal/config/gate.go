package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// Default gate thresholds: the maximum number of findings of each severity the
// gate tolerates before failing. Blockers always fail the gate (limit 0).
const (
	defaultMaxMajors     = 5  // max major findings tolerated before the gate fails
	defaultMaxAdvisories = 10 // max advisory findings tolerated before the gate fails
	defaultMaxTotal      = 20 // max total findings tolerated before the gate fails
)

func DefaultGateConfig() analysis.GateConfig {
	return analysis.GateConfig{
		Blockers:   0,
		Majors:     defaultMaxMajors,
		Advisories: defaultMaxAdvisories,
		Total:      defaultMaxTotal,
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
