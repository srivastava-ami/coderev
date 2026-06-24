# coderev v3 — Implementation Plan

## Goal
Transform coderev into a genuine polyglot SonarQube replacement by adding full
Python and Rust coverage, fixing foundational bugs, shipping a quality-gate API,
and building a plugin ecosystem — all without a UI or server.

## Worktree strategy

```
main ───────────────────────────────────── (stable, frozen)
                          
dev (integration branch) ←── merge all worktrees sequentially

Merge order into dev:
  1. wt/bugfixes          (P0)
  2. wt/magic-numbers     (P1) + wt/config-updates     (P2) [parallel]
  3. wt/python-core       (P1) + wt/rust-core           (P1) [parallel]
  4. wt/python-extended   (P2) + wt/rust-extended       (P2) [parallel]
  5. wt/quality-gate      (P3) + wt/plugin-ecosystem    (P3) [parallel]
  6. Dogfood: coderev --standards code_review_standards.toml .
```

## Phase order

| Phase | Worktree | Description | Effort |
|-------|----------|-------------|--------|
| P0    | bugfixes | 4 critical bugs | ~10 lines |
| P1    | python-core | print, except:eval guards + 3 walkers | 3 new functions |
| P1    | rust-core | unwrap, panic guards + 2 walkers | 2 new functions |
| P1    | magic-numbers | generic literal scanner for all 5 languages | 1 function + LangDef |
| P2    | config-updates | default_standards, cobertura parser, registry | ~5 files |
| P2    | python-extended | SQL, subprocess, mutable defaults, wildcard | 4 new functions |
| P2    | rust-extended | unsafe, transmute, await-in-loop, clone | 4 new functions |
| P3    | quality-gate | JSON output, gate TOML, evaluator | new `internal/gate/` |
| P3    | plugin-ecosystem | manifest, install, registry subcommands | new `internal/plugin/` |

## Dependency constraints

```
bugfixes ──→ python-core ──→ python-extended
bugfixes ──→ rust-core   ──→ rust-extended
bugfixes ──→ magic-numbers, config-updates  (both depend on fixed runner)

quality-gate     ← independent (no lang dep)
plugin-ecosystem ← independent (no lang dep)
```

Each worktree is `git worktree add ../nx-code-review-<name> <dependency-branch>`.

## Detailed worktree breakdown

### wt/bugfixes — Phase 0

| File | Change |
|---|---|
| `walker_patterns.go:34` | Add `w.checkFloatingPromise(line, lineNum)` |
| `duplication.go:118-119` | Rule: `"file_structure.duplication"`, Pillar: `"file_structure"` |
| `rule_registry.go` | Add `"file_structure.duplication"` entry with CWE-1041 |
| `npmaudit/adapter.go:109` | `exec.LookPath` → `os.Stat` |
| `runner.go:191` | Path-boundary check in `matchesException` |

### wt/python-core — Phase 1

New file: `internal/adapters/treesitter/walker_python.py`
- `checkPythonPrint` — detect `print(` in non-test .py files (observability.blocker)
- `checkPythonEmptyExcept` — stateful bare except + pass detection (stability.blocker)
- `checkPythonEval` — `eval(`, `exec(`, `compile(` (security.blocker)

Modify:
- `walker_patterns.go` — import + wire 3 new checks in `checkPatterns()`
- `walker_helpers.go` — add `pythonGuard()` (returns skip=true if not Python)
- `rule_registry.go` — add 3 rule IDs with CWE tags

### wt/rust-core — Phase 1

New file: `internal/adapters/treesitter/walker_rust.rs`
- `checkRustUnwrap` — `.unwrap()`, `.expect("...")` (stability.major)
- `checkRustPanic` — `panic!(`, `unreachable!(`, `unimplemented!(` (stability.major)

Modify:
- `walker_patterns.go` — import + wire 2 new checks
- `walker_helpers.go` — add `rustGuard()`
- `rule_registry.go` — add 2 rule IDs with CWE tags

### wt/magic-numbers — Phase 1

- Add `LiteralTypes []string` to `LangDef` in `languages.go`
- Add `checkMagicNumber()` in `walker_metrics.go`
- Wire into `checkPatterns()` loop
- Ensure `"hardcoding.magic_numbers"` in `rule_registry.go`

### wt/config-updates — Phase 2

- `default_standards.toml`: add python, rust to `applies_to`
- `coverage/adapter.go`: add `parseCobertura()` for `coverage.xml`
- `rule_registry.go`: final pass — ensure all new rule IDs have tags

### wt/python-extended — Phase 2

Add to `walker_python.go`:
- `checkPythonSQLConcat` — f-strings with SQL keywords (security.blocker)
- `checkPythonSubprocessInjection` — os.system/subprocess with concat (security.blocker)
- `checkPythonMutableDefaultArg` — `def foo(x=[])` (stability.major)
- `checkPythonWildcardImport` — `from X import *` (stability.major)

Add `LangPython` to `dupEligible()` in `duplication.go`.

### wt/rust-extended — Phase 2

Add to `walker_rust.go`:
- `checkRustUnsafe` — `unsafe {` blocks (security.blocker)
- `checkRustTransmute` — `std::mem::transmute(` (type_safety.blocker)
- `checkRustAwaitInLoop` — `.await` inside for/while (performance.major)
- `checkRustNeedlessClone` — `.clone()` signals (performance.advisory)

Add `LangRust` to `dupEligible()` in `duplication.go`.

### wt/quality-gate — Phase 3

New package: `internal/gate/`
- `config.go` — `.coderev-gate.toml` parser
- `evaluator.go` — gate evaluation against findings
- `json.go` — `--format json` structured stdout output

Modify:
- `cmd/coderev/main.go` — add `--gate`, `--format json` flags
- `cmd/coderev/output.go` — wire JSON output

### wt/plugin-ecosystem — Phase 3

New package: `internal/plugin/`
- `manifest.go` — plugin manifest TOML parser
- `installer.go` — download, checksum verify, extract
- `registry.go` — list/search installed plugins

Modify:
- `cmd/coderev/setup.go` — add `plugin` subcommand tree

## Dogfooding

After final merge into dev:

```bash
make build
./bin/coderev --standards code_review_standards.toml --config tool_config.toml .
```

Expected: coderev catches its own violations. New Python/Rust walkers
produce zero findings on this Go/TS codebase (correctly skipped by language
guards). The `checkFloatingPromise` fix now fires on any JS/TS test files
with unhandled promises.
