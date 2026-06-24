# Contributing

## Principles

1. **No network at runtime.** Analysis is offline. External tools run as local subprocesses.
2. **Graceful degradation.** A missing optional tool produces a warning, never a hard failure.
3. **Standards file is the contract.** Every enforced rule has an entry in `code_review_standards.toml`. No thresholds in Go source.
4. **The swap boundary is sacred.** `internal/analysis/adapter.go:ToolAdapter` is the only interface new tools implement.
5. **Dogfood.** `coderev .` must show no new blockers before every PR.

---

## Development setup

```bash
git clone <repo>
cd nx-code-review-standard

make build              # → ./bin/coderev
coderev install-deps    # installs gitleaks, semgrep, madge
coderev install-hooks   # installs pre-commit + pre-push hooks

go test ./...           # run tests
coderev .               # dogfood scan
```

---

## Before submitting a PR

```bash
go test ./...     # tests pass
coderev .         # no new blockers
go build ./...    # clean build
go vet ./...      # vet passes
```

PR description must include: `what`, `why`, `risk`, `how_to_test`, `rollback`.
