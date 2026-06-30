package analysis

// Standards is the typed representation of code_review_standards.toml.
type Standards struct {
	Meta              Meta                  `toml:"meta"`
	Security          SecurityStd           `toml:"security"`
	Stability         StabilityStd          `toml:"stability"`
	Hardcoding        HardcodingStd         `toml:"hardcoding"`
	TypeSafety        TypeSafetyStd         `toml:"type_safety"`
	Complexity        ComplexityStd         `toml:"complexity"`
	FileStructure     FileStructureStd      `toml:"file_structure"`
	Observability     ObservabilityStd      `toml:"observability"`
	Documentation     DocumentationStd      `toml:"documentation"`
	Testing           TestingStd            `toml:"testing"`
	Performance       PerformanceStd        `toml:"performance"`
	NxConventions     NxConventionsStd      `toml:"nx_conventions"`
	GoConventions     GoConventionsStd      `toml:"go_conventions"`
	PythonConventions PythonConventionsStd  `toml:"python_conventions"`
	RustConventions   RustConventionsStd    `toml:"rust_conventions"`
	JavaScriptConventions JavaScriptConventionsStd `toml:"javascript_conventions"`
	NodeJsConventions     NodeJsConventionsStd     `toml:"nodejs_conventions"`
	TerraformConventions  TerraformConventionsStd  `toml:"terraform_conventions"`

	// Generic map-based fields for TOML-driven extensibility.
	Pillars  map[string]map[string]GenericRuleCategory `toml:"-"`
	Severity map[string]string                        `toml:"-"`
	RawData  map[string]interface{}                   `toml:"-"`
	Exceptions        []Exception           `toml:"exceptions"`
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

// UnmarshalTOML populates the generic Pillars map after TOML deserialization.
func (s *Standards) UnmarshalTOML(data interface{}) error {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	s.RawData = m
	PopulateGenericPillars(s, m)
	RegisterRulesFromStandards(s)
	return nil
}
