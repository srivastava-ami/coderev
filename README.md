# coderev

[![CI](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml/badge.svg?branch=main)](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml)
[![License: BUSL-1.1](https://img.shields.io/badge/License-BUSL--1.1-blue.svg)](LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/srivastava-ami/coderev?style=social)](https://github.com/srivastava-ami/coderev)

> ⭐ **If coderev is useful to you, please [star the repo](https://github.com/srivastava-ami/coderev)** — it's the easiest way to support the project and help others find it. Built by [Amit Srivastava](https://github.com/srivastava-ami).

**Deterministic, polyglot code-standards enforcement. No server. No LLM. No per-seat licence.**

TypeScript · JavaScript · Go · **Python** · **Rust**

```bash
coderev .
```
```
  files scanned : 48
  findings      : 2 blockers · 4 major · 27 advisory
  status: ✗ FAIL (blockers must be resolved)
  report: coderev-report.md
```

All findings are produced by deterministic static analysis — zero LLM calls, zero network, zero cost.

---

## Install

```bash
# Homebrew (macOS / Linux) — recommended
brew install srivastava-ami/tools/coderev

# curl installer
curl -fsSL https://raw.githubusercontent.com/srivastava-ami/coderev/main/scripts/install.sh | bash

# From source (Go 1.22+)
make install

# Docker (zero host dependencies)
docker pull ghcr.io/srivastava-ami/coderev:latest
```

**Nothing else to install.** `coderev .` is self-contained: secret scanning, circular-dependency detection, and the injection rules are all native Go. No `brew install`, `npm install`, or `pip install`, and no external scanners required. (Optional: gitleaks/semgrep/madge add extra depth — enable them in `tool_config.toml` and they auto-install to `~/.coderev/tools/` only if turned on.)

## Distribution

| Method | What happens | Who runs it |
|---|---|---|
| **Homebrew tap** | Casks in `Formula/coderev.rb` — points to GitHub release assets | Automated on tag push (`release.yml`) |
| **curl installer** | `scripts/install.sh` — fetches latest GitHub release, copies to `~/.local/bin/` | User-initiated |
| **`make dist`** | Cross-compiles for darwin/linux × amd64/arm64 → `bin/dist/` | Developer or CI |
| **GitHub release** | `release.yml` — tags `v*` trigger `make dist`, create release, upload binaries, update Homebrew formula | Tag push |
| **Docker image** | `docker-publish.yml` — multi-arch (`linux/amd64`, `linux/arm64`) published to `ghcr.io` | Tag push or manual |
| **`make install`** | `go build` + copy to `/usr/local/bin/` | Developer |

All release binaries are built by `release.yml` with `-ldflags="-s -w"` (stripped, DWARF removed), versioned via `git describe`.

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
  --standards <path>   path to custom standards TOML (built-in defaults apply if omitted)
  --config <path>      path to tool_config.toml (auto-discovered if omitted)
  --update-baseline    save current findings as baseline; future runs show delta (▲/▼)
  --json               output findings as structured JSON (machine-readable)
  --gate <path>        evaluate against quality gate thresholds (.coderev-gate.toml)
  --plugin-dir <path>  custom plugin directory (default: ~/.config/coderev/plugins)

Subcommands:
  graph [directory]            build and export native code graph (graph.json + self-contained graph.html)
  setup                        install scanner dependencies + git hooks (full onboarding)
  install-hooks                install pre-commit, pre-push, post-commit git hooks
  install-deps                 download optional external tools (gitleaks, semgrep, madge) to ~/.coderev/tools/
  plugin install <manifest>    install a plugin from its coderev-plugin.toml manifest
  plugin list                  list installed plugins

`graph` also accepts `--output <dir>` to set the output directory (default: `<target>/.coderev/graph`).
```

Quality gate TOML (`--gate`):
```toml
# .coderev-gate.toml  (defaults: 0 blockers, 5 majors, 10 advisories, 20 total)
max_blockers  = 0
max_majors    = 5
max_advisories = 10
max_total     = 20
```

With `--json`, gate result is embedded in the JSON output. Without `--json`, pass/fail is printed after the summary. Exit code `1` on failure.

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

## Rules catalog

All 56 built-in rules, grouped by pillar, with full TOML configuration and severity defaults:
→ **[docs/rules-reference.md](docs/rules-reference.md)**

## Standards

Standards are built in — no config file needed.

Built-in defaults apply automatically — scan any repo with zero setup. To override thresholds for a specific repo, pass a custom TOML file:

```bash
coderev --standards /path/to/custom.toml .
```

---

## Adapters

| Adapter | Checks | Type | Default |
|---|---|---|---|
| `treesitter` | complexity, type safety, hardcoding, security patterns, documentation, structure, duplication — **all 5 languages** | native (pure Go) | ✅ on |
| `depcve` | dependency CVE via offline OSV snapshot (npm, Go, PyPI) — shipped snapshot in repo, cached from remote | native (pure Go) | ✅ on |
| `secrets` | secrets & credentials (regex + Shannon entropy) | native (pure Go) | ✅ on |
| `imports` | circular deps, NX boundaries (Tarjan SCC) | native (pure Go) | ✅ on |
| `npmaudit` | vulnerable npm packages (legacy fallback — depcve replaces for non-npm ecosystems) | external (npm) | ✅ on |
| `coverage` | line coverage threshold (lcov, cobertura) | reads existing report | ✅ on |
| `gitleaks` | extra secret rules | external | ⚪ optional |
| `semgrep` | wider OWASP / injection / crypto | external | ⚪ optional |
| `madge` | circular-deps cross-check | external | ⚪ optional |
| `custom` | any tool via NDJSON | external | ⚪ optional |

Native Go adapters cover **TypeScript, JavaScript, Go, Python, and Rust** with **no external tools**: tree-sitter walkers for structure/complexity/security patterns, plus native secret scanning, native circular-dependency detection, and native offline OSV dependency-CVE scanning. gitleaks/semgrep/madge are optional enrichment — the native adapters already cover their rules.

The optional external scanners auto-install only when you enable them in `tool_config.toml`. To pre-install them explicitly:

```bash
coderev install-deps   # downloads gitleaks + semgrep + madge to ~/.coderev/tools/
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

## Plugins

Plugins are external binaries discovered automatically from `~/.config/coderev/plugins/`. Each plugin ships with a `coderev-plugin.toml` manifest:

```toml
# my-linter-plugin.toml
name         = "my-linter"
version      = "1.0.0"
description  = "Custom linter for internal conventions"
binary       = "my-linter"
capabilities = ["conventions.custom.*"]
languages    = ["go", "python"]
```

```bash
coderev plugin install my-linter-plugin.toml   # copy to ~/.config/coderev/plugins/
coderev plugin list                             # list installed
```

On every scan, coderev discovers and loads all plugins from the plugin directory. Plugin binaries must be on `$PATH`.

---

## Code graph

```bash
coderev graph .                # writes .coderev/graph/graph.json + graph.html
coderev graph --output ./out   # custom output directory
```

Produces a fully **offline, deterministic** code graph from source files — nodes are files, functions, and types; edges are imports, calls, and containment. The output is a reusable `graph.json` (deterministic byte-for-byte) and a self-contained `graph.html` (interactive SVG, zero CDN dependencies) — view it in any browser with no server.

Uses the same file discovery as the scanner (honours `.gitignore`). The output directory is configurable via `--output`, `[graph] output_dir` in `tool_config.toml`, or the default `.coderev/graph/`.

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
| Polyglot (one tool) | ✅ TS/JS/Go/Python/Rust | ✅ | ❌ per language | ✅ |
| Local / offline | ✅ | ❌ server | ✅ | ❌ cloud |
| Standards in git | ✅ TOML | ❌ UI | partial | ❌ |
| Inline PR comments | ✅ | ✅ | ❌ | ✅ |
| Zero infrastructure | ✅ | ❌ | ✅ | ❌ |
| Machine-readable output | ✅ JSON | ✅ SARIF | ❌ | ❌ |
| Quality gate (pass/fail) | ✅ | ✅ | ❌ | ❌ |
| Plugin ecosystem | ✅ | ✅ | ✅ | ❌ |
| Per-seat cost | free | $$ | free | $24–40/mo |

---

## License

Business Source License 1.1 — free for non-commercial use. Converts to Apache 2.0 on 2030-06-23. See [LICENSE](LICENSE).
