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

// LoadToolConfig parses a tool_config.toml overlaid on top of the built-in
// defaults. If path is empty, only the defaults are returned. This means a
// partial tool_config.toml (e.g. only [llm]) keeps all adapter defaults active.
func LoadToolConfig(path string) (analysis.ToolConfig, error) {
	tc, err := defaultToolConfig()
	if err != nil || path == "" {
		return tc, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return analysis.ToolConfig{}, fmt.Errorf("reading tool config: %w", err)
	}
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
