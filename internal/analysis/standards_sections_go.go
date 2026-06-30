package analysis

// ── Go Conventions ────────────────────────────────────────────────────────────
// Enterprise-grade production Go rules covering concurrency, resources, errors, and nil safety.

type GoConventionsStd struct {
	Severity        string                    `toml:"severity"`
	ErrorHandling   GoErrorHandlingStd        `toml:"error_handling"`
	Concurrency     GoConcurrencyStd          `toml:"concurrency"`
	ContextProp     GoContextPropStd          `toml:"context_propagation"`
	Naming          GoNamingStd               `toml:"naming"`
	InterfaceDesign GoInterfaceDesignStd      `toml:"interface_design"`
	ResourceMgmt    GoResourceManagementStd   `toml:"resource_management"`
	NilSafety       GoNilSafetyStd            `toml:"nil_safety"`
}

// GoErrorHandlingStd enforces error wrapping and checked error handling patterns.
type GoErrorHandlingStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// GoConcurrencyStd enforces safe concurrent patterns: goroutine leaks, race conditions,
// deadlock patterns, channel safety, and select timeouts.
type GoConcurrencyStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// GoContextPropStd enforces context.Context threading through the call stack.
type GoContextPropStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// GoNamingStd enforces Go naming conventions (receiver names, unexported private names).
type GoNamingStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

// GoInterfaceDesignStd enforces small, focused interfaces and interface segregation.
type GoInterfaceDesignStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// GoResourceManagementStd enforces proper resource cleanup: unclosed bodies,
// file descriptor leaks, connection pool exhaustion, and defer unlock ordering.
type GoResourceManagementStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

// GoNilSafetyStd enforces nil checks before dereferences and safe nil operations.
type GoNilSafetyStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}
