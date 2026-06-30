# Phase 1 Enterprise Rules Reference

**New in v0.17:** 95 enterprise-grade convention rules across 6 languages, expanding coderev from 55 to 150 total rules.

---

## Quick Overview

### Rule Count by Language

| Language | Core Rules | Phase 1 Rules | Total |
|----------|-----------|---------------|-------|
| Go | 5 | 20 | 25 |
| Python | 7 | 18 | 25 |
| Rust | 8 | 15 | 23 |
| JavaScript/TypeScript | 0 | 16 | 16 |
| Node.js | 0 | 14 | 14 |
| Terraform | 0 | 12 | 12 |
| **Totals** | **55** | **95** | **150** |

---

## Rule Categories

### Go (20 rules)

**Concurrency Safety** (6) — goroutine leaks, race conditions, deadlocks, channel safety, timeouts  
**Error Handling** (4) — unchecked errors, error wrapping, defer/panic ordering  
**Interface Design** (3) — bloat, segregation, receiver consistency  
**Resource Management** (4) — unclosed bodies, file descriptors, pool exhaustion, memory leaks  
**Nil Safety** (3) — pointer dereference, slice iteration, method calls  

### Python (18 rules)

**Type Safety** (5) — type hints, None handling, dynamic attributes, type consistency, duck typing  
**Async/Concurrency** (4) — unclosed resources, deadlocks, task leaks, event loop mismatches  
**Exception Handling** (4) — bare except, exception swallowing, exception chaining, finally side effects  
**Import Organization** (3) — circular imports, import order, unused imports  
**Memory/Resources** (2) — resource leaks, unbounded growth  

### Rust (15 rules)

**Memory Safety** (5) — unsafe blocks, panics, unwrap, lifetimes, mutable statics  
**Error Handling** (4) — error propagation, result discard, panic hooks, custom error impl  
**Patterns** (4) — clone heavy, expensive operations in loops, iterator chains, async safety  
**Borrowing** (2) — lifetime issues, mutable borrow scope  

### JavaScript/TypeScript (16 rules)

**Type Safety** (5) — any types, type coercion, optional chaining, null coalescing, type assertions  
**Promises/Async** (5) — unhandled promises, floating promises, async/await chains, race hazards, callback hell  
**Modules** (3) — circular dependencies, import order, wildcard imports  
**Data Flow Security** (3) — DOM XSS, eval usage, prototype pollution  

### Node.js (14 rules)

**Streams** (4) — not piped, backpressure ignored, error handlers, leaks  
**Event Emitters** (3) — listener leaks, once vs on, unhandled error events  
**Async Patterns** (4) — callback hell, promise swallowing, incomplete iterators, unbounded concurrency  
**Performance** (3) — timer leaks, unbounded buffering, CPU blocking  

### Terraform (12 rules)

**Best Practices** (4) — hardcoded values, provider pinning, sensitive defaults, state exposure  
**Resource Design** (4) — naming consistency, count vs for_each, module coupling, data source safety  
**Compliance** (4) — public exposure, encryption, logging, backups  

---

## Severity Levels

- **🔴 Blocker** (Enterprise-critical): Security, compliance, reliability, resource safety
- **🟡 Major** (Must-fix): Performance, stability, standard violations
- **🔵 Advisory** (Nice-to-have): Code style, design improvements

---

## Detection Method

All Phase 1 rules are detected via the **native treesitter adapter** — pure Go, zero external dependencies. Rules use:
- **Pattern matching** — text-based heuristics with high confidence
- **AST analysis** — tree-sitter parsing for syntax-aware detection
- **Context analysis** — multiline patterns for advanced detection

No Semgrep, gitleaks, or external tools required.

---

## Configuration

Enable Phase 1 rules in `tool_config.toml`:

```toml
[adapters.treesitter]
enabled = true
description = "Built-in: AST parsing via tree-sitter library"
rules = [
  # ... existing rules ...
  # Phase 1 Go conventions (20 rules)
  "go_conventions.goroutine_leak",
  "go_conventions.race_condition",
  "go_conventions.deadlock_pattern",
  # ... all 95 Phase 1 rules
]
```

Or use the built-in defaults (all Phase 1 rules enabled by default):

```bash
coderev .  # Automatically uses Phase 1 rules
```

---

## Examples

### Go: Detect Goroutine Leak

```go
// ❌ DETECTED: goroutine_leak
go func() {
  conn := newConnection()
  conn.Write(data)
  // missing: ctx cancellation or wait group
}()
```

**Remediation:** Use `context.Context` for cancellation or `sync.WaitGroup` for tracking.

### Python: Detect Unclosed Async Resource

```python
# ❌ DETECTED: unclosed_async_resource
async def fetch():
  session = aiohttp.ClientSession()  # Missing: async with
  return await session.get(url)
```

**Remediation:** Use `async with aiohttp.ClientSession() as session:`

### Rust: Detect Unsafe Block Without Comment

```rust
// ❌ DETECTED: unsafe_block_justification
unsafe {
  ptr.write(data);
}
```

**Remediation:** Add `// SAFETY: ...` comment explaining why unsafe is needed.

### Node.js: Detect Memory Leak (Timer)

```javascript
// ❌ DETECTED: memory_leak_timers
setInterval(() => {
  console.log('tick');
}, 1000);
// missing: const id = setInterval(...); clearInterval(id);
```

**Remediation:** Store timer ID and call `clearInterval()` when done.

### Terraform: Detect Unencrypted Storage

```hcl
# ❌ DETECTED: encryption_disabled
resource "aws_s3_bucket" "data" {
  bucket = "my-bucket"
  # missing: server_side_encryption_configuration
}
```

**Remediation:** Add encryption configuration block.

---

## Migration Guide

### For Existing Users

No action required. Phase 1 rules are enabled by default in v0.17.

**Impact:**
- Existing 55 rules continue working unchanged
- 95 new findings will appear in scans
- No breaking changes to APIs or configuration

### For CI/CD Gates

Update `.coderev-gate.toml` to account for new findings:

```toml
# Old gate (v0.16)
max_blockers = 0
max_majors = 5

# New gate (v0.17) — adjust for Phase 1 findings
max_blockers = 0
max_majors = 20  # Phase 1 adds ~15-20 major findings in typical repos
```

### For Custom Standards

If you're using custom standards files, add Phase 1 sections:

```toml
[go_conventions]
severity = "major"

[go_conventions.concurrency]
rule = "go_conventions.goroutine_leak"

# ... repeat for all Phase 1 rule categories
```

---

## Known Limitations (Phase 1)

1. **Context Propagation** (`go_conventions.context_propagation`): Disabled due to false positives on library calls. Will refine with full AST support in Phase 2.

2. **Multi-line Pattern Detection**: Some rules use heuristics instead of full AST analysis (Rust lifetime checks, Terraform module coupling). Manual review recommended.

3. **Language-Specific Tools**: Terraform/Infrastructure rules are HCL-only. Equivalent rules for other IaC languages (CloudFormation, Ansible) are Phase 2 work.

---

## What's Next (Phase 2)

- **20 Cross-Language Enterprise Rules** (OWASP, PCI-DSS, HIPAA, SOC2)
- **Refined AST Analysis** for multi-line patterns
- **Terraform Phase 2** (CloudFormation, Ansible, Pulumi)
- **Enterprise Profiles** (Strict, Standard, StartUp)

---

## Support & Feedback

Found an issue or have a suggestion? Open an issue on [GitHub](https://github.com/srivastava-ami/coderev/issues).

---

**v0.17 — Phase 1 Complete**  
95 enterprise rules • 6 languages • 150 total rules • Shipped
