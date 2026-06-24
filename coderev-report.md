# Code Review Report: src

> ✅ **PASS** · 40 findings · 0 blocker(s) · 0 major · 40 advisory  
> Scanned **46 files** · Generated Wed, 24 Jun 2026 03:34:24 UTC  
> Standards: `/src/code_review_standards.toml` v2.1.0

> 📊 **Baseline saved** — future runs will track trends against these 40 findings.

---

## Summary

| Severity | Count |
|----------|-------|
| 🔴 Blocker | 0 |
| 🟡 Major | 0 |
| 🔵 Advisory | 40 |
| **Total** | **40** |

<details>
<summary>Findings by pillar</summary>

| Pillar | Count |
|--------|-------|
| complexity | 32 |
| file_structure | 8 |

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
    ├─ 2. Walk target directory
    │      classify files by language; apply skip rules (node_modules, vendor, …)
    │      in --diff mode: intersect with `git diff --name-only <ref>`
    │
    ├─ 3. Run adapters in parallel
    │       ├─ treesitter  →  AST-based: complexity, type safety, patterns, security
    │       ├─ semgrep     →  OWASP injection / auth / crypto          (if installed)
    │       ├─ gitleaks    →  secret scanning                          (if installed)
    │       ├─ madge       →  circular deps, NX boundaries             (if installed)
    │       ├─ npmaudit    →  vulnerable npm packages                  (if npm present)
    │       ├─ coverage    →  line coverage threshold                  (if report file exists)
    │       └─ custom[*]  →  any NDJSON-emitting binary               (if configured)
    │
    ├─ 4. Merge + deduplicate findings  (key: Rule | File | Line)
    ├─ 5. Apply exceptions from standards file
    ├─ 6. Compute baseline delta  (▲ regressions / ▼ improvements vs last run)
    ├─ 7. Detect or synthesise architecture doc
    │
    └─ 8. Output
            ├─ markdown  →  coderev-report.md         (default)
            ├─ html      →  coderev-report.html        (--format html)
            ├─ sarif     →  coderev-report.sarif       (--format sarif → GitHub Code Scanning)
            └─ gh PR     →  inline review comments     (--annotate-pr, via gh CLI)
```

### The adapter boundary

The entire tool is built around one interface:

```go
type ToolAdapter interface {
    Name()         string
    IsAvailable()  bool        // false → skipped gracefully, never a hard failure
    Capabilities() []string    // rule IDs this adapter handles
    Run(ctx context.Context, req RunRequest) ([]Finding, error)
}
```

Every scanner — tree-sitter, semgrep, gitleaks, madge, coverage — implements this. Nothing else in the codebase cares which tools are installed or how many. Adding a new tool means implementing four methods. Replacing a built-in means setting `enabled = false` in `tool_config.toml` and wiring a replacement in `buildAdapters()`.

For tools that emit NDJSON output, no Go is needed at all — the `script` adapter bridges any external binary directly via `tool_config.toml`.

### Tree-sitter as the primary engine

The majority of rules are satisfied by tree-sitter running in-process (pure Go / CGO). It parses source files from text alone — no running build, no language server, no network. Supported: TypeScript, TSX, JavaScript, Go, Python, Rust.

Rules it covers: cyclomatic complexity, cognitive complexity, function length, parameter count, nesting depth, `any` type, empty catch, hardcoded URLs, `eval`, `innerHTML`, weak crypto, `__proto__`, non-null assertions, await-in-loop, floating promises, commented-out code, TODO format, NX deep imports, cross-file duplication.

External adapters cover what tree-sitter cannot: secret scanning (gitleaks), OWASP injection patterns (semgrep), dependency CVEs (npm audit), circular imports (madge).

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
<summary>🟡 <b>complexity</b> ![A](https://img.shields.io/badge/reliability-A-brightgreen) — 32 finding(s)</summary>

| File | Line | Rule | Severity | Message |
|------|------|------|----------|---------|
| `/src/cmd/coderev/adapters.go` | 15 | `complexity.cyclomatic` | 🔵 advisory | function 'buildAdapters': cyclomatic complexity 8 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/cmd/coderev/main.go` | 34 | `complexity.function_length` | 🔵 advisory | function 'main': 35 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `/src/cmd/coderev/main.go` | 70 | `complexity.cyclomatic` | 🔵 advisory | function 'run': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/cmd/coderev/main.go` | 70 | `complexity.max_return_count` | 🔵 advisory | function 'run': 6 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/cmd/coderev/main.go` | 139 | `complexity.cyclomatic` | 🔵 advisory | function 'postAnnotate': cyclomatic complexity 7 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/cmd/coderev/main.go` | 179 | `complexity.max_return_count` | 🔵 advisory | function 'resolveTarget': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/cmd/coderev/main.go` | 221 | `complexity.boolean_param_flag` | 🔵 advisory | function 'resolveOutputPath': boolean flag parameter 'flag, format string' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `/src/cmd/coderev/setup.go` | 77 | `complexity.max_return_count` | 🔵 advisory | function 'runInstallHooks': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/cmd/coderev/setup.go` | 156 | `complexity.boolean_param_flag` | 🔵 advisory | function 'installGitleaks': boolean flag parameter 'hasBrew bool' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `/src/cmd/coderev/setup.go` | 168 | `complexity.max_return_count` | 🔵 advisory | function 'installSemgrep': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/cmd/coderev/setup.go` | 168 | `complexity.boolean_param_flag` | 🔵 advisory | function 'installSemgrep': boolean flag parameter 'hasBrew, hasPipx, hasPip3 bool' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `/src/cmd/coderev/setup.go` | 186 | `complexity.boolean_param_flag` | 🔵 advisory | function 'installMadge': boolean flag parameter 'hasNPM bool' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `/src/cmd/coderev/setup.go` | 198 | `complexity.cyclomatic` | 🔵 advisory | function 'installGitleaksFromRelease': cyclomatic complexity 8 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/cmd/coderev/setup.go` | 198 | `complexity.max_return_count` | 🔵 advisory | function 'installGitleaksFromRelease': 6 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/internal/adapters/treesitter/duplication.go` | 46 | `complexity.boolean_param_flag` | 🔵 advisory | function 'indexFile': boolean flag parameter 'hashMap map[uint64][]dupLoc' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `/src/internal/adapters/treesitter/duplication.go` | 58 | `complexity.boolean_param_flag` | 🔵 advisory | function 'emitDupFindings': boolean flag parameter 'hashMap map[uint64][]dupLoc' — flag arguments make callers hard to read |
| | | | | 💡 *Replace flag params with two separate functions or an options object.* |
| `/src/internal/analysis/runner.go` | 116 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/internal/analysis/runner.go` | 116 | `complexity.max_return_count` | 🔵 advisory | function '<anonymous>': 6 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/internal/analysis/runner.go` | 108 | `complexity.function_length` | 🔵 advisory | function 'walkFiles': 36 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `/src/internal/adapters/treesitter/walker_metrics.go` | 21 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 7 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/internal/adapters/treesitter/walker.go` | 89 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 8 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/internal/config/toolconfig.go` | 78 | `complexity.function_length` | 🔵 advisory | function 'defaultToolConfig': 34 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `/src/internal/report/generator.go` | 15 | `complexity.max_return_count` | 🔵 advisory | function 'Generate': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/internal/report/markdown_helpers.go` | 57 | `complexity.cyclomatic` | 🔵 advisory | function 'ratingBadge': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/internal/report/markdown_helpers.go` | 57 | `complexity.max_return_count` | 🔵 advisory | function 'ratingBadge': 6 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/internal/config/standards_test.go` | 65 | `complexity.cyclomatic` | 🔵 advisory | function 'TestLoadParsesComplexityThresholds': cyclomatic complexity 7 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/internal/adapters/treesitter/walker_patterns.go` | 75 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/internal/adapters/treesitter/walker_patterns.go` | 118 | `complexity.cyclomatic` | 🔵 advisory | function '<anonymous>': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/internal/report/model.go` | 118 | `complexity.max_return_count` | 🔵 advisory | function 'pillarRating': 5 return statements (max 4) — consider restructuring |
| | | | | 💡 *Use a single return with a result variable, or extract the branches to named helpers.* |
| `/src/internal/report/model.go` | 166 | `complexity.cyclomatic` | 🔵 advisory | function 'buildPillar': cyclomatic complexity 6 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |
| `/src/internal/report/markdown.go` | 20 | `complexity.function_length` | 🔵 advisory | function 'writeMarkdown': 33 lines (max 30) |
| | | | | 💡 *Each function = one verb acting on one noun.* |
| `/src/internal/report/markdown.go` | 140 | `complexity.cyclomatic` | 🔵 advisory | function 'writeExceptions': cyclomatic complexity 7 (advisory at 5) |
| | | | | 💡 *Extract branches to named helpers; prefer strategy/policy objects over switch trees.* |

</details>

<details>
<summary>🟡 <b>file_structure</b> ![A](https://img.shields.io/badge/reliability-A-brightgreen) — 8 finding(s)</summary>

| File | Line | Rule | Severity | Message |
|------|------|------|----------|---------|
| `/src/internal/architecture/detector.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 204 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `/src/cmd/coderev/main.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 240 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `/src/internal/adapters/treesitter/duplication.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 177 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `/src/internal/analysis/runner.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 226 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `/src/internal/config/standards_sections.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 232 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `/src/internal/report/sarif.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 177 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `/src/internal/report/model.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 217 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |
| `/src/internal/report/markdown.go` | 1 | `file_structure.file_length` | 🔵 advisory | file has 175 lines (advisory threshold 150) |
| | | | | 💡 *Split by concern: types.ts, constants.ts, helpers.ts, <feature>.service.ts.* |

</details>

## Hot Files

<details>
<summary>17 file(s) with findings</summary>

| File | Language | Lines | Findings | Heat |
|------|----------|-------|----------|------|
| `/src/cmd/coderev/main.go` | go | 240 | 7 | `█████` |
| `/src/cmd/coderev/setup.go` | go | 300 | 7 | `█████` |
| `/src/internal/analysis/runner.go` | go | 226 | 4 | `██░░░` |
| `/src/internal/adapters/treesitter/duplication.go` | go | 177 | 3 | `██░░░` |
| `/src/internal/report/model.go` | go | 217 | 3 | `██░░░` |
| `/src/internal/report/markdown.go` | go | 175 | 3 | `██░░░` |
| `/src/internal/adapters/treesitter/walker_patterns.go` | go | 142 | 2 | `█░░░░` |
| `/src/internal/report/markdown_helpers.go` | go | 97 | 2 | `█░░░░` |
| `/src/internal/config/toolconfig.go` | go | 112 | 1 | `░░░░░` |
| `/src/internal/config/standards_test.go` | go | 147 | 1 | `░░░░░` |
| `/src/internal/config/standards_sections.go` | go | 232 | 1 | `░░░░░` |
| `/src/internal/adapters/treesitter/walker_metrics.go` | go | 137 | 1 | `░░░░░` |
| `/src/internal/architecture/detector.go` | go | 204 | 1 | `░░░░░` |
| `/src/internal/adapters/treesitter/walker.go` | go | 118 | 1 | `░░░░░` |
| `/src/internal/report/generator.go` | go | 45 | 1 | `░░░░░` |
| `/src/cmd/coderev/adapters.go` | go | 62 | 1 | `░░░░░` |
| `/src/internal/report/sarif.go` | go | 177 | 1 | `░░░░░` |

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
| `cmd/coderev/setup.go` | `hardcoding.urls_and_paths` | installGitleaksFromRelease() must call the GitHub releases API by its canonical URL — there is no configurable alternative for a self-contained install command |
| `cmd/coderev/setup.go` | `complexity.function_length` | installGitleaksFromRelease() is bootstrap installer code: detect OS/arch, fetch release metadata, download tarball, extract, install — one atomic operation with no reusable sub-units |
| `cmd/coderev/setup.go` | `file_structure.file_length` | setup.go contains all install subcommands (setup, install-deps, install-hooks) as a single cohesive unit; splitting it would fragment one feature across multiple files |

</details>

---

*Generated by [coderev](https://github.com/srivastava-ami/coderev) · Wed, 24 Jun 2026 03:34:24 UTC*
