package config

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// defaultToolConfigTOML is the built-in default tool configuration. Keeping the
// defaults (including every URL) in embedded TOML rather than Go literals means
// coderev's own source carries no hardcoded URLs to externalise.
//
//go:embed default_tool_config.toml
var defaultToolConfigTOML string

// LoadToolConfig parses a tool_config.toml. If path is empty, the built-in
// defaults (embedded default_tool_config.toml) are returned.
func LoadToolConfig(path string) (analysis.ToolConfig, error) {
	if path == "" {
		return defaultToolConfig()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return analysis.ToolConfig{}, fmt.Errorf("reading tool config: %w", err)
	}
	var tc analysis.ToolConfig
	if _, err := toml.Decode(string(data), &tc); err != nil {
		return analysis.ToolConfig{}, fmt.Errorf("parsing tool config TOML: %w", err)
	}
	return tc, nil
}

func defaultToolConfig() (analysis.ToolConfig, error) {
	var tc analysis.ToolConfig
	if _, err := toml.Decode(defaultToolConfigTOML, &tc); err != nil {
		return tc, fmt.Errorf("parsing embedded default tool config: %w", err)
	}
	return tc, nil
}
