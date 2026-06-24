# Contributing

## Principles

1. **No network at runtime.** Analysis is offline. External tools run as local subprocesses.
2. **Graceful degradation.** A missing optional tool produces a warning, never a hard failure.
3. **Standards file is the contract.** Every enforced rule has an entry in `code_review_standards.toml`. No thresholds in Go source.
4. **Domain never imports infrastructure.** `internal/analysis/` must not import `internal/config/` or any adapter package. All type definitions live in the domain — `config/` imports `analysis/`, never the reverse.
5. **The swap boundaries are sacred.** Two ports are defined in `internal/analysis/`: `ToolAdapter` (analysis) and `DiffService` (SCM diff). New tools implement these interfaces — no other code changes needed.
6. **Toolmgr for external deps.** External scanners are downloaded by `internal/toolmgr/` on first run. Never assume `brew`, `pip`, or `npm` availability — use `toolmgr` to resolve binaries.
7. **Dogfood.** `coderev .` must show **0 blockers** before every PR.

---

## Development setup

```bash
git clone <repo>
cd nx-code-review-standard

make build              # → ./bin/coderev
coderev install-hooks   # installs pre-commit + pre-push hooks

go test ./...           # run tests
coderev .               # dogfood scan (must show 0 blockers)
                        # gitleaks / semgrep / madge auto-install on first run
```

---

## Worktree-based development

For parallel feature work (e.g. adding a new language walker), use git worktrees:

```bash
# Create a worktree for a new feature
git worktree add ../coderev-feature-branch feature-branch

# Work in both branches simultaneously
# Each has its own go.mod, build cache, bin/ — no cross-contamination

# When done
git worktree remove ../coderev-feature-branch
```

This allows you to build and test two (or more) branches at the same time without stashing or re-building.

---

## Adding a new language walker

1. Create a new file `internal/analysis/walkers/<language>_patterns.go`
2. Implement language-specific AST checks using tree-sitter grammar
3. Register rules in `internal/analysis/rule_registry.go` with the appropriate `applies_to` language list
4. Add test fixtures in `testdata/<language>/`
5. Run `coderev .` on a repo that uses that language — verify findings are accurate
6. Rebuild the graphif: `go run ~/.claude/skills/graphify/main.go rebuild`

---

## Tool Manager internals

External scanners are auto-installed by `internal/toolmgr/toolmgr.go`. Key facts for contributors:

- **`toolmgr.EnsureAll()`** is called by `cmd/coderev/main.go` at the start of every scan.
- Tools are stored in `~/.coderev/tools/` — resolved via `toolmgr.ToolPath(name)`.
- `cmd/coderev/adapters.go` uses `resolveTool()` which checks toolmgr path first, `$PATH` second, config value third.
- To add a new auto-installed tool: add a `tool` entry in `toolmgr.go` with an `Install` function and a corresponding `ensure*()` function.
- **Never download to `/tmp/` or CWD** — always use `~/.coderev/tools/` for cacheable, user-scoped storage.
- Toolmgr never blocks the scan — if installation fails, the adapter reports `IsAvailable() == false` and scan continues with degraded coverage.

## Rebuilding the knowledge graph

After significant structural changes, rebuild the graphify index:

```bash
go run ~/.claude/skills/graphify/main.go rebuild
```

This scans the graphify-out/ directory and regenerates the index for semantic search.

---

## Cross-compiling (release builds)

```bash
make dist   # builds for darwin/linux × amd64/arm64 + windows/amd64
            # output → bin/dist/
```

Binaries are stripped (`-ldflags="-s -w"`) and carry the version tag from `git describe`.

---

## Release process

See [docs/releasing.md](releasing.md).

---

## Before submitting a PR

```bash
go test ./...     # tests pass
coderev .         # 0 blockers — dogfood pass required
go build ./...    # clean build
go vet ./...      # vet passes
```

PR description must include: `what`, `why`, `risk`, `how_to_test`, `rollback`.
