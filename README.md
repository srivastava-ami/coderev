# coderev

[![CI](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml/badge.svg?branch=main)](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml)
[![License: BUSL-1.1](https://img.shields.io/badge/License-BUSL--1.1-blue.svg)](LICENSE)

**A single binary that gates code quality across TS · JS · Go · Python · Rust — deterministic, offline, no LLM, no server, no per-seat cost.**

## What you get

```bash
coderev .
```
```
  files scanned : 48
  findings      : 2 blockers · 4 major · 27 advisory
  status: ✗ FAIL (blockers must be resolved)
  report: coderev-report.md
```

Every finding comes from native static analysis — **zero LLM calls, zero network, zero cost.** One binary covers secret scanning, circular-dependency detection, and the injection rules: no `npm install`, no `pip install`, no external scanners required.

## Code map — one binary, zero tokens

```bash
coderev graph .          # → .coderev/graph/graph.json + self-contained graph.html
```

Builds a code knowledge graph natively in Go — nodes are files, functions, and types; edges are imports, calls, and containment. It's the **single-binary, zero-token replacement for Python graphify**: no Python, no venv, runs as a plain shell process and consumes **zero agent tokens**. Output is byte-for-byte deterministic and the `graph.html` is fully self-contained (interactive SVG, no CDN) — open it in any browser, no server. Honours `.gitignore`.

```bash
coderev graph --output ./out   # custom output directory
```

The output directory is configurable via `--output`, `[graph] output_dir` in `tool_config.toml`, or the default `.coderev/graph/`.

## Install

```bash
brew install srivastava-ami/tools/coderev                                              # macOS / Linux
curl -fsSL https://raw.githubusercontent.com/srivastava-ami/coderev/main/scripts/install.sh | bash
make install                                                                            # from source (Go 1.22+)
docker pull ghcr.io/srivastava-ami/coderev:latest                                       # zero host deps
```

**Nothing else to install.** Optional: gitleaks / semgrep / madge add extra depth — enable them in `tool_config.toml` and they auto-install to `~/.coderev/tools/` only when turned on.

## Review a PR

```bash
gh pr checkout 42
coderev --diff main .                # scan only changed files
coderev --annotate-pr --diff main .  # post inline comments (repo & PR auto-detected)
```

Override the auto-detected repo/PR if needed:
```bash
coderev --annotate-pr --repo owner/repo --pr 42 --diff main .
```

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

Exit code `1` when blockers are found — blocks the merge. Full workflow: `.github/workflows/code-quality.yml`.

## LLM review (context-aware, advisory)

```bash
# Enable once — uses your existing Claude subscription via the CLI
coderev config llm --enable --provider cli

# Review only changed files, with full code-graph context
coderev review --diff main .

# Post as an inline PR comment (no duplicates — upserts in place)
coderev review --diff main --post-pr .
```

`coderev review` assembles a compact, context-aware prompt — git diff hunks, the graph neighborhood of changed functions (callers + callees, 2 hops), and static findings — and sends it to the configured LLM. The LLM identifies logical issues, edge cases, and correctness concerns that static analysis can't catch.

**Advisory only:** the review never affects the scan gate (`coderev .` owns pass/fail). Works with any provider: `cli` (Claude/Ollama via the CLI), `api` (Anthropic API key), or any OpenAI-compatible endpoint.

## Adapters

coderev's native Go adapters cover **TypeScript, JavaScript, Go, Python, and Rust with no external tools** — on by default:

| Adapter | Checks | Default |
|---|---|---|
| `treesitter` | complexity, type safety, hardcoding, security patterns, documentation, structure, duplication — all 5 languages | ✅ on |
| `secrets` | secrets & credentials (regex + Shannon entropy) | ✅ on |
| `imports` | circular deps, NX boundaries (Tarjan SCC) | ✅ on |
| `depcve` | dependency CVE via offline OSV snapshot (npm, Go, PyPI) | ✅ on |

gitleaks / semgrep / madge are **optional enrichment** — the native adapters already cover their rules. Custom adapters and plugins (any external binary, no Go required) are supported too.

→ **Full adapter matrix, custom-adapter & plugin setup, and all CLI flags: [docs/distribution.md](docs/distribution.md)**

## Standards & rules

Standards are **built into the binary — no config file needed.** Scan any repo with zero setup. To override thresholds for a specific repo, pass a custom TOML file:

```bash
coderev --standards /path/to/custom.toml .
```

All **55 built-in rules**, grouped by pillar with full TOML configuration and severity defaults: → **[docs/rules-reference.md](docs/rules-reference.md)**

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

## Why this exists

AI coding agents (Claude Code, Copilot, Cursor) now write most of the code in fast-moving teams. Standards in CLAUDE.md / AGENTS.md are advisory — no violation list, no severity, no gate.

`coderev` is the missing piece: a single binary, built-in standards, a single exit code.

| | coderev | SonarQube | ESLint/Biome | CodeRabbit |
|---|---|---|---|---|
| Polyglot (one tool) | ✅ TS/JS/Go/Python/Rust | ✅ | ❌ per language | ✅ |
| Local / offline | ✅ | ❌ server | ✅ | ❌ cloud |
| Standards in git | ✅ TOML | ❌ UI | partial | ❌ |
| Inline PR comments | ✅ | ✅ | ❌ | ✅ |
| Zero infrastructure | ✅ | ❌ | ✅ | ❌ |
| Machine-readable output | ✅ JSON | ✅ SARIF | ❌ | ❌ |
| Quality gate (pass/fail) | ✅ | ✅ | ❌ | ❌ |
| Plugin ecosystem | ✅ | ✅ | ✅ | ❌ |
| Per-seat cost | free | $$ | free | $24–40/mo |

## License

Business Source License 1.1 — free for non-commercial use. Converts to Apache 2.0 on 2030-06-23. See [LICENSE](LICENSE).

Built by [Amit Srivastava](https://github.com/srivastava-ami). ⭐ If coderev is useful, [star the repo](https://github.com/srivastava-ami/coderev).
