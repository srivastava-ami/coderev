# Distribution & full CLI reference

This page holds the full distribution matrix, the complete CLI flag list, the full adapter matrix, and plugin/custom-adapter setup. The [README](../README.md) covers the common path.

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
```

`graph` also accepts `--output <dir>` to set the output directory (default: `<target>/.coderev/graph`).

### Quality gate TOML (`--gate`)

```toml
# .coderev-gate.toml  (defaults: 0 blockers, 5 majors, 10 advisories, 20 total)
max_blockers  = 0
max_majors    = 5
max_advisories = 10
max_total     = 20
```

With `--json`, the gate result is embedded in the JSON output. Without `--json`, pass/fail is printed after the summary. Exit code `1` on failure.

## Full adapter matrix

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

The optional external scanners auto-install only when you enable them in `tool_config.toml`. To pre-install them explicitly:

```bash
coderev install-deps   # downloads gitleaks + semgrep + madge to ~/.coderev/tools/
```

### Custom adapter — no Go required

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
