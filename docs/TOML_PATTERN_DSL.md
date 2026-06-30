# TOML Pattern DSL Reference

A complete guide to defining code analysis rules in TOML. All rules live in `internal/config/rules/` and are embedded in the binary. No code changes needed to add a new rule.

## Quick Start

Create a file `internal/config/rules/phase1/my_rules.toml`:

```toml
[rules.security_sql_injection]
rule = "security.sql_injection"
severity = "blocker"
pillar = "security"
description = "SQL injection vulnerability"
remediation = "Use parameterized queries. Never concatenate user input into SQL strings."
cwe = "CWE-89"
owasp = "A01:2021"
languages = ["python", "javascript"]

[[rules.security_sql_injection.patterns]]
type = "multi_line"
pattern = "sql_concat"
pattern_sequence = ["SELECT.*?WHERE", "\\+\\s*", "[a-zA-Z_]"]
lines = 3
message = "Potential SQL injection: concatenation with variable"
```

Restart coderev. The new rule is live.

## Rule Structure

Every rule TOML file contains a `[rules]` section. Each rule is a subsection:

```toml
[rules.RULE_ID]
rule = "pillar.name"              # Full rule ID (format: pillar.name)
severity = "blocker"              # blocker, major, advisory
pillar = "security"               # security, complexity, stability, etc.
description = "Human-readable"    # One sentence
remediation = "How to fix it"     # Concrete action
cwe = "CWE-89"                    # Common Weakness Enumeration (optional)
owasp = "A01:2021"                # OWASP Top 10 reference (optional)
standards = ["OWASP-2021-A01"]   # List of standards (optional)
reference_url = "https://..."     # External documentation (optional)
languages = ["go", "python"]      # Target languages; empty = all
```

### Metadata Fields

- **rule**: Unique identifier. Format: `pillar.name` (e.g., `security.sql_injection`).
- **severity**: One of `blocker` (fail scan), `major` (warning), `advisory` (info).
- **pillar**: Category. Standard pillars: `security`, `complexity`, `stability`, `reliability`, `performance`, `hardcoding`, `type_safety`, `documentation`, `observability`, `testing`.
- **remediation**: Concrete action to fix (1-2 sentences).
- **cwe**: Common Weakness Enumeration ID (e.g., `CWE-89` for SQL injection).
- **owasp**: OWASP Top 10 category (e.g., `A01:2021` for injection).
- **standards**: List of compliance standards (e.g., `["PCI-DSS-4.1", "HIPAA"]`).
- **reference_url**: Link to detailed documentation or standard definition.
- **languages**: Supported languages. Empty or omitted = all languages.

## Pattern Types

A rule contains one or more `[[rules.RULE_ID.patterns]]` sections. Each pattern is evaluated independently; if **any** pattern matches, the rule fires.

### 1. **string_match** — Regex or literal substring

Simplest pattern. Detects literal strings or regex patterns.

```toml
[[rules.hardcoding_urls.patterns]]
type = "string_match"
pattern = "http://"                         # Regex: match http:// URLs
exclude = ["localhost", "127.0.0.1", "test"]  # Ignore these substrings
confidence = "high"
message = "Hardcoded unencrypted URL"
```

**Fields:**
- `pattern`: Regex (compiled) or literal string to match.
- `exclude`: List of substrings. If ANY exclude substring is found in the line, the pattern does NOT match.
- `confidence`: `high`, `medium`, or `low` (advisory; not enforced by severity).
- `message`: Finding message shown to user.

**Examples:**

```toml
# Detect hardcoded secrets
pattern = "password\\s*=\\s*['\"]"

# Detect console.log in JS/TS
pattern = "console\\.(log|error|warn)\\("

# Detect SQL keywords
pattern = "SELECT|INSERT|UPDATE|DELETE"
```

---

### 2. **method_call** — Detect function/method invocation

Matches function or method calls (e.g., `eval()`, `fetch()`, `db.Query()`).

```toml
[[rules.security_eval.patterns]]
type = "method_call"
pattern = "eval\\s*\\("                    # Regex for the method signature
exclude = ["safeEval", "evalJSON"]         # Don't flag these alternatives
message = "eval() executes arbitrary code — use Function() or parse JSON"
```

**Fields:**
- `pattern`: Regex matching the method/function call syntax.
- `exclude`: List of safe alternatives to ignore.
- `message`: Finding message.

**Examples:**

```toml
# Detect process.exit() in library code
pattern = "process\\.exit\\s*\\("

# Detect unsafe DOM methods
pattern = "innerHTML\\s*="

# Detect subprocess calls
pattern = "subprocess\\.call|exec|Popen"
```

---

### 3. **import** — Detect dangerous imports

Flags imports of unsafe/forbidden modules.

```toml
[[rules.security_unsafe_import.patterns]]
type = "import"
pattern = "pickle|marshal"                 # Module names
exclude = ["safe_pickle_wrapper"]          # Allowed wrappers
message = "Unsafe deserialization — use json instead"
```

**Fields:**
- `pattern`: Module name (literal or part of module path).
- `exclude`: Safe wrappers or approved alternatives.

**Examples:**

```toml
# Detect eval imports
pattern = "eval"

# Detect dangerous cryptography modules
pattern = "Crypto\\.Cipher|hashlib"

# Detect deprecated packages
pattern = "deprecated_lib"
```

---

### 4. **variable_assignment** — Detect hardcoded secrets in assignments

Finds secret-like variable names assigned to literal values (not env vars).

```toml
[[rules.hardcoding_secrets.patterns]]
type = "variable_assignment"
exclude = ["JWT_SECRET", "API_KEY", "DB_PASSWORD"]  # Secret-like names to flag
message = "Secret variable assigned to hardcoded literal"
```

**Note:** This pattern is an early-stage heuristic. For production use, prefer `string_match` with explicit patterns or `semantic` checks.

---

### 5. **function_def** — Detect functions with too many parameters

Flags functions exceeding a parameter threshold.

```toml
[[rules.complexity_params.patterns]]
type = "function_def"
pattern = "5"                              # Max params (naive counting)
message = "Function has >5 parameters — use a parameter object"
```

**Limitations:** Parameter counting is naive (splits on commas) and doesn't handle nested generics well. Use `metric` for AST-aware complexity checks.

---

### 6. **loop_contains** — Detect operations inside loops

Detects keywords/patterns inside loops (e.g., memory allocation in hot paths).

```toml
[[rules.performance_loop_allocation.patterns]]
type = "loop_contains"
inside = "for"                              # Loop type: for, while, foreach
contains = ["new ", "malloc", "Vector()"]   # Keywords to find inside loop
message = "Memory allocation in loop — allocate before loop"
```

**Fields:**
- `inside`: `for`, `while`, or `foreach` (language-dependent).
- `contains`: List of keywords to find inside the loop.

**Examples:**

```toml
# Detect database queries in loops (N+1 pattern)
inside = "for"
contains = ["SELECT", "Query", "db.find"]

# Detect regex compilation in loops
inside = "while"
contains = ["new Regex", "re.compile"]
```

---

### 7. **multi_line** — Detect pattern sequences

Matches a sequence of patterns on consecutive (or nearly consecutive) lines.

```toml
[[rules.performance_n_plus_one.patterns]]
type = "multi_line"
pattern_sequence = ["for\\s*\\(", "(SELECT|Query)"]  # Patterns in order
lines = 5                                   # Lookahead max 5 lines
message = "N+1 query pattern — use JOIN instead"
```

**Fields:**
- `pattern_sequence`: List of regex patterns, matched in order.
- `lines`: Lookahead window (lines to scan ahead).

**Examples:**

```toml
# Detect loop with async/await misuse
pattern_sequence = ["for\\s*\\(", "await"]
lines = 3

# Detect promise without error handling
pattern_sequence = ["\\.then\\(", "^(?!.*\\.catch)"]  # .then without .catch
lines = 2

# Detect SQL + concatenation = injection
pattern_sequence = ["SELECT.*WHERE", "\\+\\s*"]
lines = 3
```

---

### 8. **negative** — Detect missing patterns

Fires when a function call is present **but** a required guard/handler is absent.

```toml
[[rules.reliability_timeout.patterns]]
type = "negative"
function_calls = ["fetch", "axios.get", "requests.get"]  # Detect these calls
missing = "timeout|WithTimeout"                          # Require this nearby
message = "Network call without timeout — risk of hanging indefinitely"
```

**Fields:**
- `function_calls`: List of function/method names to detect.
- `missing`: Pattern that should appear within 5 lines but doesn't.

**Examples:**

```toml
# Detect channel ops without timeout
function_calls = ["<-ch", "ch<-"]
missing = "time.After|context"

# Detect database query without error check
function_calls = ["db.Query", "cursor.execute"]
missing = "if.*err|except"

# Detect file open without close
function_calls = ["open(", "File::new"]
missing = "close\\(|with\\s|defer"
```

---

### 9. **metric** — Detect quantitative violations

Fires when a metric (cyclomatic complexity, method count, etc.) exceeds a threshold.

```toml
[[rules.complexity_cyclomatic.patterns]]
type = "metric"
metric = "cyclomatic_complexity"            # Metric to measure
threshold_gt = 10                           # Threshold (greater than)
message = "Cyclomatic complexity >10 — extract branches into helpers"
```

**Supported Metrics:**
- `cyclomatic_complexity`: Branch count (if/else/switch/case/try/catch).
- `method_count`: Number of methods in an interface.
- `coupling`: Fan-out (external function calls).
- `nesting_depth`: Max nesting level.

**Examples:**

```toml
# Detect god classes
metric = "method_count"
threshold_gt = 20

# Detect highly coupled functions
metric = "coupling"
threshold_gt = 15

# Detect deep nesting
metric = "nesting_depth"
threshold_gt = 4
```

---

### 10. **semantic** — Language-specific heuristics

Detects language-specific patterns (nil dereference in Go, unwrap in Rust, etc.). **Use with caution:** these are heuristics, not AST-aware.

```toml
[[rules.go_nil_dereference.patterns]]
type = "semantic"
semantic = "nil_pointer_dereference"        # Heuristic to apply
language = "go"                             # Language-specific
message = "Dereference without nil check — potential panic"
```

**Supported Semantics:**
- `nil_pointer_dereference` (Go): Dereference without prior nil check.
- `unchecked_cast` (Rust): Unsafe cast (`as T`) without validation.
- `resource_leak` (Python): File open without close or context manager.

---

## Organization

Rules are organized by phase and category:

```
internal/config/rules/
  core/
    sample.toml          # 3 core patterns (hardcoded_http, cyclomatic, error_handling)
  phase1/
    go_sample.toml       # 5 Go Phase 1 rules (goroutine_leak, deadlock, etc.)
    python_sample.toml   # (Future)
  phase2/
    compliance_sample.toml # 5 compliance rules (OWASP, PCI-DSS, etc.)
    security_sample.toml   # (Future)
  phase3/
    (Future)
```

Phases indicate maturity and coverage:
- **core**: Foundation rules (all languages, high confidence).
- **phase1**: Language-specific conventions (Go, Python, Rust, etc.).
- **phase2**: Compliance & security (OWASP, PCI-DSS, HIPAA, etc.).
- **phase3+**: Advanced patterns and integrations.

---

## Common Mistakes

### ❌ Hardcoding URLs in rules

```toml
# Bad: hardcoded URL in remediation
remediation = "See https://example.com for details"

# Good: externalise to reference_url
reference_url = "https://example.com/details"
remediation = "See the reference URL for details"
```

### ❌ Overly broad patterns

```toml
# Bad: matches too much, high false positive rate
pattern = "http"  # Matches "http_client", "httpd", etc.

# Good: anchor the pattern
pattern = "http://"  # Only unencrypted URLs
```

### ❌ Not excluding safe alternatives

```toml
# Bad: flags safe helpers
pattern = "eval\\("

# Good: exclude safe wrappers
pattern = "eval\\("
exclude = ["safeEval", "JSON.parse"]
```

### ❌ Ambiguous language specifications

```toml
# Bad: rule fires for every language (if empty)
languages = []

# Good: be explicit
languages = ["typescript", "javascript"]
```

### ❌ Non-deterministic rules

```toml
# Bad: regex with high cost (catastrophic backtracking)
pattern = "(a|ab)*b"  # Can be exponentially slow

# Good: simple, linear patterns
pattern = "async\\s+function"
```

---

## Examples

### Example 1: SQL Injection Detection

```toml
[rules.security_sql_injection]
rule = "security.sql_injection"
severity = "blocker"
pillar = "security"
description = "SQL injection via string concatenation"
remediation = "Use parameterized queries. Never concatenate user input into SQL."
cwe = "CWE-89"
owasp = "A01:2021"
reference_url = "https://owasp.org/Top10/A01_2021-Injection/"
languages = ["python", "javascript", "go"]

[[rules.security_sql_injection.patterns]]
type = "multi_line"
pattern_sequence = ["SELECT.*WHERE", "\\+"]  # SQL + concatenation
lines = 3
message = "Potential SQL injection: query + string concatenation"
```

### Example 2: Goroutine Leak (Go)

```toml
[rules.go_goroutine_leak]
rule = "go.goroutine_leak"
severity = "blocker"
pillar = "stability"
description = "Goroutine without context cancellation"
remediation = "Pass context.Context and use context.WithCancel/WithTimeout"
standards = ["CWE-772"]
languages = ["go"]

[[rules.go_goroutine_leak.patterns]]
type = "string_match"
pattern = "go\\s+[a-zA-Z_].*\\(\\)"
message = "Goroutine launched without context"
```

### Example 3: Timeout Missing (Multi-language)

```toml
[rules.reliability_timeout]
rule = "reliability.timeout_missing"
severity = "major"
pillar = "reliability"
description = "Network/database operation without timeout"
remediation = "Add timeout: context.WithTimeout(), requests.get(..., timeout=30)"
standards = ["CWE-1091"]
languages = ["go", "python", "javascript"]

[[rules.reliability_timeout.patterns]]
type = "negative"
function_calls = ["fetch", "axios", "requests.get", "http.Client", "Query"]
missing = "timeout|WithTimeout|context"
message = "Network/database call without timeout — risk of indefinite hang"
```

### Example 4: PCI-DSS HTTPS Enforcement

```toml
[rules.compliance_pci_dss_encryption]
rule = "compliance.pci_dss_encryption"
severity = "blocker"
pillar = "security"
description = "Unencrypted HTTP (PCI-DSS 4.1 violation)"
remediation = "Use HTTPS/TLS. Enforce with HSTS headers (Strict-Transport-Security)."
standards = ["PCI-DSS-4.1"]
reference_url = "https://www.pcisecuritystandards.org/documents/PCI_DSS-v3_2_1.pdf"
languages = []  # All languages

[[rules.compliance_pci_dss_encryption.patterns]]
type = "string_match"
pattern = "http://"
exclude = ["localhost", "127.0.0.1", "test"]
message = "Unencrypted HTTP detected — PCI-DSS requires HTTPS"
```

---

## Testing Your Rules

### Verify the rule loads

```bash
coderev --help
# Rules are loaded automatically; no explicit flag needed
```

### Test against a file

```bash
cat > test.go << 'EOF'
package main
func main() {
	go doSomething()  // Should match go.goroutine_leak
}
func doSomething() {}
EOF

coderev scan test.go --json | jq '.findings[] | select(.rule | contains("goroutine"))'
```

### Check embedded rules

```bash
# Look inside the binary to verify TOML was embedded
strings coderev | grep "rules.go_goroutine_leak" | head -1
```

---

## Debugging

If a rule doesn't fire:

1. **Check language match**: Verify the file extension is in the rule's `languages` list.
2. **Test the pattern**: Use `grep` or an online regex tester to ensure the pattern matches.
3. **Check exclusions**: An `exclude` list might be suppressing the match.
4. **Inspect logs**: `coderev scan . --verbose` shows pattern matching details.
5. **Verify TOML syntax**: Run `toml` parser on the rule file (invalid TOML fails silently on load).

---

## FAQ

**Q: Can I override a rule with custom settings?**

A: Not yet. Rules are embedded and built-in. Future phases will support per-repo `code_review_standards.toml` overrides.

**Q: How often are rules updated?**

A: Rules are updated with each release. Install the latest via `brew upgrade coderev` or `curl` installer.

**Q: Can I add custom rules?**

A: Yes: fork the repo, add a TOML file under `internal/config/rules/`, and rebuild. Phase B+ will support external rule files.

**Q: What if I have a false positive?**

A: Open an issue with the file, pattern, and expected behavior. The rule will be refined or the `exclude` list extended.
