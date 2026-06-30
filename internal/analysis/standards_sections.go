package analysis

// ── Security ──────────────────────────────────────────────────────────────────

type SecurityStd struct {
	Severity string         `toml:"severity"`
	Secrets  SecretsStd     `toml:"secrets"`
	Supply   SupplyChainStd `toml:"supply_chain"`
}

type SecretsStd struct {
	Rule        string   `toml:"rule"`
	Patterns    []string `toml:"patterns"`
	Remediation string   `toml:"remediation"`
}

type SupplyChainStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

// ── Stability ─────────────────────────────────────────────────────────────────

type StabilityStd struct {
	Severity      string           `toml:"severity"`
	ErrorHandling ErrorHandlingStd `toml:"error_handling"`
}

type ErrorHandlingStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

// ── Hardcoding ────────────────────────────────────────────────────────────────

type HardcodingStd struct {
	Severity          string          `toml:"severity"`
	EnvironmentValues EnvValuesStd    `toml:"environment_values"`
	MagicNumbers      MagicNumbersStd `toml:"magic_numbers"`
}

type EnvValuesStd struct {
	Rule        string   `toml:"rule"`
	Examples    []string `toml:"examples"`
	Remediation string   `toml:"remediation"`
}

type MagicNumbersStd struct {
	Severity    string `toml:"severity"`
	Rule        string `toml:"rule"`
	Exceptions  []int  `toml:"exceptions"`
	Remediation string `toml:"remediation"`
}

// ── Type Safety ───────────────────────────────────────────────────────────────

type TypeSafetyStd struct {
	Severity   string        `toml:"severity"`
	NoAny      NoAnyStd      `toml:"no_any"`
	NullSafety NullSafetyStd `toml:"null_safety"`
}

type NoAnyStd struct {
	Rule        string   `toml:"rule"`
	Checks      []string `toml:"checks"`
	Remediation string   `toml:"remediation"`
}

type NullSafetyStd struct {
	Rule   string   `toml:"rule"`
	Checks []string `toml:"checks"`
}

// ── Complexity ────────────────────────────────────────────────────────────────

type ComplexityStd struct {
	Severity    string            `toml:"severity"`
	Cyclomatic  CyclomaticStd     `toml:"cyclomatic"`
	Cognitive   CognitiveStd      `toml:"cognitive"`
	Function    FunctionLengthStd `toml:"function_length"`
	Parameters  ParameterStd      `toml:"parameter_count"`
	Nesting     NestingStd        `toml:"nesting"`
	Duplication DuplicationStd    `toml:"duplication"`
}

type CyclomaticStd struct {
	MaxValue    int    `toml:"max_value"`
	AdvisoryAt  int    `toml:"advisory_at"`
	HardBlockAt int    `toml:"hard_block_at"`
	Remediation string `toml:"remediation"`
}

type CognitiveStd struct {
	MaxValue int    `toml:"max_value"`
	Tool     string `toml:"tool"`
}

type FunctionLengthStd struct {
	MaxLines    int    `toml:"max_lines"`
	AdvisoryAt  int    `toml:"advisory_at"`
	Remediation string `toml:"remediation"`
}

type ParameterStd struct {
	MaxCount    int    `toml:"max_count"`
	Remediation string `toml:"remediation"`
}

type NestingStd struct {
	MaxDepth    int    `toml:"max_depth"`
	Remediation string `toml:"remediation"`
}

type DuplicationStd struct {
	Rule            string `toml:"rule"`
	ThresholdTokens int    `toml:"threshold_tokens"`
	Remediation     string `toml:"remediation"`
}

// ── File Structure ────────────────────────────────────────────────────────────

type FileStructureStd struct {
	Severity    string         `toml:"severity"`
	FileLength  FileLengthStd  `toml:"file_length"`
	ClassLength ClassLengthStd `toml:"class_length"`
}

type FileLengthStd struct {
	MaxLines    int    `toml:"max_lines"`
	AdvisoryAt  int    `toml:"advisory_at"`
	Remediation string `toml:"remediation"`
}

type ClassLengthStd struct {
	MaxLines    int    `toml:"max_lines"`
	Remediation string `toml:"remediation"`
}

// ── Observability ─────────────────────────────────────────────────────────────

type ObservabilityStd struct {
	Severity string     `toml:"severity"`
	Logging  LoggingStd `toml:"logging"`
}

type LoggingStd struct {
	Rule            string   `toml:"rule"`
	RequiredFields  []string `toml:"required_fields"`
	Checks          []string `toml:"checks"`
	ForbiddenLevels []string `toml:"forbidden_levels"`
}

// ── Documentation ─────────────────────────────────────────────────────────────

type DocumentationStd struct {
	Severity       string            `toml:"severity"`
	CommentQuality CommentQualityStd `toml:"comment_quality"`
	NoTombstones   NoTombstonesStd   `toml:"no_comment_tombstones"`
	TodoFormat     TodoFormatStd     `toml:"todo_format"`
}

type CommentQualityStd struct {
	Rule        string   `toml:"rule"`
	BadPatterns []string `toml:"bad_patterns"`
	Remediation string   `toml:"remediation"`
}

type NoTombstonesStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Description string `toml:"description"`
	Remediation string `toml:"remediation"`
}

type TodoFormatStd struct {
	Rule        string `toml:"rule"`
	Pattern     string `toml:"pattern"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

// ── Testing ───────────────────────────────────────────────────────────────────

type TestingStd struct {
	Severity string      `toml:"severity"`
	Coverage CoverageStd `toml:"coverage"`
}

type CoverageStd struct {
	Lines          int `toml:"lines"`
	Branches       int `toml:"branches"`
	Functions      int `toml:"functions"`
	Statements     int `toml:"statements"`
	NewCodeMinimum int `toml:"new_code_minimum"`
}

// ── Performance ───────────────────────────────────────────────────────────────

type PerformanceStd struct {
	Database PerformanceDBStd    `toml:"database"`
	Async    PerformanceAsyncStd `toml:"async"`
}

type PerformanceDBStd struct {
	Severity string   `toml:"severity"`
	Checks   []string `toml:"checks"`
}

type PerformanceAsyncStd struct {
	Severity string   `toml:"severity"`
	Checks   []string `toml:"checks"`
}

// ── Python Conventions ─────────────────────────────────────────────────────────

type PythonConventionsStd struct {
	Severity          string               `toml:"severity"`
	TypeSafety        PythonTypeSafetyStd  `toml:"type_safety"`
	AsyncPatterns     PythonAsyncStd       `toml:"async_patterns"`
	ExceptionHandling PythonExceptionStd   `toml:"exception_handling"`
	Imports           PythonImportsStd     `toml:"imports"`
	MemoryResources   PythonMemoryStd      `toml:"memory_resources"`
}

type PythonTypeSafetyStd struct {
	TypeHintsMissing    RuleEntryStd `toml:"type_hints_missing"`
	NoneCoercion        RuleEntryStd `toml:"none_coercion"`
	DynamicAttribute    RuleEntryStd `toml:"dynamic_attribute"`
	TypeInconsistency   RuleEntryStd `toml:"type_inconsistency"`
	DuckTypingUnsafe    RuleEntryStd `toml:"duck_typing_unsafe"`
}

type PythonAsyncStd struct {
	UnclosedAsyncResource RuleEntryStd `toml:"unclosed_async_resource"`
	AsyncDeadlock         RuleEntryStd `toml:"async_deadlock"`
	TaskLeak              RuleEntryStd `toml:"task_leak"`
	EventLoopMismatch     RuleEntryStd `toml:"event_loop_mismatch"`
}

type PythonExceptionStd struct {
	BareExcept           RuleEntryStd `toml:"bare_except"`
	ExceptionSwallowing  RuleEntryStd `toml:"exception_swallowing"`
	ExceptionChaining    RuleEntryStd `toml:"exception_chaining"`
	FinallySideEffects   RuleEntryStd `toml:"finally_side_effects"`
}

type PythonImportsStd struct {
	CircularImport RuleEntryStd `toml:"circular_import"`
	ImportOrder    RuleEntryStd `toml:"import_order"`
	UnusedImport   RuleEntryStd `toml:"unused_import"`
}

type PythonMemoryStd struct {
	ResourceLeak    RuleEntryStd `toml:"resource_leak"`
	UnboundedGrowth RuleEntryStd `toml:"unbounded_growth"`
}

type RuleEntryStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

// ── JavaScript/TypeScript Conventions ──────────────────────────────────────

type JavaScriptConventionsStd struct {
	Severity       string                  `toml:"severity"`
	TypeSafety    JSTypeSafetyStd         `toml:"type_safety"`
	PromisesAsync JSPromisesAsyncStd      `toml:"promises_async"`
	Modules       JSModulesStd            `toml:"modules"`
	DataFlow      JSDataFlowStd           `toml:"data_flow"`
}

// TypeSafety category: 5 rules
type JSTypeSafetyStd struct {
	AnyTypeUsage        JSRuleStd `toml:"any_type_usage"`
	TypeCoercion        JSRuleStd `toml:"type_coercion"`
	OptionalChainingUse JSRuleStd `toml:"optional_chaining_overuse"`
	NullCoalescingUse   JSRuleStd `toml:"null_coalescing_correct"`
	TypeAssertionUnsafe JSRuleStd `toml:"type_assertion_unsafe"`
}

// PromisesAsync category: 5 rules
type JSPromisesAsyncStd struct {
	UnhandledPromise   JSRuleStd `toml:"unhandled_promise"`
	FloatingPromise    JSRuleStd `toml:"floating_promise"`
	AsyncAwaitChaining JSRuleStd `toml:"async_await_chain"`
	PromiseRaceHazard  JSRuleStd `toml:"promise_race_hazard"`
	CallbackHell       JSRuleStd `toml:"callback_hell"`
}

// Modules category: 3 rules
type JSModulesStd struct {
	CircularDependency JSRuleStd `toml:"circular_dependency"`
	ImportOrder        JSRuleStd `toml:"import_order"`
	WildcardImport     JSRuleStd `toml:"wildcard_import"`
}

// DataFlow category: 3 rules (security-focused)
type JSDataFlowStd struct {
	DomXSS             JSRuleStd `toml:"dom_xss"`
	EvalUsage          JSRuleStd `toml:"eval_usage"`
	PrototypePollution JSRuleStd `toml:"prototype_pollution"`
}

type JSRuleStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
	Description string `toml:"description"`
}

// ── Rust Conventions ───────────────────────────────────────────────────────────

type RustConventionsStd struct {
	Severity string `toml:"severity"`
	// Memory Safety (5 rules)
	UnsafeBlockJustif    RustUnsafeBlockJustifStd    `toml:"unsafe_block_justification"`
	PanicInLibrary       RustPanicInLibraryStd       `toml:"panic_in_library"`
	UnwrapInLibrary      RustUnwrapInLibraryStd      `toml:"unwrap_in_library"`
	UnboundedLifetime    RustUnboundedLifetimeStd    `toml:"unbounded_lifetime"`
	MutableStatic        RustMutableStaticStd        `toml:"mutable_static"`
	// Error Handling (4 rules)
	ErrorPropagation     RustErrorPropagationStd     `toml:"error_propagation"`
	ResultDiscard        RustResultDiscardStd        `toml:"result_discard"`
	PanicHookMissing     RustPanicHookMissingStd     `toml:"panic_hook_missing"`
	CustomErrorImpl       RustCustomErrorImplStd      `toml:"custom_error_impl"`
	// Patterns (4 rules)
	CloneHeavy           RustCloneHeavyStd           `toml:"clone_heavy"`
	ExpensiveOpLoop      RustExpensiveOpLoopStd      `toml:"expensive_operation_loop"`
	IterCollectChain     RustIterCollectChainStd     `toml:"iter_collect_chain"`
	AsyncCancelSafety    RustAsyncCancelSafetyStd    `toml:"async_cancel_safety"`
	// Borrowing (2 rules)
	BorrowedRefLifetime  RustBorrowedRefLifetimeStd  `toml:"borrowed_reference_lifetime"`
	MutableBorrowScope   RustMutableBorrowScopeStd   `toml:"mutable_borrow_scope"`
}

type RustUnsafeBlockJustifStd struct {
	Rule              string `toml:"rule"`
	Severity          string `toml:"severity"`
	RequireComment    bool   `toml:"require_comment"`
	CommentKeywords   []string `toml:"comment_keywords"`
	Remediation       string `toml:"remediation"`
}

type RustMutableStaticStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustBorrowedRefLifetimeStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustMutableBorrowScopeStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustUnboundedLifetimeStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustPanicInLibraryStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustUnwrapInLibraryStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustErrorPropagationStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustResultDiscardStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustCloneHeavyStd struct {
	Rule            string `toml:"rule"`
	Severity        string `toml:"severity"`
	CloneThreshold  int    `toml:"clone_threshold"`
	Remediation     string `toml:"remediation"`
}

type RustExpensiveOpLoopStd struct {
	Rule        string   `toml:"rule"`
	Severity    string   `toml:"severity"`
	ExpensiveOps []string `toml:"expensive_ops"`
	Remediation string   `toml:"remediation"`
}

type RustIterCollectChainStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustAsyncCancelSafetyStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustPanicHookMissingStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

type RustCustomErrorImplStd struct {
	Rule        string `toml:"rule"`
	Severity    string `toml:"severity"`
	Remediation string `toml:"remediation"`
}

// ── NX Conventions ────────────────────────────────────────────────────────────

type NxConventionsStd struct {
	Severity   string        `toml:"severity"`
	Boundaries BoundariesStd `toml:"boundaries"`
	Tags       TagsStd       `toml:"tags"`
}

type BoundariesStd struct {
	Rule        string `toml:"rule"`
	Description string `toml:"description"`
	Tool        string `toml:"tool"`
}

type TagsStd struct {
	Rule            string   `toml:"rule"`
	RequiredTagAxes []string `toml:"required_axes"`
}

// ── Node.js Conventions ───────────────────────────────────────────────────────

type NodeJsConventionsStd struct {
	Severity      string                 `toml:"severity"`
	Streams       NodeJsStreamsStd       `toml:"streams"`
	EventEmitters NodeJsEventEmittersStd `toml:"event_emitters"`
	AsyncPatterns NodeJsAsyncPatternsStd `toml:"async_patterns"`
	Performance   NodeJsPerformanceStd   `toml:"performance"`
}

type NodeJsStreamsStd struct {
	StreamNotPiped       RuleEntryStd `toml:"stream_not_piped"`
	BackpressureIgnored  RuleEntryStd `toml:"backpressure_ignored"`
	StreamErrorUnhandled RuleEntryStd `toml:"stream_error_unhandled"`
	StreamLeak           RuleEntryStd `toml:"stream_leak"`
}

type NodeJsEventEmittersStd struct {
	EventListenerLeak    RuleEntryStd `toml:"event_listener_leak"`
	OnceVsOn             RuleEntryStd `toml:"once_vs_on"`
	ErrorEventUnhandled  RuleEntryStd `toml:"error_event_unhandled"`
}

type NodeJsAsyncPatternsStd struct {
	CallbackHell              RuleEntryStd `toml:"callback_hell"`
	PromiseSwallowing         RuleEntryStd `toml:"promise_swallowing"`
	AsyncIteratorIncomplete   RuleEntryStd `toml:"async_iterator_incomplete"`
	ConcurrentOperationsMax   RuleEntryStd `toml:"concurrent_operations_unbounded"`
}

type NodeJsPerformanceStd struct {
	MemoryLeakTimers RuleEntryStd `toml:"memory_leak_timers"`
	UnboundedBuffer  RuleEntryStd `toml:"unbounded_buffer"`
	CpuBlocking      RuleEntryStd `toml:"cpu_blocking"`
}
