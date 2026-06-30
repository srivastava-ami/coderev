# coderev

[![CI](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml/badge.svg?branch=main)](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml)
[![License: BUSL-1.1](https://img.shields.io/badge/License-BUSL--1.1-blue.svg)](LICENSE)

**One command to scan your code, map it, and get an AI review.**

## Install

```bash
brew install srivastava-ami/tools/coderev
```

No Python, no Node, no config file needed.

## Run it

```bash
coderev .
```

Scans your code for bugs, secrets, and quality issues in one pass — no extra tools required. Produces a markdown report at `.coderev/report.md`.

```
✓ PASS  (or ✗ FAIL — fix the blockers shown in the report)

  adapters: treesitter, imports ✓, coverage ✓
  graph:    .coderev/graph/graph.json
  report:   .coderev/report.md
```

### What it checks

| What | How |
|---|---|
| Code quality | Tree-sitter AST analysis — cyclomatic complexity, deep nesting, long functions, magic numbers, duplication |
| Secrets & credentials | Pattern-based secret detection — API keys, passwords left in code |
| Circular dependencies | Import cycle detection for JS/TS and Go |
| Dependency CVEs | Known vulnerabilities in npm/Go/Python packages (via OSV.dev snapshot) |
| Code coverage | Reads Go/coverage/Java/JS coverage reports, warns on untested files |
| Security patterns | Semgrep rules — SQL injection, hardcoded URLs, unsafe inputs |
| Hardcoded secrets | Gitleaks — git history + files |
| JS module structure | Madge — circular deps and module graph |
| npm audit | Standard npm audit output |
| Graph analysis | Architecture-layer violation detection from the code graph |
| Custom plugins | Script-based adapters from `tool_config.toml` |

Works on **TypeScript, JavaScript, Go, Python, and Rust** out of the box.

## Output formats

```bash
coderev .                        # markdown report (default)
coderev . --format html          # self-contained HTML report
coderev . --format sarif         # SARIF (static analysis results interchange)
coderev . --json                 # findings as JSON to stdout
coderev . --output ./myreport.md # custom output path
```

## Review only what changed

```bash
coderev --diff main .
```

Only scans files changed since `main`. Useful before you push or open a PR.

## Quality gate

```bash
coderev --gate .coderev-gate.toml .
```

Evaluates findings against configurable thresholds (blockers allowed, min score). Fails the exit code if the gate isn't met — use in CI to enforce standards.

## Baseline & trend tracking

```bash
coderev --update-baseline .
```

Saves current findings to `.coderev/baseline.json`. Future runs show added/removed findings as a delta.

## AI-powered code review

```bash
# 1. Configure your LLM provider
coderev config llm --enable --provider cli --command "claude -p {prompt}"
coderev config llm --enable --provider anthropic  # reads ANTHROPIC_API_KEY
coderev config llm --enable --provider ollama

# 2. Run coderev with review (uses findings + graph context)
coderev --review .

# Or review a specific diff
coderev review --diff main .

# Post the AI review back to a PR comment
export GITHUB_TOKEN=ghp_...
coderev review --diff main --post-pr .
```

The review prompt includes:
- git diff hunks
- code graph neighbors of changed files
- static analysis findings
- `.coderevignore` filtering

For large diffs, the review is automatically chunked per-file and sent in parallel.

### Full graph review (no diff required)

```bash
coderev --full-review .
```

Reviews every file in the code graph — useful for onboarding or deep architecture audits.

## Code graph

```bash
coderev graph .                            # build graph
coderev graph . --output ./custom-dir      # custom output dir
```

Builds a native graph (`.coderev/graph/graph.json` + `graph.html`) with file, function, and type nodes connected by imports, calls, and containment edges.

## Ask the LLM anything

```bash
coderev ask "What does the graph adapter do?"
```

Sends a raw prompt to the configured LLM provider using the tool config.

## GitHub PR annotations

```bash
coderev --diff main --annotate-pr --repo owner/repo --pr 42 .
```

Posts findings as inline PR comments (requires `gh` CLI and `GITHUB_TOKEN`). Repo and PR number auto-detect from git remote if omitted.

## Plugins

```bash
coderev plugin install manifest.toml   # install a custom plugin
coderev plugin list                    # list installed plugins
```

Extend analysis with script-based adapters defined in manifest files.

## Setup

```bash
coderev setup   # install external deps (gitleaks, semgrep, madge) + git hooks
coderev install-deps   # external deps only
coderev install-hooks  # pre-commit/pre-push git hooks only
```

## Add it to GitHub Actions

```yaml
name: coderev
on: [pull_request]
jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run coderev
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          docker run --rm -v "$GITHUB_WORKSPACE:/src" \
            -e GH_TOKEN \
            ghcr.io/srivastava-ami/coderev:latest \
            --diff "origin/${{ github.base_ref }}" \
            --annotate-pr \
            --gate /src/.coderev-gate.toml \
            --repo "${{ github.repository }}" \
            --pr "${{ github.event.pull_request.number }}" \
            /src
```

## Using with AI agents (Claude Code, Copilot, Cursor)

```markdown
## Quality gate
Before every commit: `coderev .`
Fix all blockers listed in `.coderev/report.md` before pushing.
For AI review: read `.coderev/review.md` after running.
```

## License

Business Source License 1.1 — free for non-commercial use. Converts to Apache 2.0 on 2030-06-23.

Built by [Amit Srivastava](https://github.com/srivastava-ami). ⭐ [Star the repo](https://github.com/srivastava-ami/coderev) if it's useful.
