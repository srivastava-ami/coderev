# coderev — Architecture

## The problem

AI coding agents (Claude Code, Copilot, Cursor) now write most of the code in fast-moving teams. Two things broke:

**1. Standards enforcement disappeared.** Teams write rules in `CLAUDE.md` / `AGENTS.md` — but those are context, not enforcement. Agents comply roughly 70% of the time. There is no violation list, no severity, no gate.

**2. Code review doesn't scale.** Human reviewers catch different things each time. A `CLAUDE.md` rule that one reviewer enforces strictly becomes invisible to another. Consistency requires tooling, not people.

The result: code that passes TypeScript's type checker and looks fine in a diff silently violates architectural boundaries, embeds secrets, accumulates cyclomatic complexity, or skips structured logging — and no one knows until production.

---

## What coderev does

`coderev` is a single binary that reads a `code_review_standards.toml` file and statically analyses a codebase against every rule defined in it. It produces a report (Markdown, HTML, or SARIF) and exits with code `1` when blockers are found.

**Design constraints that drove every decision:**

- **No LLM at runtime.** Analysis is deterministic, reproducible, and costs nothing to run. The same repo scanned twice gives the same findings.
- **No server, no network.** Everything runs locally. External scanners (gitleaks, semgrep, madge) run as subprocesses — they are never remote calls.
- **Standards live in git.** The `code_review_standards.toml` is committed alongside the code. Every rule change is a pull request. Every exception is tracked with a justification and expiry date.
- **One binary, zero setup.** Built-in defaults are embedded in the binary. A fresh clone can be scanned with `coderev .` — no config file required.

---

## Architecture

### The pipeline

```
coderev [target]
    │
    ├─ 1. Resolve standards
    │      --standards flag → target dir → ~/.config/coderev/ → embedded defaults
    │      (binary never fails on missing standards — always has a fallback)
    │
    ├─ 2. Auto-install external tools (toolmgr)
    │      ensure gitleaks, semgrep, madge are available in ~/.coderev/tools/
    │      downloads static binaries (gitleaks) or uses pipx/brew/npm (semgrep, madge)
    │      warns on failure, never fails the scan — degraded coverage is not a hard error
    │
    ├─ 3. Discover plugins
    │      scan ~/.config/coderev/plugins/ (or --plugin-dir) for *-plugin.toml manifests
    │      resolve plugin binaries via $PATH; register into in-memory registry
    │
    ├─ 4. Walk target directory
    │      classify files by language; apply skip rules (node_modules, vendor, …)
    │      in --diff mode: DiffService.ChangedFiles() filters to changed files
    │
    ├─ 5. Run adapters in parallel
    │       ├─ treesitter  →  AST-based: 55 rules across TS/JS/Go/Python/Rust
    │       ├─ semgrep     →  OWASP injection / auth / crypto          (if installed)
    │       ├─ gitleaks    →  secret scanning                          (if installed)
    │       ├─ madge       →  circular deps, NX boundaries             (if installed)
    │       ├─ npmaudit    →  vulnerable npm packages                  (if npm present)
    │       ├─ coverage    →  line coverage threshold                  (if report file exists)
    │       ├─ custom[*]   →  any NDJSON-emitting binary               (if configured)
    │       └─ plugins[*]  →  registered plugin binaries               (if installed)
    │
    ├─ 6. Merge + deduplicate findings  (key: Rule | File | Line)
    ├─ 7. Apply exceptions from standards file
    ├─ 8. Compute baseline delta  (▲ regressions / ▼ improvements vs last run)
    ├─ 9. Detect or synthesise architecture doc
    ├─ 10. Evaluate quality gate  (--gate or --json)
    │       compare finding counts against .coderev-gate.toml thresholds
    │       default gate: 0 blockers, 5 majors, 10 advisories, 20 total
    │
    └─ 11. Output
            ├─ markdown  →  coderev-report.md         (default)
            ├─ html      →  coderev-report.html        (--format html)
            ├─ sarif     →  coderev-report.sarif       (--format sarif → GitHub Code Scanning)
            ├─ json      →  stdout                     (--json, includes gate result)
            └─ gh PR     →  inline review comments     (--annotate-pr, via gh CLI)
```

### Ports (hexagonal architecture)

The domain lives entirely in `internal/analysis/`. Two ports define the boundary from domain → outer layers:

#### 1. `ToolAdapter` — analysis port

```go
type ToolAdapter interface {
    Name()         string
    IsAvailable()  bool
    Capabilities() []string
    Run(ctx context.Context, req RunRequest) ([]Finding, error)
}
```

All domain types (`Standards`, `ToolConfig`, `Exception`, `GateConfig`, `Finding`, `FileInfo`) are defined in `internal/analysis/` — the TOML deserialization in `internal/config/` imports the domain, never the reverse. Adapters (`treesitter`, `gitleaks`, etc.) import only `analysis` types and implement `ToolAdapter`.

#### 2. `DiffService` — SCM port

```go
type DiffService interface {
    ChangedFiles(target, baseRef string) (map[string]bool, error)
}
```

The concrete `gitDiffService` lives in `cmd/coderev/` (the composition root). The domain never shells out to git — it calls the interface.

```
                    ┌──────────────────────────────────┐
                    │   cmd/coderev/ (composition root) │
                    └──────┬───────────────────────────┘
          ┌──────────────┬─┼───────────┬───────────┐
          ▼              ▼ ▼           ▼           ▼
  ┌──────────────┐  ┌──────────┐ ┌──────────┐ ┌──────────┐
  │ adapters/*   │  │toolmgr/  │ │ config/  │ │ report/  │
  │ (driven)     │  │(infra)   │ │(TOML)    │ │ quality/ │
  └──────┬───────┘  └──────────┘ └────┬─────┘ └────┬─────┘
         │  imports                   │  imports   │  imports
         ▼                            ▼            ▼
  ┌────────────────────────────────────────────────────┐
  │           internal/analysis/ ★ DOMAIN              │
  │  (ToolAdapter port, DiffService port, types)       │
  └────────────────────────────────────────────────────┘
         ▲
         │  never imports infra
         │
  all other packages
```

Dependencies always point **inward**: adapters → domain, config → domain, report → domain. Domain imports nothing outside stdlib.

### The adapter boundary

```go
type ToolAdapter interface {
    Name()         string
    IsAvailable()  bool        // false → skipped gracefully, never a hard failure
    Capabilities() []string    // rule IDs this adapter handles
    Run(ctx context.Context, req RunRequest) ([]Finding, error)
}
```

Every scanner — tree-sitter, semgrep, gitleaks, madge, coverage — implements this. Nothing else in the codebase cares which tools are installed or how many. Adding a new tool means implementing four methods and wiring it in `cmd/coderev/adapters.go`.

For tools that emit NDJSON output, no Go is needed at all — the `script` adapter bridges any external binary via `tool_config.toml`.

### Tree-sitter as the primary engine

The majority of rules are satisfied by tree-sitter running in-process (pure Go / CGO). It parses source files from text alone — no running build, no language server, no network. Supported: TypeScript, TSX, JavaScript, Go, **Python**, **Rust** (5 languages, 55 built-in rules).

**Cross-cutting rules (all languages):** cyclomatic complexity, cognitive complexity, function length, parameter count, nesting depth, hardcoded URLs, magic number literals, cross-file duplication.

**TypeScript / JavaScript:** `any` type, empty catch, `eval`, `innerHTML`, weak crypto, `__proto__` pollution, non-null assertions, await-in-loop, floating promises, commented-out code, TODO format, NX deep imports.

**Go:** `fmt.Print` detection, `panic` in libraries, SQL string concatenation, `context.TODO` usage, `defer` inside loops.

**Python:** `print()` detection, bare `except:`, `eval()`/`exec()`, SQL injection via string concat, `subprocess(..., shell=True)`, mutable default arguments, wildcard imports.

**Rust:** `.unwrap()`, `panic!()`, `.expect()`, `unsafe { }` blocks, `transmute`, `.clone()` on copy types, `todo!()` / `unimplemented!()`, `dbg!()` macro.

External adapters cover what tree-sitter cannot: secret scanning (gitleaks), OWASP injection patterns (semgrep), dependency CVEs (npm audit), circular imports (madge).

### Tool Manager (auto-install)

External scanners (gitleaks, semgrep, madge) are automatically downloaded on first scan via `internal/toolmgr/`. Tools are stored in `~/.coderev/tools/` — user-scoped, no sudo, clean uninstall by removing the directory.

| Tool | Install strategy | Download source |
|---|---|---|
| gitleaks | Static binary from GitHub release | `github.com/gitleaks/gitleaks/releases/` |
| semgrep | `pipx` → `pip3` → `brew` → Linux static binary | PyPI / GitHub |
| madge | `npm install --prefix` to tool cache | npm registry |

The composition root (`cmd/coderev/main.go`) calls `toolmgr.EnsureAll()` at scan start. Toolmgr never blocks — if a tool cannot be installed, it logs a warning and the adapter reports `IsAvailable() == false`. `internal/toolmgr/` is infrastructure (it shells out, downloads, writes to disk); it is never imported by the domain.

### Standards configuration

All thresholds and rules are defined in `code_review_standards.toml`. The binary contains embedded defaults so any repo is scannable out of the box. Teams override by placing their own file in the repo root.

Exceptions are first-class: each exception carries the rule, the file, a justification, an approver, and an expiry date. They are audited in the report.

---

## Usage patterns

### 1. Manual — single developer

```bash
# scan the whole repo
coderev .

# scan only what changed since main (fast, focused)
coderev --diff main .

# get an interactive HTML report
coderev --format html .
open coderev-report.html
```

### 2. PR review workflow

```bash
gh pr checkout 42
coderev --annotate-pr --diff main .
```

coderev posts a comment directly on the exact line of every blocker and major finding. Auto-detects repo slug and PR number from git context; override with `--repo` / `--pr` if needed.

### 3. CI gate (GitHub Actions)

```yaml
- name: Run coderev
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    PR_REPO: ${{ github.repository }}
    PR_NUMBER: ${{ github.event.pull_request.number }}
    BASE_REF: ${{ github.base_ref }}
  run: |
    docker run --rm -v "$GITHUB_WORKSPACE:/src" -e GH_TOKEN \
      ghcr.io/srivastava-ami/coderev:latest \
      --diff "origin/${BASE_REF}" --annotate-pr \
      --repo "${PR_REPO}" --pr "${PR_NUMBER}" /src
```

Exit code `1` on blockers blocks the merge.

### 4. With AI agents — zero token cost

coderev runs as an independent shell process. The AI agent triggers it and reads the output file — the analysis itself consumes no agent tokens and makes no LLM calls.

**In CI / hooks (fully automated — agent never intervenes):**

The pre-commit hook runs `coderev --diff HEAD .` on every commit. If blockers are found, the commit is rejected with the report path. The agent never sees the findings unless it reads the report file intentionally.

**Agent reads the report on demand:**

```bash
# agent runs this as a shell command
coderev --diff main . --output /tmp/coderev-findings.md

# agent then reads the file — one read, structured findings, no token streaming
```

The Markdown report is designed to be machine-parseable: findings are grouped by severity, each finding has a rule ID, file path, line number, and remediation text. An agent can read the whole report in one file read and act on it directly.

**In CLAUDE.md / AGENTS.md (makes the agent run it automatically):**

```markdown
## Quality gate
Before every commit: run `coderev .`
Report: `coderev-report.md`
Fix all blockers before pushing. Advisory findings must be addressed or justified.
```

This wires coderev into the agent's behaviour without any LLM involvement in the analysis phase — the agent runs a shell command, reads a file, acts on structured output.

---

## Why not X

**Why not SonarQube / CodeClimate?** They require a running server, remote code upload, and per-seat licences. coderev is a binary that runs in a CI job or on a developer laptop with no infrastructure.

**Why not ESLint / Biome / Ruff?** These are per-language tools. A polyglot repo (TypeScript + Go + Python) needs three separate configs and three separate CI steps, with no unified report. coderev produces one report across all languages.

**Why not a CodeRabbit / code review AI?** LLM-based review is non-deterministic — the same PR gets different findings each run. It cannot be used as a hard CI gate. coderev's rule-based engine gives the same result every time, making it suitable for blocking merges.

**Why not LSP?** LSP requires a working build environment and a running language server per language. Tree-sitter parses from source text alone — zero build, zero server, works on any file including partial or broken code.
