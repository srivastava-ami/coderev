package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Standards is the typed representation of code_review_standards.toml.
type Standards struct {
	Meta           Meta              `toml:"meta"`
	Security       SecurityStd       `toml:"security"`
	Stability      StabilityStd      `toml:"stability"`
	Hardcoding     HardcodingStd     `toml:"hardcoding"`
	TypeSafety     TypeSafetyStd     `toml:"type_safety"`
	Complexity     ComplexityStd     `toml:"complexity"`
	FileStructure  FileStructureStd  `toml:"file_structure"`
	Observability  ObservabilityStd  `toml:"observability"`
	Documentation  DocumentationStd  `toml:"documentation"`
	Testing        TestingStd        `toml:"testing"`
	Performance    PerformanceStd    `toml:"performance"`
	NxConventions      NxConventionsStd      `toml:"nx_conventions"`
	GoConventions      GoConventionsStd      `toml:"go_conventions"`
	PythonConventions  PythonConventionsStd  `toml:"python_conventions"`
	RustConventions    RustConventionsStd    `toml:"rust_conventions"`
	Exceptions         []Exception           `toml:"exceptions"`
}

type Meta struct {
	Version     string   `toml:"version"`
	LastUpdated string   `toml:"last_updated"`
	AppliesTo   []string `toml:"applies_to"`
}

// Exception allows opt-out of specific rules on a per-file/module basis.
type Exception struct {
	Rule          string `toml:"rule"`
	FileOrModule  string `toml:"file_or_module"`
	Justification string `toml:"justification"`
	ApprovedBy    string `toml:"approved_by"`
	Expires       string `toml:"expires"`
	Ticket        string `toml:"ticket"`
}

// Load parses a standards TOML file into a Standards struct.
func Load(path string) (Standards, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Standards{}, fmt.Errorf("reading standards file: %w", err)
	}
	return loadTOML(data)
}

func loadTOML(data []byte) (Standards, error) {
	var s Standards
	if _, err := toml.Decode(string(data), &s); err != nil {
		return Standards{}, fmt.Errorf("parsing standards TOML: %w", err)
	}
	return s, nil
}
