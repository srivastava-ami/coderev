# coderev

[![CI](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml/badge.svg?branch=main)](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/srivastava-ami/coderev)](https://goreportcard.com/report/github.com/srivastava-ami/coderev)
[![License: BUSL-1.1](https://img.shields.io/badge/License-BUSL--1.1-blue.svg)](LICENSE)

**Deterministic, polyglot code-standards enforcement. No server. No LLM. No per-seat licence.**

```bash
coderev .
```
```
  files scanned : 48
  findings      : 2 blockers · 4 major · 27 advisory
  status: ✗ FAIL (blockers must be resolved)
  report: coderev-report.md
```

---

## Install

```bash
# Homebrew (macOS / Linux) — recommended
brew tap srivastava-ami/coderev
brew install coderev

# curl installer — also installs gitleaks, semgrep, madge
curl -fsSL https://raw.githubusercontent.com/srivastava-ami/coderev/main/scripts/install.sh | bash

# From source (Go 1.22+)
make install

# Docker (zero host dependencies)
docker pull ghcr.io/srivastava-ami/coderev:latest
```

---

## Review a PR

```bash
# 1. Check out the branch
gh pr checkout 42

# 2. Scan only changed files
coderev --diff main .

# 3. Post inline comments on the PR
coderev --annotate-pr --diff main .
```

Auto-detects repo slug and PR number from git context. Override if needed:
```bash
coderev --annotate-pr --repo owner/repo --pr 42 --diff main .
```

---

## All flags

```
coderev [directory] [flags]

  --diff <ref>         scan only files changed since <ref> (e.g. main, HEAD~1)
  --annotate-pr        post findings as inline GitHub PR comments
  --repo owner/repo    override repo slug (auto-detected from git remote)
  --pr <number>        override PR number (auto-detected from gh pr view)
  --format <fmt>       markdown (default) | html | sarif
  --output <path>      custom output path
  --standards <path>   path to code_review_standards.toml (auto-discovered if omitted)
  --config <path>      path to tool_config.toml (auto-discovered if omitted)
  --update-baseline    save current findings as baseline; future runs show delta (▲/▼)
```

---

## CI — GitHub Actions

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

Exit code `1` when blockers are found — blocks the merge.

Full workflow: `.github/workflows/code-quality.yml`

---

## Standards file

Rules live in one TOML file committed to the repo root:

```toml
# code_review_standards.toml
[complexity.cyclomatic]
max_value     = 10
hard_block_at = 15

[complexity.function_length]
max_lines = 40

[[exceptions]]
rule           = "complexity.cyclomatic"
file_or_module = "src/legacy/parser.ts"
justification  = "Third-party parser — tracked in JIRA-4421"
expires        = "2026-12-31"
```

No standards file? Built-in defaults apply automatically — scan any repo with zero setup.

---

## Adapters

| Adapter | Checks | Install |
|---------|--------|---------|
| `treesitter` | complexity, type safety, patterns, security | built-in |
| `gitleaks` | secrets & credentials | `brew install gitleaks` |
| `semgrep` | OWASP / injection / crypto | `brew install semgrep` |
| `madge` | circular deps, NX boundaries | `npm i -g madge` |
| `npmaudit` | vulnerable npm packages | ships with Node |
| `coverage` | line coverage threshold | reads existing lcov / go cover |
| `custom` | any tool via NDJSON | your binary |

```bash
make install-deps   # installs gitleaks + semgrep + madge
```

Custom adapter — no Go required:
```toml
# tool_config.toml
[[adapters.custom]]
name     = "my-checker"
binary   = "/usr/local/bin/my-checker"
enabled  = true
protocol = "ndjson"
rules    = ["security.custom.*"]
args     = ["--format=coderev-json", "{{target}}"]
```

---

## Using with AI agents

coderev runs as an independent shell process — the analysis consumes **zero agent tokens** and makes no LLM calls. The agent triggers it, reads the output file, and acts on structured findings.

**Wire it into your CLAUDE.md / AGENTS.md:**

```markdown
## Quality gate
Before every commit: `coderev .`
Report is at `coderev-report.md` — fix all blockers before pushing.
```

**Or run on demand from the agent:**

```bash
coderev --diff main . --output /tmp/findings.md
# agent reads /tmp/findings.md — one file read, structured output
```

The Markdown report is machine-parseable: every finding has a rule ID, file path, line number, and remediation text. The agent reads the report in one shot and acts on it — no token streaming from the analysis engine.

---

## Why this exists

AI coding agents (Claude Code, Copilot, Cursor) now write most of the code in fast-moving teams. Standards in CLAUDE.md / AGENTS.md are advisory — no violation list, no severity, no gate.

`coderev` is the missing piece: a single binary, a single TOML file, a single exit code.

| | coderev | SonarQube | ESLint/Biome | CodeRabbit |
|---|---|---|---|---|
| Polyglot (one tool) | ✅ | ✅ | ❌ per language | ✅ |
| Local / offline | ✅ | ❌ server | ✅ | ❌ cloud |
| Standards in git | ✅ TOML | ❌ UI | partial | ❌ |
| Inline PR comments | ✅ | ✅ | ❌ | ✅ |
| Zero infrastructure | ✅ | ❌ | ✅ | ❌ |
| Per-seat cost | free | $$ | free | $24–40/mo |

---

## License

Business Source License 1.1 — free for non-commercial use. Converts to Apache 2.0 on 2030-06-23. See [LICENSE](LICENSE).
