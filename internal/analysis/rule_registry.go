package analysis

// RuleMeta carries the classification metadata for a rule ID.
type RuleMeta struct {
	ID          string
	Description string
	Tags        []string // e.g. ["owasp:A03:2021", "cwe:89"]
	Standards   []string // e.g. ["OWASP-2021-A03", "CWE-89"]
}

// RuleRegistry maps every built-in rule ID to its metadata.
// emitFinding uses this to auto-populate Tags and Standards on every Finding.
var RuleRegistry = map[string]RuleMeta{
	// complexity
	"complexity.cyclomatic":         {Tags: []string{"cwe:1121"}, Standards: []string{"CWE-1121"}},
	"complexity.cognitive":          {Tags: []string{"cwe:1121"}, Standards: []string{"CWE-1121"}},
	"complexity.function_length":    {Tags: []string{"cwe:1121"}},
	"complexity.parameter_count":    {Tags: []string{"cwe:1121"}},
	"complexity.nesting":            {Tags: []string{"cwe:1121"}},
	"complexity.max_return_count":   {Tags: []string{"cwe:1121"}},
	"complexity.boolean_param_flag": {Tags: []string{"cwe:1121"}},

	// file_structure
	"file_structure.file_length":   {},
	"file_structure.class_length":  {},
	"file_structure.duplication":   {Tags: []string{"cwe:1041"}, Standards: []string{"CWE-1041"}},
	"file_structure.circular_deps": {Tags: []string{"cwe:1120"}, Standards: []string{"CWE-1120"}},

	// type_safety
	"type_safety.no_any":               {Tags: []string{"cwe:704"}, Standards: []string{"CWE-704"}},
	"type_safety.no_non_null_assertion": {Tags: []string{"cwe:476"}, Standards: []string{"CWE-476"}},
	"type_safety.no_force_cast":         {Tags: []string{"cwe:704"}, Standards: []string{"CWE-704"}},

	// observability
	"observability.logging": {Tags: []string{"owasp:A09:2021"}, Standards: []string{"OWASP-2021-A09"}},

	// stability
	"stability.error_handling":      {Tags: []string{"owasp:A05:2021", "cwe:390"}, Standards: []string{"OWASP-2021-A05", "CWE-390"}},
	"stability.no_floating_promise": {Tags: []string{"cwe:390"}, Standards: []string{"CWE-390"}},
	"stability.no_throw_literal":    {Tags: []string{"cwe:390"}, Standards: []string{"CWE-390"}},
	"stability.no_await_in_loop":    {Tags: []string{"cwe:835"}, Standards: []string{"CWE-835"}},

	// hardcoding
	"hardcoding.urls_and_paths": {Tags: []string{"owasp:A05:2021", "cwe:259"}, Standards: []string{"OWASP-2021-A05", "CWE-259"}},
	"hardcoding.magic_numbers":  {Tags: []string{"cwe:1121"}, Standards: []string{"CWE-1121"}},

	// documentation
	"documentation.no_comment_tombstones": {},
	"documentation.todo_format":           {},
	"documentation.missing_comment":       {},

	// security (built-in pattern checks)
	"security.no_eval":              {Tags: []string{"owasp:A03:2021", "cwe:95"}, Standards: []string{"OWASP-2021-A03", "CWE-95"}},
	"security.no_inner_html":        {Tags: []string{"owasp:A03:2021", "cwe:79"}, Standards: []string{"OWASP-2021-A03", "CWE-79"}},
	"security.no_weak_crypto":       {Tags: []string{"owasp:A02:2021", "cwe:327"}, Standards: []string{"OWASP-2021-A02", "CWE-327"}},
	"security.no_prototype_pollution": {Tags: []string{"owasp:A08:2021", "cwe:1321"}, Standards: []string{"OWASP-2021-A08", "CWE-1321"}},
	// secret_fallback_literal is fully native: dot/bracket forms in
	// walker_security_fallback.go, destructuring-default form in walker_injection.go.
	// semgrep is now optional enrichment, not required for this rule.
	"security.secret_fallback_literal": {Tags: []string{"owasp:A07:2021", "cwe:798"}, Standards: []string{"OWASP-2021-A07", "CWE-798"}},
	// security via external adapters
	"security.secrets":      {Tags: []string{"owasp:A07:2021", "cwe:798"}, Standards: []string{"OWASP-2021-A07", "CWE-798"}},
	"security.dependencies": {Tags: []string{"owasp:A06:2021"}, Standards: []string{"OWASP-2021-A06"}},

	// testing
	"testing.coverage": {},

	// nx_conventions
	"nx_conventions.no_deep_import": {},
	"nx_conventions.boundaries":     {},

	// rust_conventions (legacy)
	"rust.no_unwrap":     {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"rust.no_panic":      {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"rust.no_expect":     {},
	"rust.no_unsafe":     {Tags: []string{"cwe:119"}, Standards: []string{"CWE-119"}},
	"rust.no_transmute":  {Tags: []string{"cwe:704"}, Standards: []string{"CWE-704"}},
	"rust.clone_on_copy": {},
	"rust.no_todo":       {},
	"rust.no_dbg_macro":  {},

	// rust_conventions (Phase 1: 15 enterprise-grade rules)
	"rust_conventions.unsafe_block_justification": {Tags: []string{"cwe:119"}, Standards: []string{"CWE-119"}},
	"rust_conventions.panic_in_library":           {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"rust_conventions.unwrap_in_library":          {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"rust_conventions.unbounded_lifetime":         {Tags: []string{"cwe:682"}, Standards: []string{"CWE-682"}},
	"rust_conventions.mutable_static":             {Tags: []string{"cwe:366"}, Standards: []string{"CWE-366"}},
	"rust_conventions.error_propagation":          {Tags: []string{"cwe:707"}, Standards: []string{"CWE-707"}},
	"rust_conventions.result_discard":             {Tags: []string{"cwe:391"}, Standards: []string{"CWE-391"}},
	"rust_conventions.panic_hook_missing":         {Tags: []string{"cwe:248"}, Standards: []string{"CWE-248"}},
	"rust_conventions.custom_error_impl":          {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"rust_conventions.clone_heavy":                {Tags: []string{"cwe:398"}, Standards: []string{"CWE-398"}},
	"rust_conventions.expensive_operation_loop":   {Tags: []string{"cwe:1121"}, Standards: []string{"CWE-1121"}},
	"rust_conventions.iter_collect_chain":         {Tags: []string{"cwe:1121"}, Standards: []string{"CWE-1121"}},
	"rust_conventions.async_cancel_safety":        {Tags: []string{"cwe:248"}, Standards: []string{"CWE-248"}},
	"rust_conventions.borrowed_reference_lifetime": {Tags: []string{"cwe:562"}, Standards: []string{"CWE-562"}},
	"rust_conventions.mutable_borrow_scope":        {Tags: []string{"cwe:416"}, Standards: []string{"CWE-416"}},

	// go_conventions
	"go.fmt_print":               {Tags: []string{"owasp:A09:2021"}, Standards: []string{"OWASP-2021-A09"}},
	"go.panic_in_lib":            {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"go.sql_string_concat":       {Tags: []string{"owasp:A03:2021", "cwe:89"}, Standards: []string{"OWASP-2021-A03", "CWE-89"}},
	"go.context_todo":            {},
	"go.defer_in_loop":           {Tags: []string{"cwe:772"}, Standards: []string{"CWE-772"}},
	"go.fmt_errorf_no_format":    {Tags: []string{"cwe:134"}, Standards: []string{"CWE-134"}},
	"go.io_copy_no_limit":        {Tags: []string{"cwe:400"}, Standards: []string{"CWE-400"}},

	// python_conventions (legacy)
	"python.fmt_print":           {},
	"python.no_bare_except":      {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"python.no_eval_exec":        {Tags: []string{"owasp:A03:2021", "cwe:95"}, Standards: []string{"OWASP-2021-A03", "CWE-95"}},
	"python.sql_injection":       {Tags: []string{"owasp:A03:2021", "cwe:89"}, Standards: []string{"OWASP-2021-A03", "CWE-89"}},
	"python.no_subprocess_shell": {Tags: []string{"owasp:A03:2021", "cwe:78"}, Standards: []string{"OWASP-2021-A03", "CWE-78"}},
	"python.no_mutable_default":  {},
	"python.no_wildcard_import":  {},

	// python_conventions (18 new enterprise-grade rules)
	// Type safety (5 rules)
	"python_conventions.type_hints_missing":  {Tags: []string{"cwe:704"}, Standards: []string{"CWE-704"}},
	"python_conventions.none_coercion":       {Tags: []string{"cwe:476"}, Standards: []string{"CWE-476"}},
	"python_conventions.dynamic_attribute":   {Tags: []string{"cwe:476"}, Standards: []string{"CWE-476"}},
	"python_conventions.type_inconsistency":  {Tags: []string{"cwe:704"}, Standards: []string{"CWE-704"}},
	"python_conventions.duck_typing_unsafe":  {Tags: []string{"cwe:704"}, Standards: []string{"CWE-704"}},
	// Async/concurrency (4 rules)
	"python_conventions.unclosed_async_resource": {Tags: []string{"cwe:772", "cwe:400"}, Standards: []string{"CWE-772", "CWE-400"}},
	"python_conventions.async_deadlock":          {Tags: []string{"cwe:833"}, Standards: []string{"CWE-833"}},
	"python_conventions.task_leak":               {Tags: []string{"cwe:401", "cwe:772"}, Standards: []string{"CWE-401", "CWE-772"}},
	"python_conventions.event_loop_mismatch":     {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	// Exception handling (4 rules)
	"python_conventions.bare_except":          {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"python_conventions.exception_swallowing": {Tags: []string{"cwe:391"}, Standards: []string{"CWE-391"}},
	"python_conventions.exception_chaining":   {Tags: []string{"cwe:707"}, Standards: []string{"CWE-707"}},
	"python_conventions.finally_side_effects": {Tags: []string{"cwe:248"}, Standards: []string{"CWE-248"}},
	// Import organization (3 rules)
	"python_conventions.circular_import": {Tags: []string{"cwe:1120"}, Standards: []string{"CWE-1120"}},
	"python_conventions.import_order":    {Tags: []string{"cwe:1120"}, Standards: []string{"CWE-1120"}},
	"python_conventions.unused_import":   {Tags: []string{"cwe:1104"}, Standards: []string{"CWE-1104"}},
	// Memory & resource management (2 rules)
	"python_conventions.resource_leak":     {Tags: []string{"cwe:772", "cwe:775"}, Standards: []string{"CWE-772", "CWE-775"}},
	"python_conventions.unbounded_growth":  {Tags: []string{"cwe:400", "cwe:401"}, Standards: []string{"CWE-400", "CWE-401"}},

	// nodejs_conventions (Phase 1: 7 enterprise-grade rules for production Node.js)
	// Streams (4 rules)
	"nodejs_conventions.stream_not_piped":       {Tags: []string{"cwe:400", "cwe:401"}, Standards: []string{"CWE-400", "CWE-401"}},
	"nodejs_conventions.backpressure_ignored":   {Tags: []string{"cwe:400", "cwe:401"}, Standards: []string{"CWE-400", "CWE-401"}},
	"nodejs_conventions.stream_error_unhandled": {Tags: []string{"cwe:391", "cwe:248"}, Standards: []string{"CWE-391", "CWE-248"}},
	"nodejs_conventions.stream_leak":            {Tags: []string{"cwe:772", "cwe:775"}, Standards: []string{"CWE-772", "CWE-775"}},
	// Event Emitters (3 rules)
	"nodejs_conventions.event_listener_leak":   {Tags: []string{"cwe:401", "cwe:772"}, Standards: []string{"CWE-401", "CWE-772"}},
	"nodejs_conventions.once_vs_on":            {Tags: []string{"cwe:1121"}, Standards: []string{"CWE-1121"}},
	"nodejs_conventions.error_event_unhandled": {Tags: []string{"cwe:248", "cwe:391"}, Standards: []string{"CWE-248", "CWE-391"}},
}

// ── Generic TOML-Driven Standards Infrastructure ──────────────────────────
// This enables new rule categories to be added without Go code changes.

// GenericRuleCategory represents a single rule category with arbitrary fields.
// It deserializes any TOML nested table into a map for flexible rule definitions.
type GenericRuleCategory map[string]interface{}

// GetString retrieves a string field from the generic category.
func (g GenericRuleCategory) GetString(key string) string {
	if v, ok := g[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetStringSlice retrieves a []string field.
func (g GenericRuleCategory) GetStringSlice(key string) []string {
	v, ok := g[key]
	if !ok {
		return nil
	}
	slice, ok := v.([]interface{})
	if !ok {
		return nil
	}
	return extractStrings(slice)
}

func extractStrings(slice []interface{}) []string {
	var out []string
	for _, item := range slice {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// GetInt retrieves an int field.
func (g GenericRuleCategory) GetInt(key string) int {
	if v, ok := g[key]; ok {
		switch val := v.(type) {
		case int64:
			return int(val)
		case float64:
			return int(val)
		case int:
			return val
		}
	}
	return 0
}

// GetBool retrieves a bool field.
func (g GenericRuleCategory) GetBool(key string) bool {
	if v, ok := g[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}