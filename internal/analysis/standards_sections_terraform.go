package analysis

// ── Terraform Conventions ─────────────────────────────────────────────────────
// 12 enterprise-grade rules for Infrastructure as Code (IaC) best practices,
// resource design, and compliance across three categories.

type TerraformConventionsStd struct {
	Severity       string                       `toml:"severity"`
	BestPractices  TerraformBestPracticesStd    `toml:"best_practices"`
	ResourceDesign TerraformResourceDesignStd   `toml:"resource_design"`
	Compliance     TerraformComplianceStd       `toml:"compliance"`
}

// ── Category 1: Best Practices (4 rules) ──────────────────────────────────────
// Focus: configuration quality, versioning, and source control safety

type TerraformBestPracticesStd struct {
	HardcodedValues         TerraformRuleStd `toml:"hardcoded_values"`
	ProviderVersionPinning  TerraformRuleStd `toml:"provider_version_pinning"`
	VariableDefaultsSensitive TerraformRuleStd `toml:"variable_defaults_sensitive"`
	StateFileExposure       TerraformRuleStd `toml:"state_file_exposure"`
}

// HardcodedValuesStd detects hardcoded resource names, regions, and environment-specific values
// that should be variables for reusability and portability.
type TerraformRuleStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// ── Category 2: Resource Design (4 rules) ─────────────────────────────────────
// Focus: module design, resource patterns, and data source safety

type TerraformResourceDesignStd struct {
	ResourceNaming    TerraformRuleStd `toml:"resource_naming"`
	CountVsForEach    TerraformRuleStd `toml:"count_vs_for_each"`
	ModuleCoupling    TerraformRuleStd `toml:"module_coupling"`
	DataSourceSafety  TerraformRuleStd `toml:"data_source_safety"`
}

// ── Category 3: Compliance (4 rules) ──────────────────────────────────────────
// Focus: security, reliability, and audit requirements

type TerraformComplianceStd struct {
	PublicResourceExposure TerraformRuleStd `toml:"public_resource_exposure"`
	EncryptionDisabled     TerraformRuleStd `toml:"encryption_disabled"`
	LoggingDisabled        TerraformRuleStd `toml:"logging_disabled"`
	BackupMissing          TerraformRuleStd `toml:"backup_missing"`
}
