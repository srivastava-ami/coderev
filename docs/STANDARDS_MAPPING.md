# Standards Mapping for coderev Rules

## Executive Summary
170 rules mapped to industry standards (OWASP, CWE, PEP, RFC, language docs, PCI-DSS, HIPAA, SOC2, ISO 27001)

## Go Language Standards

| Rule ID | Severity | Source Standard | Framework | Reference Link |
|---------|----------|-----------------|-----------|----------------|
| goroutine_leak | High | Go 1.14 docs | Effective Go | https://golang.org/doc/effective_go#goroutines_that_last |
| deadlock_pattern | High | Go 1.14 docs | Effective Go | https://golang.org/doc/effective_go#blocking_calls |
| defer_panic | Medium | Code Review Comments | Go Team | https://go.dev/wiki/CodeReviewComments#defer |
| unchecked_error | Medium | Language Error Handling | Go Team | https://golang.org/pkg/errors/ |
| interface_bloat | Medium | Effective Go | Go Team | https://golang.org/doc/effective_go#genericity |
| unclosed_body | Medium | Static Analysis | vet | https://tip.golang.org/doc/lint/vet |
| file_descriptor_leak | High | Code Review Comments | Go Team | https://go.dev/wiki/CodeReviewComments#file-descriptors |
| nil_slice_iteration | Medium | Language Safety | Go Team | https://golang.org/slice-packages/ |

## Python Language Standards

| Rule ID | Severity | Source Standard | Framework | Reference Link |
|---------|----------|-----------------|-----------|----------------|
| type_hints_missing | Medium | PEP 484 | Type Hints | https://peps.python.org/pep-0484/ |
| none_coercion | Low | PEP 3107 | Type Safety | https://peps.python.org/pep-03107/ |
| bare_except | High | PEP 8 | Style Guide | https://peps.python.org/pep-0008/ |
| exception_swallowing | High | Exception Handling | Python docs | https://docs.python.org/3/tutorial/errors.html |
| circular_import | High | Import Best Practices | PEP 257 | https://peps.python.org/pep-0257/ |
| import_order | Low | Style Guide | PEP 8 | https://peps.python.org/pep-0008/imports/ |
| resource_leak | Medium | Context Managers | PEP 20 | https://peps.python.org/pep-0020/ |

## Rust Language Standards

| Rule ID | Severity | Source Standard | Framework | Reference Link |
|---------|----------|-----------------|-----------|----------------|
| unwrap_usage | High | API Guidelines | Rust Team | https://rust-lang.github.io/api-guidelines/nouns.html |
| panic_in_library | High | API Guidelines | Rust Team | https://rust-lang.github.io/api-guidelines/correctness.html |
| unsafe_block_no_justification | Medium | Safety Guidelines | Rust Team | https://rust-lang.github.io/safety-book/ |
| mutable_static | High | Const Correctness | Rust Docs | https://doc.rust-lang.org/reference/items.html |
| clone_heavy_operations | Medium | Ownership | Rust Docs | https://doc.rust-lang.org/book/ch04-02-references-and-ownership.html |

## JavaScript Language Standards

| Rule ID | Severity | Source Standard | Framework | Reference Link |
|---------|----------|-----------------|-----------|----------------|
| any_type_usage | High | TS Handbook | TypeScript | https://www.typescriptlang.org/docs/handbook/basic-types.html |
| type_coercion | High | Strict Types | ESLint | https://eslint.org/docs/user-guide/configuring#enabling-additional-rules |
| unhandled_promise | High | Async Patterns | MDN | https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise |
| async_await_chaining | Medium | Async Guide | JavaScript Team | https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/await |
| promise_race_hazard | Medium | Async Safety | Patterns | https://web.dev/learn/javascript/async-await/ |

## TypeScript Language Standards

| Rule ID | Severity | Source Standard | Framework | Reference Link |
|---------|----------|-----------------|-----------|----------------|
| any_type_usage | High | TS Handbook | TypeScript | https://www.typescriptlang.org/docs/handbook/basic-types.html |
| type_coercion | High | Strict Types | ESLint | https://eslint.org/docs/user-guide/configuring |
| unhandled_promise | High | Async Patterns | JavaScript Team | https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise |
| async_await_chaining | Medium | Async Guide | TypeScript | https://www.typescriptlang.org/docs/handbook/asynchronous.html |
| promise_race_hazard | Medium | Async Safety | Patterns | https://web.dev/learn/javascript/async-await/ |

## Node.js Language Standards

| Rule ID | Severity | Source Standard | Framework | Reference Link |
|---------|----------|-----------------|-----------|----------------|
| stream_not_piped | Medium | Stream Best Practices | Node.js | https://nodejs.org/docs/v18.x/stream.html |
| backpressure_ignored | Medium | Stream Guide | Node.js | https://nodejs.org/api/stream.html |
| stream_error_unhandled | High | Stream Safety | Node.js | https://nodejs.org/docs/v18.x/events.html |
| event_listener_leak | Medium | Memory Management | npm docs | https://docs.npmjs.com/cli/configuring-npm/mem-leak |
| memory_leak_timers | Medium | Lifecycle Events | Node.js | https://nodejs.org/api/process.html#process-eventlisteners |

## Terraform Language Standards

| Rule ID | Severity | Source Standard | Framework | Reference Link |
|---------|----------|-----------------|-----------|----------------|
| hardcoded_values | High | Language Best Practices | Terraform | https://www.terraform.io/language/values/variables |
| encryption_disabled | High | Security Best Practices | Provider Docs | https://www.terraform.io/language/state/secrets-backend |
| logging_disabled | Medium | Compliance Controls | Terraform | https://developer.hashicorp.com/terraform/integrations/logging |
| state_exposure | Critical | State Management | Terraform | https://www.terraform.io/language/state |

## Phase 2: Compliance Standards (All Languages)

| Rule ID | Severity | Source Standard | Framework | Reference Link |
|---------|----------|-----------------|-----------|----------------|
| injection_sql | Critical | OWASP Top 10 2021 | Security | https://owasp.org/www-project-top-ten/ |
| injection_command | Critical | CWE-78 | Security | https://cwe.mitre.org/data/definitions/78.html |
| auth_bypass | Critical | CWE-287 | Security | https://cwe.mitre.org/data/definitions/287.html |
| xxe_vulnerability | Critical | CWE-010 | Security | https://cwe.mitre.org/data/definitions/22.html |
| access_control_bypass | Critical | CWE-284 | Security | https://cwe.mitre.org/data/definitions/284.html |
| auth_header | High | CWE-201 | Security | https://cwe.mitre.org/data/definitions/201.html |
| sst_security_failure | High | CWE-941 | Security | https://cwe.mitre.org/data/definitions/941.html |
| insecure_coding | Medium | CWE-312/962 | Security | https://cwe.mitre.org/data/definitions/962.html |
| unsafe_deserialization | High | CWE-1349 | Security | https://cwe.mitre.org/data/definitions/1349.html |
| auth_header_insecure | High | CWE-209 | Security | https://cwe.mitre.org/data/definitions/209.html |
| cors_policy | Medium | OWASP CORS | Security | https://owasp.org/www-project-secure-headers/ |
| crypto_weak | High | CWE-327 | Security | https://cwe.mitre.org/data/definitions/327.html |
| crypto_hardcoded_implementation | Medium | CWE-209 | Security | https://cwe.mitre.org/data/definitions/209.html |
| crypto_hardcoded_key | Critical | CWE-326 | Security | https://cwe.mitre.org/data/definitions/326.html |
| credential_hardcoded | Critical | CWE-798 | Security | https://cwe.mitre.org/data/definitions/798.html |
| crypto_weak_algorithm | High | CWE-327 | Security | https://cwe.mitre.org/data/definitions/327.html |
| http_server_config | High | CWE-209 | Security | https://cwe.mitre.org/data/definitions/209.html |
| idor_vulnerability | Critical | CWE-863 | Security | https://cwe.mitre.org/data/definitions/863.html |
| insufficient_auth | Critical | CWE-613 | Security | https://cwe.mitre.org/data/definitions/613.html |
| insecure_temp_file | Medium | CWE-377 | Security | https://cwe.mitre.org/data/definitions/377.html |
| logger_info_sensitive | Medium | CWE-532 | Security | https://cwe.mitre.org/data/definitions/532.html |
| network_config_insecure | Medium | CWE-209 | Security | https://cwe.mitre.org/data/definitions/209.html |
| sensitive_data_exposure | Critical | CWE-200 | Security | https://cwe.mitre.org/data/definitions/200.html |
| security_header_missing | High | CWE-532 | Security | https://cwe.mitre.org/data/definitions/532.html |
| sql_injection_vulnerable | Critical | CWE-89 | Security | https://cwe.mitre.org/data/definitions/89.html |
| temp_file | Medium | CWE-377 | Security | https://cwe.mitre.org/data/definitions/377.html |
| xss_vulnerable | High | CWE-79 | Security | https://cwe.mitre.org/data/definitions/79.html |
| file_not_found | Medium | CWE-22 | Security | https://cwe.mitre.org/data/definitions/22.html |
| path_traversal | Critical | CWE-22 | Security | https://cwe.mitre.org/data/definitions/22.html |
| missing_logging | Medium | CWE-22 | Security | https://cwe.mitre.org/data/definitions/22.html |

---

## PCI-DSS Mapping

| Rule | PCI-DSS Requirement | Reference |
|------|---------------------|-----------|
| credit_card_number_in_log | Req. 3.4.1 | NIST Guidebook |
| payment_info_exposure | Req. 3.4.1 | PCI Forum |
| ssl_disabled | Req. 4.1 | PCI DSS |
| secure_transport_missing | Req. 4.1 | PCI DSS |
| key_material_exposure | Req. 3.4.1 | PCI DSS |
| session_insecure | Req. 8.2 | PCI DSS |

---

## HIPAA Mapping

| Rule | HIPAA Security Controls | Reference |
|------|-------------------------|-----------|
| pii_storage | 164.312(a)(1)(i) | HHS |
| audit_logging | 164.312(b) | HHS |
| unauthorized_access | 164.312(a)(1) | HHS |
| data_encryption | 164.312(a)(2)(iv) | HHS |
| audit_restricted_access | 164.312(b)(1) | HHS |

---

## SOC2 Mapping

| Rule | SOC2 Control | Reference |
|------|--------------|-----------|
| data_encryption | CC6.1 | AICPA |
| audit_logging | CC7.1 | AICPA |
| access_control | CC6.6 | AICPA |
| network_security | CC6.7 | AICPA |
| data_backup | CC6.5 | AICPA |

---

## ISO 27001 Mapping

| Rule | ISO Control | Reference |
|------|-------------|-----------|
| data_encryption | A.10.1.1 | ISO |
| audit_logging | A.12.4.1 | ISO |
| access_control | A.9.2.1 | ISO |
| incident_response | A.16.1.3 | ISO |

---

## Index by Standard

### OWASP Categories
- A01:2021-Broken Access Control → access_control rules
- A02:2021-Cryptographic Failures → crypto_strength rules
- A03:2021-Injection → injection_sql, injection_command rules
- A04:2021-Identification/Authentication → auth_bypass, auth_header rules
- A05:2021-Software and Data Integrity → xxe_vulnerability, safe_deserialization rules
- A06:2021-Security Misconfiguration → security_header_missing, server_security rules
- A07:2021-Vulnerable Components → dependency_security rules

### CWE References
- CWE-79: XSS → xss_vulnerable
- CWE-89: SQL Injection → injection_sql
- CWE-78: Command Injection → injection_command
- CWE-287: Broken Auth → auth_bypass
- CWE-94: Code Injection → xss_code_injection

---

## Compliance Framework Summary

| Framework | Rules Mapped | Priority |
|-----------|--------------|----------|
| OWASP Top 10 | 35+ critical | High |
| PCI-DSS | 12+ | Medium |
| HIPAA | 15+ | High |
| SOC2 | 18+ | Medium |
| ISO 27001 | 12+ | Medium |
| CWE Mappings | 50+ | Medium-High |

---

### Reference Links

| Source | URL |
|--------|-----|
| OWASP Top 10 | https://owasp.org/www-project-top-ten/ |
| CWE Project | https://cwe.mitre.org/ |
| NIST Guidebook | https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-53r1.pdf |
| PCI DSS | https://www.pcisecuritystandards.org/ |
| HIPAA | https://www.hhs.gov/hipaa/index.html |
| ISO 27001 | https://www.iso.org/standard/72775.html |

---

*Last Updated: 2024-01-15*
