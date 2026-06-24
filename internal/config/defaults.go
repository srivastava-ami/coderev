package config

import (
	_ "embed"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

//go:embed default_standards.toml
var defaultStandardsData []byte

// LoadDefaults returns the built-in standards embedded in the binary.
func LoadDefaults() (analysis.Standards, error) {
	return loadBytes(defaultStandardsData)
}

func loadBytes(data []byte) (analysis.Standards, error) {
	return loadTOML(data)
}
