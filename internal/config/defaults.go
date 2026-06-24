package config

import (
	"embed"
	"sort"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

//go:embed defaults/*.toml
var defaultStandardsFS embed.FS

// LoadDefaults returns the built-in standards embedded in the binary.
func LoadDefaults() (analysis.Standards, error) {
	entries, err := defaultStandardsFS.ReadDir("defaults")
	if err != nil {
		return analysis.Standards{}, err
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var data []byte
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		d, err := defaultStandardsFS.ReadFile("defaults/" + e.Name())
		if err != nil {
			return analysis.Standards{}, err
		}
		data = append(data, d...)
		data = append(data, '\n')
	}

	return loadTOML(data)
}

func loadBytes(data []byte) (analysis.Standards, error) {
	return loadTOML(data)
}
