# Rules Reference

All 150 built-in rules (55 core + 95 Phase 1 enterprise), grouped by pillar, with TOML configuration and severity defaults.

**New in v0.17 (Phase 1 Complete):** 95 enterprise-grade convention rules across Go, Python, Rust, JavaScript/TypeScript, Node.js, and Terraform.

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

**Phase 1 (v0.17):** 20 enterprise-grade rules covering concurrency, error handling, resource management, interfaces, and nil safety.

### Core Rules (v0.16)

| Rule ID | What it detects | Default severity |
|---|---|---|
| `go.fmt_print` | `fmt.Println()`, `fmt.Printf()` in non-test files | advisory |
| `go.panic_in_lib` | `panic()` in non-test files | major |
| `go.sql_string_concat` | SQL query built with `fmt.Sprintf` or `+` concatenation | blocker |
| `go.context_todo` | `context.TODO()` in non-test files | advisory |
| `go.defer_in_loop` | `defer` inside a `for` loop | major |

### Phase 1 Enterprise Rules (v0.17)

#### Concurrency Safety (6 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `go_conventions.goroutine_leak` | Goroutines spawned without tracking/cancellation | blocker |
| `go_conventions.race_condition` | Unprotected shared memory access | major |
| `go_conventions.deadlock_pattern` | Channel operations without timeout in select | major |
| `go_conventions.channel_safety` | Closing channel on receiving end or send after close | blocker |
| `go_conventions.select_timeout` | select without timeout for external calls | major |

#### Error Handling (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `go_conventions.unchecked_error` | Ignored error returns with blank assignment | major |
| `go_conventions.error_wrapping` | Missing context in wrapped errors (fmt.Errorf) | major |
| `go_conventions.defer_panic` | panic() in defer blocks | blocker |
| `go_conventions.defer_unlock_order` | Incorrect defer lock/unlock ordering | major |

#### Interface Design (3 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `go_conventions.interface_bloat` | Interfaces with >5 methods | major |
| `go_conventions.interface_segregation` | Clients depending on unused interface methods | major |
| `go_conventions.pointer_receiver_consistency` | Inconsistent receiver types (pointer vs value) | advisory |

#### Resource Management (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `go_conventions.unclosed_body` | http.Response.Body not closed | blocker |
| `go_conventions.file_descriptor_leak` | Unclosed files/connections | blocker |
| `go_conventions.pool_exhaustion` | Unbounded resource acquisition | major |
| `go_conventions.memory_leak_patterns` | Unbounded maps/slices, circular references | major |

#### Nil Safety (3 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `go_conventions.nil_pointer_dereference` | Unguarded nil dereferences | major |
| `go_conventions.nil_slice_iteration` | Range over potentially nil slices | major |
| `go_conventions.nil_method_call` | Method calls on uninitialized structs | major |

```toml
[go_conventions]
severity = "major"

[go_conventions.concurrency]
rule   = "go_conventions.goroutine_leak"
checks = ["untracked_goroutine"]

[go_conventions.resource_management]
rule   = "go_conventions.unclosed_body"
checks = ["response_body_close"]
```

---

## 12. Python Conventions

**Phase 1 (v0.17):** 18 enterprise-grade rules covering type safety, async/concurrency, exception handling, imports, and resources.

### Core Rules (v0.16)

| Rule ID | What it detects | Default severity |
|---|---|---|
| `python.fmt_print` | `print()` call in non-test files | advisory |
| `python.no_bare_except` | `except:` without exception type | blocker |
| `python.no_eval_exec` | `eval()`, `exec()` | blocker |
| `python.sql_injection` | SQL + string concatenation | blocker |
| `python.no_subprocess_shell` | `os.system()`, `os.popen()`, `shell=True` | blocker |
| `python.no_mutable_default` | `def foo(x=[])` or `def foo(x={})` | blocker |
| `python.no_wildcard_import` | `from module import *` | major |

### Phase 1 Enterprise Rules (v0.17)

#### Type Safety (5 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `python_conventions.type_hints_missing` | Functions without type annotations | major |
| `python_conventions.none_coercion` | Implicit None comparisons (if x: vs if x is not None:) | major |
| `python_conventions.dynamic_attribute` | getattr/setattr on user input | blocker |
| `python_conventions.type_inconsistency` | Functions returning different types | major |
| `python_conventions.duck_typing_unsafe` | Attribute access without isinstance checks | major |

#### Async/Concurrency (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `python_conventions.unclosed_async_resource` | Async context managers not awaited (aiohttp, asyncpg) | blocker |
| `python_conventions.async_deadlock` | Blocking calls in async functions (time.sleep, requests.get) | major |
| `python_conventions.task_leak` | asyncio.create_task() not tracked/awaited | major |
| `python_conventions.event_loop_mismatch` | Manual event loop creation in wrong thread | major |

#### Exception Handling (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `python_conventions.bare_except` | `except:` without specific exception type | blocker |
| `python_conventions.exception_swallowing` | Empty except blocks (just pass) | major |
| `python_conventions.exception_chaining` | Exceptions without context (raise ... from) | major |
| `python_conventions.finally_side_effects` | Side effects/IO in finally blocks | major |

#### Import Organization (3 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `python_conventions.circular_import` | Circular dependency between modules | major |
| `python_conventions.import_order` | Incorrect order (stdlib → third-party → local) | major |
| `python_conventions.unused_import` | Unused or aliased with underscore | major |

#### Memory/Resources (2 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `python_conventions.resource_leak` | Unclosed files/connections (missing with statement) | blocker |
| `python_conventions.unbounded_growth` | Unbounded lists/dicts/caches (.append without checks) | major |

```toml
[python_conventions]
severity = "major"

[python_conventions.type_safety]
rule = "python_conventions.type_hints_missing"
checks = ["missing_annotations"]

[python_conventions.async_patterns]
rule = "python_conventions.unclosed_async_resource"
checks = ["aiohttp_session", "asyncpg_connection"]
```

---

## 13. Rust Conventions

**Phase 1 (v0.17):** 15 enterprise-grade rules covering memory safety, error handling, patterns, and borrowing.

### Core Rules (v0.16)

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

### Phase 1 Enterprise Rules (v0.17)

#### Memory Safety (5 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `rust_conventions.unsafe_block_justification` | unsafe blocks without // SAFETY comment | blocker |
| `rust_conventions.panic_in_library` | panic!() in library code (use Result) | blocker |
| `rust_conventions.unwrap_in_library` | .unwrap()/.expect() in library code | blocker |
| `rust_conventions.unbounded_lifetime` | Generic lifetimes without bounds | major |
| `rust_conventions.mutable_static` | static mut without synchronization (Mutex/atomic) | blocker |

#### Error Handling (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `rust_conventions.error_propagation` | Lossy error conversions (.ok()) | major |
| `rust_conventions.result_discard` | Discarded Results without explicit ignore | major |
| `rust_conventions.panic_hook_missing` | Missing panic handler in main.rs | major |
| `rust_conventions.custom_error_impl` | Incomplete Error trait implementation | major |

#### Patterns (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `rust_conventions.clone_heavy` | Excessive cloning instead of references | major |
| `rust_conventions.expensive_operation_loop` | Allocations in hot loops (Vec::new) | major |
| `rust_conventions.iter_collect_chain` | Unnecessary collect() in iterator chains | major |
| `rust_conventions.async_cancel_safety` | Drop inconsistencies in async code | major |

#### Borrowing (2 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `rust_conventions.borrowed_reference_lifetime` | References outliving referent | major |
| `rust_conventions.mutable_borrow_scope` | Mutable borrows with unnecessary scope | advisory |

```toml
[rust_conventions]
severity = "major"

[rust_conventions.memory_safety]
rule = "rust_conventions.unsafe_block_justification"
checks = ["missing_safety_comment"]
```

---

## 14. JavaScript/TypeScript Conventions

**Phase 1 (v0.17):** 16 enterprise-grade rules covering type safety, promises/async, module systems, and data flow security.

| Rule ID | Category | What it detects | Default severity |
|---|---|---|---|
| `js_conventions.any_type_usage` | Type Safety | `: any`, `as any`, `@ts-ignore` | blocker |
| `js_conventions.type_coercion` | Type Safety | Implicit type coercion (`==`, `!=`) | major |
| `js_conventions.optional_chaining_overuse` | Type Safety | Optional chaining on literals (`?.`) | major |
| `js_conventions.null_coalescing_correct` | Type Safety | Null coalescing with error-prone values (`??`) | major |
| `js_conventions.type_assertion_unsafe` | Type Safety | Unsafe type assertions (`as any`, `as unknown as`) | major |
| `js_conventions.unhandled_promise` | Promises/Async | `.then()` without `.catch()` | blocker |
| `js_conventions.floating_promise` | Promises/Async | Promises not awaited or returned | major |
| `js_conventions.async_await_chain` | Promises/Async | Mixing await with `.then()` | major |
| `js_conventions.promise_race_hazard` | Promises/Async | Incomplete Promise.race handling | major |
| `js_conventions.callback_hell` | Promises/Async | Deeply nested callbacks (3+ levels) | major |
| `js_conventions.circular_dependency` | Modules | Module circular imports | major |
| `js_conventions.import_order` | Modules | Mixing import styles (CJS/ESM) | major |
| `js_conventions.wildcard_import` | Modules | `import *` unless namespaced | major |
| `js_conventions.dom_xss` | Data Flow | innerHTML/textContent from untrusted sources | blocker |
| `js_conventions.eval_usage` | Data Flow | eval() or Function() with dynamic code | blocker |
| `js_conventions.prototype_pollution` | Data Flow | Object.assign with untrusted objects | blocker |

---

## 15. Node.js Conventions

**Phase 1 (v0.17):** 14 enterprise-grade rules covering streams, event emitters, async patterns, and performance.

#### Streams (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `nodejs_conventions.stream_not_piped` | Streams created but not piped | major |
| `nodejs_conventions.backpressure_ignored` | Ignoring stream.write() return value | blocker |
| `nodejs_conventions.stream_error_unhandled` | Stream error handlers missing | major |
| `nodejs_conventions.stream_leak` | Streams not destroyed on error | blocker |

#### Event Emitters (3 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `nodejs_conventions.event_listener_leak` | Listeners not removed (memory leak) | blocker |
| `nodejs_conventions.once_vs_on` | Using `.on()` for single-event listeners | major |
| `nodejs_conventions.error_event_unhandled` | Unhandled 'error' event (crashes process) | blocker |

#### Async Patterns (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `nodejs_conventions.callback_hell` | Deeply nested callbacks (3+ levels) | major |
| `nodejs_conventions.promise_swallowing` | Promises without error handling | major |
| `nodejs_conventions.async_iterator_incomplete` | Incomplete async iterator implementation | major |
| `nodejs_conventions.concurrent_operations_unbounded` | Unbounded concurrent operations (Promise.all) | major |

#### Performance (3 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `nodejs_conventions.memory_leak_timers` | setInterval without clearInterval | blocker |
| `nodejs_conventions.unbounded_buffer` | Unbounded internal buffering (.push) | major |
| `nodejs_conventions.cpu_blocking` | Sync operations blocking event loop (readFileSync) | major |

---

## 16. Terraform Conventions

**Phase 1 (v0.17):** 12 enterprise-grade rules covering best practices, resource design, and compliance.

#### Best Practices (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `terraform_conventions.hardcoded_values` | Hardcoded resource names, regions, AZs | blocker |
| `terraform_conventions.provider_version_pinning` | Unpinned provider versions (no required_version) | blocker |
| `terraform_conventions.variable_defaults_sensitive` | Sensitive data in variable defaults | blocker |
| `terraform_conventions.state_file_exposure` | State files in git (missing .gitignore) | blocker |

#### Resource Design (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `terraform_conventions.resource_naming` | Inconsistent resource naming (mixed cases) | major |
| `terraform_conventions.count_vs_for_each` | Using count for dynamic resources | major |
| `terraform_conventions.module_coupling` | Modules with hard dependencies | major |
| `terraform_conventions.data_source_safety` | Unsafe data source queries (no filters) | blocker |

#### Compliance (4 rules)
| Rule ID | What it detects | Default severity |
|---|---|---|
| `terraform_conventions.public_resource_exposure` | Publicly accessible resources without auth | blocker |
| `terraform_conventions.encryption_disabled` | Storage without encryption enabled | blocker |
| `terraform_conventions.logging_disabled` | Resources without logging enabled | major |
| `terraform_conventions.backup_missing` | No backup strategy defined (RDS, databases) | major |

---

## 17. NX Conventions

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
| **treesitter** | All complexity, file_structure, type_safety, hardcoding, observability, stability, documentation, security (4), go (5+20), python (7+18), rust (8+15), js (0+16), nodejs (0+14), terraform (0+12), nx_conventions.no_deep_import — **150 rules** | Built-in |
| **semgrep** | `security.injection.*`, `security.auth.*`, `security.cryptography` | `brew install semgrep` |
| **gitleaks** | `security.secrets` | `brew install gitleaks` |
| **madge** | `file_structure.circular_deps`, `nx_conventions.boundaries` | `npm i -g madge` |
| **npmaudit** | `security.dependencies` | Ships with Node |
| **coverage** | `testing.coverage` | Generate lcov.info first |
| **custom/script** | Any rule ID you assign | Your binary on `$PATH` |

**Note:** All 95 Phase 1 enterprise rules are handled by the native **treesitter** adapter (pure Go, zero external dependencies).
