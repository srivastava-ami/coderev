# Tool Adapters

## Overview

coderev integrates analysis tools via the **ToolAdapter** interface. Each adapter wraps a tool (native or subprocess) and exposes a common interface:

```go
type ToolAdapter interface {
    Name() string
    IsAvailable() bool
    Run(ctx context.Context, req RunRequest) ([]Finding, error)
}
```

---

## Native Tools (Built-in, Always Available)

These are written in pure Go and ship as part of the binary. No external dependencies required.

| Tool | Purpose | LOC | Layer |
|------|---------|-----|-------|
| **treesitter** | AST parsing (TS/JS/Go/Python/Rust) | 2400 | Execution |
| **imports** | Circular dependency detection (Tarjan SCC) | 650 | Execution |
| **secrets** | Pattern + entropy-based secret scanning | 268 | Execution |
| **depcve** | Offline CVE matching (cached OSV snapshot) | 396 | Execution |

### Strategy

These are the **DEFAULT** for analysis. They always work. No configuration needed.

**Use these for your quality gate** — they ship with every coderev binary and never fail due to missing binaries.

---

## External Tools (Optional, Subprocess Wrappers)

These require an external binary installed on the system. Gracefully skipped if the binary is not found.

| Tool | Purpose | Replaces | Status |
|------|---------|----------|--------|
| **gitleaks** | Git history secret scanning | native/secrets | Optional enrichment |
| **semgrep** | AST-based pattern matching (wider rule set) | native patterns in treesitter | Optional enrichment |
| **madge** | Node.js dependency graph analysis | native/imports | Optional (JS/TS only) |
| **npmaudit** | npm package vulnerability audit | native/depcve | Deprecated |
| **graphanalyze** | Architecture coupling & hotspot detection | (custom) | Optional |
| **script** | Custom shell script rules | (custom) | Opt-in via config |
| **coverage** | Code coverage threshold checking | (custom) | Opt-in via config |

### Strategy

Use these only if you need **extra depth or wider rule coverage**. If a binary is missing, that adapter is skipped silently; the scan continues with native tools.

---

## Graceful Degradation

When an external tool binary is not found:

1. The adapter's `IsAvailable()` returns false
2. `internal/analysis/runner.go` skips that adapter
3. Scan continues with remaining adapters
4. A warning is printed: `"gitleaks: binary not found — skipping"`
5. Report shows: `"⚠️ 1 adapter(s) skipped"`

**This is NOT a blocker** — your core analysis (via native tools) still runs and produces findings.

---

## Shared Utilities

All adapters can use these shared functions from `shared.go`:

- `BinaryAvailable(binary string) bool` — Check if a binary exists on $PATH or at a given path
- `RunTool(ctx context.Context, binary, name string, args []string) ([]byte, error)` — Execute a subprocess and capture output

Example:

```go
func (a *Adapter) IsAvailable() bool {
    return tools.BinaryAvailable("gitleaks")
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
    output, err := tools.RunTool(ctx, "gitleaks", "gitleaks", []string{"detect", "--json", ...})
    // Parse output and return findings
}
```

---

## Configuration

See `internal/config/default_tool_config.toml` for:

- Which adapter handles which rule
- Whether to use native (enabled) or external (disabled/skip) tools
- Fallback chains (e.g., try semgrep, fall back to native treesitter)

---

## Adding a New Tool

1. Create a new package under `internal/tools/<name>/`
2. Implement `ToolAdapter` interface:
   ```go
   type Adapter struct {
       binary string
   }
   
   func (a *Adapter) Name() string { return "my-tool" }
   func (a *Adapter) IsAvailable() bool { return BinaryAvailable(a.binary) }
   func (a *Adapter) Run(ctx context.Context, req RunRequest) ([]Finding, error) {
       // Execute tool, parse output, return findings
   }
   ```

3. Register in `cmd/coderev/adapters.go`: add a line to `buildAdapters()` instantiating your tool
4. Update `internal/config/default_tool_config.toml` to map rules to your adapter
5. Test: ensure `go build ./...` passes and `coderev . --config <your-config>` uses your adapter

---

## See Also

- `docs/architecture.md` — full 5-layer pipeline (governance → orchestration → execution → persistence → surface)
- `internal/config/` — standards and tool configuration
- `internal/analysis/runner.go` — how adapters are dispatched and findings are merged
