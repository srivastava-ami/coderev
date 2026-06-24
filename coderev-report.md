# Code Review Report: nx-code-review-standard

> ❌ **FAIL** · 192 findings · 0 blocker(s) · 3 major · 189 advisory  
> Scanned **63 files** · Generated Wed, 24 Jun 2026 00:35:03 PDT  
> Standards: `/Users/amitsrivastava/Downloads/srivastava-ami/nx-code-review-standard/code_review_standards.toml` v2.2.0

> 📊 **Baseline saved** — future runs will track trends against these 192 findings.

---

## Summary

| Severity | Count |
|----------|-------|
| 🔴 Blocker | 0 |
| 🟡 Major | 3 |
| 🔵 Advisory | 189 |
| **Total** | **192** |

<details>
<summary>Findings by pillar</summary>

| Pillar | Count |
|--------|-------|
| observability | 63 |
| hardcoding | 78 |
| complexity | 37 |
| file_structure | 14 |

</details>

## Architecture

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
    │       ├─ treesitter  →  AST-based: 53 rules across TS/JS/Go/Python/Rust
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

The majority of rules are satisfied by tree-sitter running in-process (pure Go / CGO). It parses source files from text alone — no running build, no language server, no network. Supported: TypeScript, TSX, JavaScript, Go, **Python**, **Rust** (5 languages, 53 built-in rules).

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


## Findings by Pillar

<details>
<summary>🟡 <b>complexity</b> ![A](https://img.shields.io/badge/reliability-A-brightgreen) — 37 finding(s)</summary>

| File | Line | Rule | Severity | Message |
|------|------|------|----------|---------|
| `…code-review-standard/internal/adapters/treesitter/walker.go` | 88 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 8 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…a-ami/nx-code-review-standard/internal/config/toolconfig.go` | 29 | `complexity.function_length` | 🔵 advisory | function 'defaultToolConfig': 34 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `…vastava-ami/nx-code-review-standard/cmd/coderev/adapters.go` | 20 | `complexity.cyclomatic` | 🔵 advisory | function 'buildAdapters': cyclomatic complexity 8 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…ew-standard/internal/adapters/treesitter/walker_patterns.go` | 78 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…ew-standard/internal/adapters/treesitter/walker_patterns.go` | 121 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 21 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 7 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…va-ami/nx-code-review-standard/internal/report/generator.go` | 15 | `complexity.max_return_count` | 🔵 advisory | function 'Generate': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…stava-ami/nx-code-review-standard/internal/plugin/loader.go` | 24 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…stava-ami/nx-code-review-standard/internal/plugin/loader.go` | 24 | `complexity.max_return_count` | 🔵 advisory | function '<anonymous>': 6 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…i/nx-code-review-standard/internal/config/standards_test.go` | 65 | `complexity.cyclomatic` | 🔵 advisory | function 'TestLoadParsesComplexityThresholds': cyclomatic complexity 7 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…ava-ami/nx-code-review-standard/internal/plugin/manifest.go` | 21 | `complexity.max_return_count` | 🔵 advisory | function 'LoadManifest': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…ava-ami/nx-code-review-standard/internal/analysis/runner.go` | 118 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…ava-ami/nx-code-review-standard/internal/analysis/runner.go` | 118 | `complexity.max_return_count` | 🔵 advisory | function '<anonymous>': 6 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…ava-ami/nx-code-review-standard/internal/analysis/runner.go` | 110 | `complexity.function_length` | 🔵 advisory | function 'walkFiles': 36 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `…astava-ami/nx-code-review-standard/internal/quality/gate.go` | 42 | `complexity.max_return_count` | 🔵 advisory | function 'gateMessage': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…nx-code-review-standard/internal/report/markdown_helpers.go` | 57 | `complexity.cyclomatic` | 🔵 advisory | function 'ratingBadge': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…nx-code-review-standard/internal/report/markdown_helpers.go` | 57 | `complexity.max_return_count` | 🔵 advisory | function 'ratingBadge': 6 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…astava-ami/nx-code-review-standard/internal/report/model.go` | 117 | `complexity.max_return_count` | 🔵 advisory | function 'pillarRating': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…astava-ami/nx-code-review-standard/internal/report/model.go` | 165 | `complexity.cyclomatic` | 🔵 advisory | function 'buildPillar': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 20 | `complexity.function_length` | 🔵 advisory | function 'writeMarkdown': 33 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 140 | `complexity.cyclomatic` | 🔵 advisory | function 'writeExceptions': cyclomatic complexity 7 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/plugin.go` | 41 | `complexity.max_return_count` | 🔵 advisory | function 'runPluginInstall': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…view-standard/internal/adapters/treesitter/walker_python.go` | 63 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…/srivastava-ami/nx-code-review-standard/cmd/coderev/main.go` | 32 | `complexity.function_length` | 🔵 advisory | function 'main': 38 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 78 | `complexity.max_return_count` | 🔵 advisory | function 'runInstallHooks': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 117 | `complexity.cyclomatic` | 🔵 advisory | function 'postAnnotate': cyclomatic complexity 7 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 157 | `complexity.max_return_count` | 🔵 advisory | function 'resolveTarget': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 196 | `complexity.boolean_param_flag` | 🔵 advisory | function 'resolveOutputPath': boolean flag parameter 'flag, format string' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 122 | `complexity.max_return_count` | 🔵 advisory | function 'ensureSemgrep': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 140 | `complexity.cyclomatic` | 🔵 advisory | function 'ensureMadge': cyclomatic complexity 8 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 140 | `complexity.function_length` | 🔵 advisory | function 'ensureMadge': 32 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 140 | `complexity.max_return_count` | 🔵 advisory | function 'ensureMadge': 7 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 191 | `complexity.cyclomatic` | 🔵 advisory | function 'extractFromTGZ': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `…andard/internal/adapters/treesitter/walker_magic_numbers.go` | 74 | `complexity.max_return_count` | 🔵 advisory | function 'skipLine': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `…review-standard/internal/adapters/treesitter/duplication.go` | 46 | `complexity.boolean_param_flag` | 🔵 advisory | function 'indexFile': boolean flag parameter 'hashMap map[uint64][]dupLoc' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `…review-standard/internal/adapters/treesitter/duplication.go` | 58 | `complexity.boolean_param_flag` | 🔵 advisory | function 'emitDupFindings': boolean flag parameter 'hashMap map[uint64][]dupLoc' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `…e-review-standard/internal/adapters/treesitter/walker_go.go` | 205 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |

</details>

<details>
<summary>🟡 <b>file_structure</b> ![B](https://img.shields.io/badge/reliability-B-green) — 14 finding(s)</summary>

| File | Line | Rule | Severity | Message |
|------|------|------|----------|---------|
| `…ode-review-standard/internal/analysis/standards_sections.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 244 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…ava-ami/nx-code-review-standard/internal/analysis/runner.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 232 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…i/nx-code-review-standard/internal/architecture/detector.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 204 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…astava-ami/nx-code-review-standard/internal/report/sarif.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 177 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…astava-ami/nx-code-review-standard/internal/report/model.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 216 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 175 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…view-standard/internal/adapters/treesitter/walker_python.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 156 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 214 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 236 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…review-standard/internal/adapters/treesitter/duplication.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 177 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…e-review-standard/internal/adapters/treesitter/walker_go.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 233 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `…-code-review-standard/internal/adapters/gitleaks/adapter.go` | 27 | `file_structure.duplication` | 🟡 major | ~25-token block duplicated in /Users/amitsrivastava/Downloads/srivastava-ami/nx-code-review-standard/internal/adapters/madge/adapter.go:28 |
| | | | | 💡 *Extract the shared logic into a shared utility module.* |
| `…/nx-code-review-standard/internal/adapters/madge/adapter.go` | 28 | `file_structure.duplication` | 🟡 major | ~25-token block duplicated in /Users/amitsrivastava/Downloads/srivastava-ami/nx-code-review-standard/internal/adapters/npmaudit/adapter.go:28 |
| | | | | 💡 *Extract the shared logic into a shared utility module.* |
| `…-code-review-standard/internal/adapters/npmaudit/adapter.go` | 28 | `file_structure.duplication` | 🟡 major | ~25-token block duplicated in /Users/amitsrivastava/Downloads/srivastava-ami/nx-code-review-standard/internal/adapters/semgrep/adapter.go:27 |
| | | | | 💡 *Extract the shared logic into a shared utility module.* |

</details>

<details>
<summary>🟡 <b>hardcoding</b> ![A](https://img.shields.io/badge/reliability-A-brightgreen) — 78 finding(s)</summary>

| File | Line | Rule | Severity | Message |
|------|------|------|----------|---------|
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/plugin.go` | 56 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 0644 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/plugin.go` | 89 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 0755 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 20 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 37 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…-code-review-standard/internal/adapters/gitleaks/adapter.go` | 87 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 6 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…-code-review-standard/internal/adapters/gitleaks/adapter.go` | 90 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 3 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…ode-review-standard/internal/adapters/treesitter/adapter.go` | 52 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 8 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…review-standard/internal/adapters/treesitter/duplication.go` | 69 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…review-standard/internal/adapters/treesitter/walker_docs.go` | 68 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 4 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…e-review-standard/internal/adapters/treesitter/walker_go.go` | 132 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 9 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_helpers.go` | 74 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_helpers.go` | 75 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 24 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 8 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 28 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 12 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 32 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 5 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 54 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 10 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 67 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 30 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 73 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 10 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 84 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 3 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 97 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | 108 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 4 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…ew-standard/internal/adapters/treesitter/walker_security.go` | 53 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 256 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…w-standard/internal/adapters/treesitter/walker_stability.go` | 19 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 9 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…w-standard/internal/adapters/treesitter/walker_structure.go` | 15 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 250 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…w-standard/internal/adapters/treesitter/walker_structure.go` | 19 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 150 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…w-standard/internal/adapters/treesitter/walker_structure.go` | 41 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 120 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…va-ami/nx-code-review-standard/internal/analysis/finding.go` | 25 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…va-ami/nx-code-review-standard/internal/analysis/finding.go` | 26 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 7 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 8 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 15 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1121 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 16 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1121 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 17 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1121 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 18 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1121 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 19 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1121 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 20 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1121 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 21 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1121 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 26 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1041 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 27 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 1120 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 30 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 704 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 31 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 476 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 32 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 704 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 35 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 38 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 39 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 390 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 40 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 390 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 41 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 835 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 44 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 53 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 54 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 55 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 56 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 58 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 59 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 69 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 703 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 70 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 703 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 72 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 119 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 73 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 704 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 79 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 80 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 703 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 81 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 83 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 772 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 87 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 703 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 88 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 89 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | 90 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2021 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…vastava-ami/nx-code-review-standard/internal/config/gate.go` | 15 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 5 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…vastava-ami/nx-code-review-standard/internal/config/gate.go` | 16 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 10 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…vastava-ami/nx-code-review-standard/internal/config/gate.go` | 17 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 20 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…a-ami/nx-code-review-standard/internal/config/toolconfig.go` | 58 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 80.0 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 123 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 30 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…nx-code-review-standard/internal/report/markdown_helpers.go` | 24 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 5 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…nx-code-review-standard/internal/report/markdown_helpers.go` | 25 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 5 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…nx-code-review-standard/internal/report/markdown_helpers.go` | 33 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 9 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…astava-ami/nx-code-review-standard/internal/report/model.go` | 119 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 5 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…astava-ami/nx-code-review-standard/internal/report/model.go` | 121 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 3 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…astava-ami/nx-code-review-standard/internal/report/model.go` | 123 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 5 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |
| `…astava-ami/nx-code-review-standard/internal/report/model.go` | 125 | `hardcoding.magic_number` | 🔵 advisory | magic number literal: 2 — use a named constant |
| | | | | 💡 *Extract the literal into a const or enum with a descriptive name.* |

</details>

<details>
<summary>🟡 <b>observability</b> ![A](https://img.shields.io/badge/reliability-A-brightgreen) — 63 finding(s)</summary>

| File | Line | Rule | Severity | Message |
|------|------|------|----------|---------|
| `…mi/nx-code-review-standard/internal/output/ghpr/annotate.go` | 57 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…mi/nx-code-review-standard/internal/report/markdown_arch.go` | 16 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…mi/nx-code-review-standard/internal/report/markdown_arch.go` | 31 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…mi/nx-code-review-standard/internal/report/markdown_arch.go` | 49 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…mi/nx-code-review-standard/internal/report/markdown_arch.go` | 51 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…nx-code-review-standard/internal/report/markdown_helpers.go` | 81 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…nx-code-review-standard/internal/report/markdown_helpers.go` | 94 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 11 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 12 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 13 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 14 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 18 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 19 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 20 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 21 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 22 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 23 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 26 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 32 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | 33 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 26 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 27 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 34 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 36 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 38 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 51 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 57 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 58 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 59 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 60 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 65 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 83 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 85 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 97 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 100 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 112 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 120 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 160 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | 171 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/plugin.go` | 59 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/plugin.go` | 73 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/plugin.go` | 76 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/plugin.go` | 78 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…/srivastava-ami/nx-code-review-standard/cmd/coderev/main.go` | 73 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 88 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 95 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 96 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 103 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 104 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 111 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 131 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 134 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | 136 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 35 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 37 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 71 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 74 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 105 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 107 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | 141 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 57 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 60 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | 62 | `go.fmt_print` | 🔵 advisory | fmt.Println/Printf in production code — bypasses structured logging |
| | | | | 💡 *Use slog, zap, or zerolog with structured fields and log-level control.* |

</details>

## Hot Files

<details>
<summary>41 file(s) with findings</summary>

| File | Language | Lines | Findings | Heat |
|------|----------|-------|----------|------|
| `…/nx-code-review-standard/internal/analysis/rule_registry.go` | go | 94 | 38 | `█████` |
| `…ava-ami/nx-code-review-standard/internal/report/markdown.go` | go | 175 | 23 | `███░░` |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/output.go` | go | 35 | 13 | `█░░░░` |
| `…srivastava-ami/nx-code-review-standard/cmd/coderev/setup.go` | go | 139 | 12 | `█░░░░` |
| `…stava-ami/nx-code-review-standard/cmd/coderev/runhelpers.go` | go | 214 | 11 | `█░░░░` |
| `…iew-standard/internal/adapters/treesitter/walker_metrics.go` | go | 137 | 10 | `█░░░░` |
| `…ava-ami/nx-code-review-standard/internal/toolmgr/toolmgr.go` | go | 236 | 9 | `█░░░░` |
| `…rivastava-ami/nx-code-review-standard/cmd/coderev/plugin.go` | go | 102 | 7 | `░░░░░` |
| `…astava-ami/nx-code-review-standard/internal/report/model.go` | go | 216 | 7 | `░░░░░` |
| `…nx-code-review-standard/internal/report/markdown_helpers.go` | go | 97 | 7 | `░░░░░` |
| `…mi/nx-code-review-standard/internal/report/markdown_arch.go` | go | 62 | 4 | `░░░░░` |
| `…review-standard/internal/adapters/treesitter/duplication.go` | go | 177 | 4 | `░░░░░` |
| `…ava-ami/nx-code-review-standard/internal/analysis/runner.go` | go | 232 | 4 | `░░░░░` |
| `…e-review-standard/internal/adapters/treesitter/walker_go.go` | go | 233 | 3 | `░░░░░` |
| `…vastava-ami/nx-code-review-standard/internal/config/gate.go` | go | 32 | 3 | `░░░░░` |
| `…-code-review-standard/internal/adapters/gitleaks/adapter.go` | go | 92 | 3 | `░░░░░` |
| `…w-standard/internal/adapters/treesitter/walker_structure.go` | go | 50 | 3 | `░░░░░` |
| `…a-ami/nx-code-review-standard/internal/config/toolconfig.go` | go | 63 | 2 | `░░░░░` |
| `…stava-ami/nx-code-review-standard/internal/plugin/loader.go` | go | 46 | 2 | `░░░░░` |
| `…view-standard/internal/adapters/treesitter/walker_python.go` | go | 156 | 2 | `░░░░░` |
| `…iew-standard/internal/adapters/treesitter/walker_helpers.go` | go | 112 | 2 | `░░░░░` |
| `…va-ami/nx-code-review-standard/internal/analysis/finding.go` | go | 61 | 2 | `░░░░░` |
| `…/srivastava-ami/nx-code-review-standard/cmd/coderev/main.go` | go | 147 | 2 | `░░░░░` |
| `…ew-standard/internal/adapters/treesitter/walker_patterns.go` | go | 145 | 2 | `░░░░░` |
| `…i/nx-code-review-standard/internal/config/standards_test.go` | go | 147 | 1 | `░░░░░` |
| `…code-review-standard/internal/adapters/treesitter/walker.go` | go | 117 | 1 | `░░░░░` |
| `…astava-ami/nx-code-review-standard/internal/report/sarif.go` | go | 177 | 1 | `░░░░░` |
| `…ew-standard/internal/adapters/treesitter/walker_security.go` | go | 75 | 1 | `░░░░░` |
| `…w-standard/internal/adapters/treesitter/walker_stability.go` | go | 134 | 1 | `░░░░░` |
| `…/nx-code-review-standard/internal/adapters/madge/adapter.go` | go | 76 | 1 | `░░░░░` |

</details>

<details>
<summary>Exceptions / suppressions</summary>

| File / Module | Rule | Justification |
|---------------|------|---------------|
| `internal/report/sarif.go` | `hardcoding.urls_and_paths` | SARIF spec requires literal schema URIs — these are specification constants, not configurable URLs |
| `internal/adapters/treesitter/walker_security.go` | `security.no_weak_crypto` | Pattern strings are detection constants, not actual crypto calls — the checker flags its own rule list |
| `testdata` | `hardcoding.urls_and_paths` | testdata/sample-ts contains intentional violations to validate the scanner — not production code |
| `testdata` | `security.no_eval` | testdata/sample-ts intentional violations |
| `testdata` | `security.no_inner_html` | testdata/sample-ts intentional violations |
| `testdata` | `security.no_weak_crypto` | testdata/sample-ts intentional violations |
| `testdata` | `security.no_prototype_pollution` | testdata/sample-ts intentional violations |
| `testdata` | `type_safety.no_any` | testdata/sample-ts intentional violations |
| `testdata` | `type_safety.no_non_null_assertion` | testdata/sample-ts intentional violations |
| `testdata` | `type_safety.no_force_cast` | testdata/sample-ts intentional violations |
| `testdata` | `observability.logging` | testdata/sample-ts intentional violations |
| `testdata` | `stability.no_await_in_loop` | testdata/sample-ts intentional violations |
| `internal/toolmgr/toolmgr.go` | `hardcoding.urls_and_paths` | tool download URLs (GitHub releases for gitleaks/semgrep) and package install commands (npm) are infrastructure constants with no configurable alternative for a self-contained auto-installer |
| `internal/adapters/treesitter/walker_go.go` | `go.fmt_print` | Pattern strings ("fmt.Println(", etc.) in the detection array trigger the rule on the checker's own source — detection constants, not production print calls |
| `internal/adapters/treesitter/walker_go.go` | `go.panic_in_lib` | String literal "panic(" in the detection pattern and message text triggers the rule on its own source code |
| `internal/adapters/treesitter/walker_go.go` | `go.sql_string_concat` | SQL keyword strings in goSQLConcatPatterns and goLineHasSQLKeyword are detection constants — not user-facing SQL construction |
| `internal/adapters/treesitter/walker_go.go` | `go.context_todo` | The string "context.TODO()" appears in detection logic and message text — detection constants, not production code |
| `internal/adapters/treesitter/walker_go.go` | `go.defer_in_loop` | The string "defer " appears inside the state-machine for-loop as a string literal being searched — detection constant, not a deferred call |

</details>

<details>
<summary>⚠️ Adapter warnings</summary>

- **gitleaks**: gitleaks: exit status 1 — stderr: 
    ○
    │╲
    │ ○
    ○ ░
    ░    gitleaks

[90m12:34AM[0m [31mFTL[0m [1mReport path is not writable: /dev/stdout[0m [36merror=[0m[31m[1m"open /dev/stdout: permission denied"[0m[0m


</details>

---

*Generated by [coderev](https://github.com/srivastava-ami/coderev) · Wed, 24 Jun 2026 00:35:03 PDT*
