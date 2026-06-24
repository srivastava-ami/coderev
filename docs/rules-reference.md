# Rules Reference

All 55 built-in rules, grouped by pillar, with TOML configuration and severity defaults.

---

## Default standards

coderev ships with built-in standards embedded in the binary, organised by pillar.
No configuration file is needed — just run `coderev .`.

| File | Pillar |
|---|---|
| complexity.toml | Cyclomatic, cognitive, nesting, parameters, function length |
| security.toml | Secrets, dependencies |
| stability.toml | Error handling, async |
| hardcoding.toml | Magic numbers, hardcoded URLs |
| type_safety.toml | TypeScript any, null assertions |
| observability.toml | Logging |
| documentation.toml | TODOs, commented-out code |
| file_structure.toml | File and class length, circular deps |
| testing.toml | Coverage thresholds |
| go.toml | Go conventions |
| python.toml | Python conventions |
| rust.toml | Rust conventions |
| nx.toml | NX monorepo boundaries |

To override for a specific repo: `coderev --standards /path/to/custom.toml .`

---

## 1. Complexity

| Rule ID | What it detects | Languages | Default severity | Configurable |
|---|---|---|---|---|
| `complexity.cyclomatic` | Cyclomatic complexity > threshold | All | blocker | max_value, advisory_at, hard_block_at |
| `complexity.cognitive` | Cognitive complexity > threshold | All | blocker | max_value |
| `complexity.function_length` | Function body lines > threshold | All | blocker | max_lines, advisory_at |
| `complexity.parameter_count` | Function parameters > threshold | All | blocker | max_count |
| `complexity.nesting` | Nesting depth > threshold | All | blocker | max_depth |
| `complexity.max_return_count` | Return statements > 4 | All | advisory | — (hardcoded) |
| `complexity.boolean_param_flag` | Boolean-flag parameter names (`isFoo`, `hasBar`, etc.) | All | advisory | — |

```toml
[complexity]
severity = "blocker"

[complexity.cyclomatic]
max_value     = 10      # blocker if exceeded
advisory_at   = 7       # advisory if exceeded but below max_value
hard_block_at = 15      # always blocker regardless of severity setting

[complexity.cognitive]
max_value = 15

[complexity.function_length]
max_lines   = 40
advisory_at = 30

[complexity.parameter_count]
max_count = 3

[complexity.nesting]
max_depth = 3

[complexity.duplication]
rule             = "file_structure.duplication"   # cross-file duplicate detection
threshold_tokens = 25
```

---

## 2. File Structure

| Rule ID | What it detects | Languages | Default severity | Configurable |
|---|---|---|---|---|
| `file_structure.file_length` | Total lines in file > threshold | All | advisory | max_lines, advisory_at |
| `file_structure.class_length` | Class/struct definition lines > threshold | All | advisory | max_lines |
| `file_structure.duplication` | Cross-file duplicate code blocks | TS/JS/Go | major | threshold_tokens |
| `file_structure.circular_deps` | Circular import chains | TS/JS | major | — (madge adapter) |

```toml
[file_structure]
severity = "advisory"

[file_structure.file_length]
max_lines   = 300
advisory_at = 200

[file_structure.class_length]
max_lines = 150
```

---

## 3. Type Safety

| Rule ID | What it detects | Languages | Default severity | Configurable |
|---|---|---|---|---|
| `type_safety.no_any` | `: any`, `as any`, `@ts-ignore` | TypeScript | blocker | — |
| `type_safety.no_non_null_assertion` | `!.` or `![` operators | TypeScript | major | — |
| `type_safety.no_force_cast` | `as unknown as X` double cast | TypeScript | major | — |

```toml
[type_safety]
severity = "blocker"

[type_safety.no_any]
rule   = "type_safety.no_any"
checks = ["no_any_type", "no_ts_ignore"]

[type_safety.null_safety]
rule   = "type_safety.null_safety"
checks = ["no_non_null_assertion"]
```

---

## 4. Stability

| Rule ID | What it detects | Languages | Default severity | Configurable |
|---|---|---|---|---|
| `stability.error_handling` | Empty catch blocks | TS/JS | blocker | — |
| `stability.no_floating_promise` | Un-awaited or unhandled async call | TS/JS | major | — |
| `stability.no_throw_literal` | `throw "string"` instead of `throw new Error()` | TS/JS | major | — |
| `stability.no_await_in_loop` | `await` inside a loop body | TS/JS | major | — |

```toml
[stability]
severity = "blocker"

[stability.error_handling]
rule   = "stability.error_handling"
checks = ["no_empty_catch", "no_swallowed_exceptions"]
```

---

## 5. Hardcoding

| Rule ID | What it detects | Languages | Default severity | Configurable |
|---|---|---|---|---|
| `hardcoding.urls_and_paths` | Hardcoded HTTP URLs (excl. localhost) | All | blocker | — |
| `hardcoding.magic_number` | Bare numeric literals not assigned to constants | All | advisory | exceptions (values to ignore) |

```toml
[hardcoding]
severity = "blocker"

[hardcoding.environment_values]
rule     = "hardcoding.urls_and_paths"
examples = ["API_URL", "DATABASE_URL", "SECRET_KEY"]

[hardcoding.magic_numbers]
severity    = "advisory"
rule        = "hardcoding.magic_number"
exceptions  = [0, 1, 2, 100, 1000]    # values to always allow
```

---

## 6. Security

| Rule ID | What it detects | Languages | Default severity | Configurable | Adapter |
|---|---|---|---|---|---|
| `security.no_eval` | `eval()`, `new Function()` | TS/JS | blocker | — | treesitter |
| `security.no_inner_html` | `.innerHTML =` assignment | TS/JS | blocker | — | treesitter |
| `security.no_weak_crypto` | MD5/SHA-1 references | All | blocker | — | treesitter |
| `security.no_prototype_pollution` | `__proto__` | TS/JS | blocker | — | treesitter |
| `security.secrets` | Hardcoded credentials, API keys | All | blocker | — | gitleaks |
| `security.secret_fallback_literal` | Env secret with hardcoded fallback | TS/JS | blocker (advisory in NODE_ENV guard) | — | treesitter + semgrep |
| `security.dependencies` | Vulnerable npm packages | JS/TS | blocker | — | npm audit |

### Built-in treesitter checks (zero dependencies):

```toml
# These run automatically — no config needed unless you want to disable them.
```

### External adapter checks (require `brew install gitleaks` or `coderev install-deps`):

```toml
# These are enabled by default if the tool is installed.
# Disable by setting enabled = false in tool_config.toml:
[adapters.gitleaks]
enabled = false
```

---

## 7. Observability

| Rule ID | What it detects | Languages | Default severity | Configurable |
|---|---|---|---|---|
| `observability.logging` | `console.log`, `console.error`, `console.warn` | TS/JS | blocker | — |

```toml
[observability]
severity = "blocker"

[observability.logging]
rule             = "observability.logging"
required_fields  = ["correlationId", "level", "message"]
checks           = ["no_console_log"]
forbidden_levels = ["console.log", "console.error", "console.warn"]
```

---

## 8. Documentation

| Rule ID | What it detects | Languages | Default severity | Configurable |
|---|---|---|---|---|
| `documentation.no_comment_tombstones` | Commented-out code | All | advisory | — |
| `documentation.todo_format` | TODO without ticket reference | All | advisory | pattern |
| `documentation.missing_comment` | Public API without doc comment | TS/JS | info | — (not yet implemented) |

```toml
[documentation]
severity = "advisory"

[documentation.comment_quality]
rule        = "documentation.comment_quality"
bad_patterns = ["TODO", "FIXME", "HACK", "XXX"]

[documentation.no_comment_tombstones]
rule        = "documentation.no_tombstones"
severity    = "advisory"
description = "Commented-out code is dead weight"
remediation = "Delete it — version control is your history."

[documentation.todo_format]
rule    = "documentation.todo_format"
pattern = "TODO\\(\\w+\\):"  # regex: e.g. TODO(amit): fix this
```

---

## 9. Testing

| Rule ID | What it detects | Languages | Default severity | Configurable | Adapter |
|---|---|---|---|---|---|
| `testing.coverage` | Lines below coverage threshold | All | major | threshold | coverage |

```toml
[testing]
severity = "blocker"

[testing.coverage]
lines       = 80
branches    = 80
functions   = 80
statements  = 80
```

To use: generate an `lcov.info` or `coverage.out` file first, then `coderev` reads it automatically.

---

## 10. Performance

| Rule ID | What it detects | Languages | Default severity |
|---|---|---|---|
| _(checks database: N+1, SELECT \*)_ | All | block |
| _(checks async: await-in-loop)_ | TS/JS | major |

```toml
[performance]

[performance.database]
severity = "blocker"
checks   = ["no_n_plus_one", "no_select_star"]

[performance.async]
severity = "major"
checks   = ["no_await_in_loop"]
```

---

## 11. Go Conventions

| Rule ID | What it detects | Default severity |
|---|---|---|
| `go.fmt_print` | `fmt.Println()`, `fmt.Printf()` in non-test files | advisory |
| `go.panic_in_lib` | `panic()` in non-test files | major |
| `go.sql_string_concat` | SQL query built with `fmt.Sprintf` or `+` concatenation | blocker |
| `go.context_todo` | `context.TODO()` in non-test files | advisory |
| `go.defer_in_loop` | `defer` inside a `for` loop | major |

```toml
[go_conventions]
severity = "advisory"

[go_conventions.error_handling]
rule   = "go.error_handling"
checks = ["no_panic_in_lib", "check_return_err"]

[go_conventions.context_propagation]
rule   = "go.context_propagation"
checks = ["no_context_todo"]
```

---

## 12. Python Conventions

| Rule ID | What it detects | Default severity |
|---|---|---|
| `python.fmt_print` | `print()` call in non-test files | advisory |
| `python.no_bare_except` | `except:` without exception type | blocker |
| `python.no_eval_exec` | `eval()`, `exec()` | blocker |
| `python.sql_injection` | SQL + string concatenation | blocker |
| `python.no_subprocess_shell` | `os.system()`, `os.popen()`, `shell=True` | blocker |
| `python.no_mutable_default` | `def foo(x=[])` or `def foo(x={})` | blocker |
| `python.no_wildcard_import` | `from module import *` | major |

```toml
[python_conventions]
severity = "advisory"
```

---

## 13. Rust Conventions

| Rule ID | What it detects | Default severity |
|---|---|---|
| `rust.no_unwrap` | `.unwrap()` in non-test files | blocker |
| `rust.no_panic` | `panic!()` in non-test files | major |
| `rust.no_expect` | `.expect()` in non-test files | advisory |
| `rust.no_unsafe` | `unsafe { }` blocks | blocker |
| `rust.no_transmute` | `transmute<`, `transmute::` | blocker |
| `rust.clone_on_copy` | `.clone()` on Copy types | advisory |
| `rust.no_todo` | `todo!()`, `unimplemented!()` | major |
| `rust.no_dbg_macro` | `dbg!()` macro | advisory |

```toml
[rust_conventions]
severity = "advisory"
```

---

## 14. NX Conventions

| Rule ID | What it detects | Default severity | Adapter |
|---|---|---|---|
| `nx_conventions.no_deep_import` | `../../` imports in non-test files | major | treesitter |
| `nx_conventions.boundaries` | Cross-boundary imports between NX libs | blocker | madge |

```toml
[nx_conventions]
severity = "blocker"

[nx_conventions.boundaries]
rule        = "nx_conventions.boundaries"
description = "Libraries must not import from app-level packages"
tool        = "madge"

[nx_conventions.tags]
rule           = "nx_conventions.tags"
required_axes  = ["scope", "type"]
```

---

## Exceptions

Skip specific rules on specific files. Every exception is tracked — visible in the report.

```toml
[[exceptions]]
rule           = "complexity.cyclomatic"
file_or_module = "src/legacy/parser.ts"
justification  = "Third-party vendored parser — tracked in JIRA-4421"
expires        = "2027-01-01"   # optional: auto-expires after this date
ticket         = "JIRA-4421"    # optional: reference ticket number

[[exceptions]]
rule           = "complexity.function_length"
file_or_module = "src/**/*.generated.ts"   # glob pattern
justification  = "Auto-generated files"
```

---

## Quality gate

Fail CI if findings exceed thresholds. Place this in a `.coderev-gate.toml` in your repo root:

```toml
max_blockers   = 0    # default, always 0 recommended
max_majors     = 5    # default
max_advisories = 10   # default
max_total      = 20   # default
```

```bash
coderev --gate .coderev-gate.toml .
```

Exit code 1 when any threshold is exceeded.

---

## Custom rules (plugins)

Install community plugins or write your own:

```bash
coderev plugin list
coderev plugin install path/to/my-plugin.toml
```

A plugin needs a manifest (`coderev-plugin.toml`) and a binary that outputs findings as NDJSON to stdout. See [docs/plugins.md](plugins.md) for the full protocol.

---

## Which adapter runs which rule?

| Adapter | Rules it handles | Requires install? |
|---|---|---|
| **treesitter** | All complexity, file_structure, type_safety, hardcoding, observability, stability, documentation, security (4), go (5), python (7), rust (8), nx_conventions.no_deep_import — **48 rules** | Built-in |
| **semgrep** | `security.injection.*`, `security.auth.*`, `security.cryptography` | `brew install semgrep` |
| **gitleaks** | `security.secrets` | `brew install gitleaks` |
| **madge** | `file_structure.circular_deps`, `nx_conventions.boundaries` | `npm i -g madge` |
| **npmaudit** | `security.dependencies` | Ships with Node |
| **coverage** | `testing.coverage` | Generate lcov.info first |
| **custom/script** | Any rule ID you assign | Your binary on `$PATH` |
