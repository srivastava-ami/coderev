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
