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
	"hardcoding.magic_number":   {},

	// documentation
	"documentation.no_comment_tombstones": {},
	"documentation.todo_format":           {},
	"documentation.missing_comment":       {},

	// security (built-in pattern checks)
	"security.no_eval":              {Tags: []string{"owasp:A03:2021", "cwe:95"}, Standards: []string{"OWASP-2021-A03", "CWE-95"}},
	"security.no_inner_html":        {Tags: []string{"owasp:A03:2021", "cwe:79"}, Standards: []string{"OWASP-2021-A03", "CWE-79"}},
	"security.no_weak_crypto":       {Tags: []string{"owasp:A02:2021", "cwe:327"}, Standards: []string{"OWASP-2021-A02", "CWE-327"}},
	"security.no_prototype_pollution": {Tags: []string{"owasp:A08:2021", "cwe:1321"}, Standards: []string{"OWASP-2021-A08", "CWE-1321"}},
	// security via external adapters
	"security.secrets":      {Tags: []string{"owasp:A07:2021", "cwe:798"}, Standards: []string{"OWASP-2021-A07", "CWE-798"}},
	"security.dependencies": {Tags: []string{"owasp:A06:2021"}, Standards: []string{"OWASP-2021-A06"}},

	// testing
	"testing.coverage": {},

	// nx_conventions
	"nx_conventions.no_deep_import": {},
	"nx_conventions.boundaries":     {},

	// go_conventions
	"go.fmt_print":         {Tags: []string{"owasp:A09:2021"}, Standards: []string{"OWASP-2021-A09"}},
	"go.panic_in_lib":      {Tags: []string{"cwe:703"}, Standards: []string{"CWE-703"}},
	"go.sql_string_concat": {Tags: []string{"owasp:A03:2021", "cwe:89"}, Standards: []string{"OWASP-2021-A03", "CWE-89"}},
	"go.context_todo":      {},
	"go.defer_in_loop":     {Tags: []string{"cwe:772"}, Standards: []string{"CWE-772"}},
}
