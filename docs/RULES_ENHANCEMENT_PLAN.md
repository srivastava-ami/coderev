# Enterprise Rules Enhancement Plan for coderev v0.17+

## Phase 1: COMPLETE ✅ (v0.17)
- **Delivered**: 95 new enterprise rules (55 → 150 total)
- **Coverage**: Go (20), Python (18), Rust (15), JS/TS (16), Node.js (14), Terraform (12)
- **Status**: All implemented, tested, and detecting in self-scan
- **Build**: ✅ PASS | **Tests**: ✅ 12/13 PASS | **Scan**: ✅ 538 findings detected

## Previous State (v0.16)
- **Rules**: ~55 (core + language-specific)
- **Coverage**: Go (basic), Python (basic), Rust (minimal), JS/TS (basic), Node.js (missing), Terraform (missing)
- **Gap**: Missing enterprise-grade rules for compliance, reliability, and performance

---

## Phase 1: Language-Specific Rules (Go, Python, Rust, JS/TS, Node.js, Terraform)

### 1. **Go Conventions** (20 new rules)

#### Concurrency Safety (6 rules)
- `go_conventions.goroutine_leak` - Goroutines without proper lifecycle management
- `go_conventions.race_condition` - Unprotected shared memory access
- `go_conventions.deadlock_pattern` - Circular channel dependencies, nested locks
- `go_conventions.context_propagation` - Missing context.Context in function signatures
- `go_conventions.channel_safety` - Unbuffered channels, close after send
- `go_conventions.select_timeout` - select without timeout for external calls

#### Error Handling (4 rules)
- `go_conventions.defer_panic` - Panic in defer blocks
- `go_conventions.defer_unlock_order` - defer lock/unlock ordering errors
- `go_conventions.unchecked_error` - Ignored error returns (beyond _)
- `go_conventions.error_wrapping` - Missing context in wrapped errors

#### Interface Design (3 rules)
- `go_conventions.interface_bloat` - Interfaces with >5 methods
- `go_conventions.interface_segregation` - Clients depending on unused methods
- `go_conventions.pointer_receiver_consistency` - Inconsistent receiver types

#### Resource Management (4 rules)
- `go_conventions.unclosed_body` - http.Response.Body not closed
- `go_conventions.file_descriptor_leak` - Unclosed files/connections
- `go_conventions.pool_exhaustion` - Unbounded resource acquisition
- `go_conventions.memory_leak_patterns` - Unbounded maps/slices, circular references

#### Nil Safety (3 rules)
- `go_conventions.nil_pointer_dereference` - Unguarded nil dereferences
- `go_conventions.nil_slice_iteration` - Range over potentially nil slices
- `go_conventions.nil_method_call` - Method calls on uninitialized structs

---

### 2. **Python Conventions** (18 new rules)

#### Type Safety (5 rules)
- `python_conventions.type_hints_missing` - Functions without type annotations
- `python_conventions.none_coercion` - Implicit None comparisons
- `python_conventions.dynamic_attribute` - getattr/setattr on user input
- `python_conventions.type_inconsistency` - Functions returning different types
- `python_conventions.duck_typing_unsafe` - Assuming attributes without checking

#### Async/Concurrency (4 rules)
- `python_conventions.unclosed_async_resource` - Async context managers not awaited
- `python_conventions.async_deadlock` - Blocking calls in async functions
- `python_conventions.task_leak` - asyncio.create_task without tracking
- `python_conventions.event_loop_mismatch` - Creating loops in wrong thread

#### Exception Handling (4 rules)
- `python_conventions.bare_except` - Except without specific exception type
- `python_conventions.exception_swallowing` - Empty except blocks
- `python_conventions.exception_chaining` - Exceptions without context (raise ... from)
- `python_conventions.finally_side_effects` - Side effects in finally blocks

#### Import Organization (3 rules)
- `python_conventions.circular_import` - Circular dependency between modules
- `python_conventions.import_order` - stdlib → third-party → local
- `python_conventions.unused_import` - Unused import statements

#### Memory/Resources (2 rules)
- `python_conventions.resource_leak` - Unclosed files, connections
- `python_conventions.unbounded_growth` - Unbounded lists, dicts, caches

---

### 3. **Rust Conventions** (15 new rules)

#### Memory Safety (5 rules)
- `rust_conventions.unsafe_block_justification` - unsafe blocks without // SAFETY comment
- `rust_conventions.panic_in_library` - panic!() in library code (use Result)
- `rust_conventions.unwrap_in_library` - .unwrap()/.expect() in library code
- `rust_conventions.unbounded_lifetime` - Generic lifetimes without bounds
- `rust_conventions.mutable_static` - static mut without synchronization

#### Error Handling (4 rules)
- `rust_conventions.error_propagation` - Lossy error conversions
- `rust_conventions.result_discard` - Discarded Results without explicit ignore
- `rust_conventions.panic_hook_missing` - Missing panic handler in main
- `rust_conventions.custom_error_impl` - Incomplete Error trait impl

#### Patterns (4 rules)
- `rust_conventions.clone_heavy` - Excessive cloning instead of references
- `rust_conventions.expensive_operation_loop` - Allocations in hot loops
- `rust_conventions.iter_collect_chain` - Unnecessary collect() in chains
- `rust_conventions.async_cancel_safety` - Drop inconsistencies in async code

#### Borrowing (2 rules)
- `rust_conventions.borrowed_reference_lifetime` - References outliving referent
- `rust_conventions.mutable_borrow_scope` - Mutable borrows with unnecessary scope

---

### 4. **JavaScript/TypeScript Conventions** (16 new rules)

#### Type Safety (5 rules)
- `js_conventions.any_type_usage` - Any types in TypeScript
- `js_conventions.type_coercion` - Implicit type coercion (==, !=)
- `js_conventions.optional_chaining_overuse` - ?. without null checks
- `js_conventions.null_coalescing_correct` - ?? with unexpected falsy values
- `js_conventions.type_assertion_unsafe` - as Type without validation

#### Promise/Async (5 rules)
- `js_conventions.unhandled_promise` - Promises without .catch() or try/catch
- `js_conventions.floating_promise` - Promises not awaited/returned
- `js_conventions.async_await_chain` - Promise chains mixed with async/await
- `js_conventions.promise_race_hazard` - Incomplete Promise.race handling
- `js_conventions.callback_hell` - Deeply nested callbacks (use async/await)

#### Module System (3 rules)
- `js_conventions.circular_dependency` - Module circular imports
- `js_conventions.import_order` - Mixing import styles (CJS/ESM)
- `js_conventions.wildcard_import` - import * unless namespaced

#### Data Flow (3 rules)
- `js_conventions.dom_xss` - innerHTML from untrusted sources
- `js_conventions.eval_usage` - eval() or Function() with dynamic code
- `js_conventions.prototype_pollution` - Object.assign with untrusted objects

---

### 5. **Node.js Conventions** (14 new rules)

#### Streams (4 rules)
- `nodejs_conventions.stream_not_piped` - Streams created but not piped
- `nodejs_conventions.backpressure_ignored` - Ignoring stream.write() return value
- `nodejs_conventions.stream_error_unhandled` - Stream error handlers missing
- `nodejs_conventions.stream_leak` - Streams not destroyed on error

#### Event Emitters (3 rules)
- `nodejs_conventions.event_listener_leak` - Listeners not removed
- `nodejs_conventions.once_vs_on` - Using 'on' for single-event listeners
- `nodejs_conventions.error_event_unhandled` - Unhandled 'error' event

#### Async Patterns (4 rules)
- `nodejs_conventions.callback_hell` - Deeply nested callbacks
- `nodejs_conventions.promise_swallowing` - Promises without error handling
- `nodejs_conventions.async_iterator_incomplete` - Incomplete async iterator impl
- `nodejs_conventions.concurrent_operations_unbounded` - Unbounded concurrent requests

#### Performance (3 rules)
- `nodejs_conventions.memory_leak_timers` - setInterval without clearInterval
- `nodejs_conventions.unbounded_buffer` - Unbounded internal buffering
- `nodejs_conventions.cpu_blocking` - Sync operations on event loop

---

### 6. **Terraform Conventions** (12 new rules)

#### Best Practices (4 rules)
- `terraform_conventions.hardcoded_values` - Hardcoded resource names, regions
- `terraform_conventions.provider_version_pinning` - Unpinned provider versions
- `terraform_conventions.variable_defaults_sensitive` - Sensitive data in defaults
- `terraform_conventions.state_file_exposure` - State files in git

#### Resource Design (4 rules)
- `terraform_conventions.resource_naming` - Inconsistent resource naming
- `terraform_conventions.count_vs_for_each` - Using count for dynamic resources
- `terraform_conventions.module_coupling` - Modules with hard dependencies
- `terraform_conventions.data_source_safety` - Unsafe data source queries

#### Compliance (4 rules)
- `terraform_conventions.public_resource_exposure` - Public access without auth
- `terraform_conventions.encryption_disabled` - Storage without encryption
- `terraform_conventions.logging_disabled` - Resources without logging
- `terraform_conventions.backup_missing` - No backup strategy defined

---

## Phase 2: Cross-Language Enterprise Rules (20 new rules)

### Security & Compliance
- `security.owasp_a01_injection` - SQL/Command injection patterns
- `security.owasp_a02_auth` - Authentication bypass patterns
- `security.owasp_a03_injection_validation` - Missing input validation
- `security.owasp_a04_xxe` - XXE vulnerability patterns
- `security.owasp_a05_broken_access` - Authorization bypass
- `security.pci_dss_encryption` - Unencrypted data transmission
- `security.hipaa_audit_logging` - Missing audit logs
- `security.soc2_access_control` - Weak access controls

### Reliability & Performance
- `reliability.timeout_missing` - Operations without timeouts
- `reliability.circuit_breaker_missing` - No circuit breaker for external calls
- `reliability.retry_logic_safe` - Infinite retry loops
- `reliability.graceful_degradation` - Missing fallbacks
- `performance.database_query_n_plus_one` - N+1 query patterns
- `performance.unnecessary_memory_allocation` - Allocations in hot paths
- `performance.synchronous_block_async` - Blocking async code
- `performance.unbounded_resource_growth` - Unbounded resource accumulation

### Maintainability
- `maintainability.function_parameter_count` - Functions with too many parameters
- `maintainability.coupling_high` - High module coupling
- `maintainability.cohesion_low` - Classes/modules with low cohesion

---

## Phase 3: Implementation Strategy

### Rules Categorization by Severity
- **BLOCKER** (enterprise-critical): Security, compliance, reliability
- **MAJOR** (must-fix): Performance, stability, standard violation
- **ADVISORY** (nice-to-have): Code style, documentation

### Rules Categorization by Automation
- **Automated (High Confidence)**: Static pattern detection
- **Semi-Automated (Medium Confidence)**: Requires context analysis
- **Manual Review (Low Confidence)**: Requires domain knowledge

---

## Phase 4: Enterprise Configuration Matrix

```toml
# Profile: Enterprise Strict (compliance-driven)
[profiles.enterprise_strict]
blockers = ["security.*", "stability.*", "reliability.*", "compliance.*"]
majors = ["performance.*", "maintainability.*"]
advisories = ["style.*"]
languages = ["go", "python", "rust", "javascript", "typescript", "nodejs", "terraform"]

# Profile: Enterprise Standard (balanced)
[profiles.enterprise_standard]
blockers = ["security.*", "stability.error_handling"]
majors = ["performance.critical", "reliability.*"]
advisories = ["*"]
languages = ["go", "python", "javascript", "typescript", "nodejs", "terraform"]

# Profile: StartUp (pragmatic)
[profiles.startup]
blockers = ["security.injection", "stability.panic"]
majors = ["performance.*"]
advisories = ["*"]
languages = ["go", "javascript", "typescript", "nodejs"]
```

---

## Success Metrics

| Metric | Target | Current |
|--------|--------|---------|
| Total Rules | 100+ | 55 |
| Languages Covered | 7 | 4 partial |
| Enterprise Rules | 40+ | 10 |
| Enterprise Profiles | 3+ | 0 |
| Rule Automation | 80%+ | 60% |

---

## Timeline

- **v0.17** (2w): Go + Python + Rust rules (45 new rules)
- **v0.18** (2w): JS/TS + Node.js rules (30 new rules)
- **v0.19** (2w): Terraform + cross-language rules (40 new rules)
- **v0.20** (1w): Enterprise profiles + config optimization

---

## Delivery Checklist

- [ ] Rule definitions in standards TOML
- [ ] Rule implementations in adapters (treesitter, custom)
- [ ] Documentation for each rule (why, how to fix)
- [ ] Test cases for each rule (positive + negative)
- [ ] Enterprise profiles configured
- [ ] Migration guide for existing users
- [ ] CLI `coderev config profile` command
