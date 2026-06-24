package config

import _ "embed"

//go:embed default_standards.toml
var defaultStandardsData []byte

// LoadDefaults returns the built-in standards embedded in the binary.
// Used when no code_review_standards.toml is found in the target repo.
func LoadDefaults() (Standards, error) {
	return loadBytes(defaultStandardsData)
}

func loadBytes(data []byte) (Standards, error) {
	return loadTOML(data)
}
