# coderev

[![CI](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml/badge.svg?branch=main)](https://github.com/srivastava-ami/coderev/actions/workflows/code-quality.yml)
[![License: BUSL-1.1](https://img.shields.io/badge/License-BUSL--1.1-blue.svg)](LICENSE)

**One command to check your code, map it, and get an AI review.**

## Install

```bash
brew install srivastava-ami/tools/coderev
```

That's it. No Python, no Node, no config file needed.

## Run it

```bash
coderev .
```

That's the whole command. Run it in any project folder. It:

1. **Scans your code** for bugs, secrets, and quality issues
2. **Builds a code map** (`graph.html`) you can open in your browser
3. **Writes a prompt file** ready for any AI to review

```
✓ PASS  (or ✗ FAIL — fix the blockers shown in coderev-report.md)

  graph:   .coderev/graph/graph.html
  prompt:  .coderev/prompt.md
```

Open `coderev-report.md` to see every finding with file name, line number, and what to fix.

## Get an AI review

Run this once to connect your Claude subscription:

```bash
coderev config llm --enable --provider cli --command "claude -p {prompt}"
```

Now every `coderev .` also writes `.coderev/review.md` — an AI review that looks at
your code map, your findings, and what actually changed, then tells you about logic
bugs and edge cases the scanner can't catch.

## Review only what changed (PR mode)

```bash
coderev --diff main .
```

Same as above but only looks at files you changed compared to `main`.
Useful before you push or open a pull request.

## Add it to GitHub Actions

Paste this into `.github/workflows/coderev.yml`:

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
            --repo "${{ github.repository }}" \
            --pr "${{ github.event.pull_request.number }}" \
            /src
```

This posts findings as inline comments on the PR and fails the check if there are blockers.

## What does it check?

Everything in one pass, no extra tools needed:

| What | Examples |
|---|---|
| Secrets & credentials | API keys, passwords left in code |
| Circular dependencies | Module A imports B imports A |
| Complexity | Functions that are too long or deeply nested |
| Security patterns | SQL injection, hardcoded URLs, unsafe inputs |
| Dependency CVEs | Known vulnerabilities in your npm/Go/Python packages |

Works on **TypeScript, JavaScript, Go, Python, and Rust** out of the box.

## Using with AI agents (Claude Code, Copilot, Cursor)

Add this to your `CLAUDE.md` or `AGENTS.md`:

```markdown
## Quality gate
Before every commit: `coderev .`
Fix all blockers listed in `coderev-report.md` before pushing.
For AI review: read `.coderev/review.md` after running.
```

The agent runs `coderev .`, reads the report and review files, and fixes issues —
without sending your whole codebase to an LLM.

## License

Business Source License 1.1 — free for non-commercial use. Converts to Apache 2.0 on 2030-06-23.

Built by [Amit Srivastava](https://github.com/srivastava-ami). ⭐ [Star the repo](https://github.com/srivastava-ami/coderev) if it's useful.
