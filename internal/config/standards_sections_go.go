package config

// ── Go Conventions ────────────────────────────────────────────────────────────

type GoConventionsStd struct {
	Severity        string               `toml:"severity"`
	ErrorHandling   GoErrorHandlingStd   `toml:"error_handling"`
	Concurrency     GoConcurrencyStd     `toml:"concurrency"`
	ContextProp     GoContextPropStd     `toml:"context_propagation"`
	Naming          GoNamingStd          `toml:"naming"`
	InterfaceDesign GoInterfaceDesignStd `toml:"interface_design"`
}

type GoErrorHandlingStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

type GoConcurrencyStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

type GoContextPropStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

type GoNamingStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

type GoInterfaceDesignStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}
