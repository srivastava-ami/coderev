# Adding coderev to Claude Code

`coderev` runs as a standalone shell command — it is **not** an MCP server or Claude tool. It consumes zero agent tokens. The analysis runs entirely in the binary; Claude reads the output file when needed.

---

## Quickest integration: add to your project's CLAUDE.md

Drop this into the `CLAUDE.md` at the root of any repo where you use Claude Code:

```markdown
## Quality gate

Before every commit, run:
```
coderev .
```
Report: `coderev-report.md`

Fix all **blockers** before pushing. Advisory findings must be addressed or have a justification in the PR description. The CI gate runs the same check — if it passes locally it passes in CI.
```

Claude will run `coderev .` as a shell command before committing. The analysis happens outside Claude — no tokens consumed for the scan itself.

---

## Reading the report from Claude

When Claude needs to act on findings, it reads the report file directly:

```bash
coderev --diff main . --output coderev-report.md
```

The Markdown report is structured — rule ID, file, line, severity, remediation — so Claude can parse and act on it in one read without streaming analysis output.

For a focused scan before a specific commit:

```bash
coderev --diff HEAD~1 .
```

---

## Wiring as a pre-commit hook via Claude Code

Claude Code can install the hook into `.git/hooks/`:

```bash
coderev install-hooks
```

After this, `coderev --diff HEAD .` runs automatically on every `git commit`. Claude never needs to invoke coderev manually — the hook handles it. If blockers are found, the commit is rejected and the report path is printed.

---

## Using with other agents

The same approach works for any agent that supports a `CLAUDE.md` / `AGENTS.md` / `.cursorrules` convention:

**Copilot / Cursor / Windsurf** — add to `.cursorrules` or `AGENTS.md`:
```
Before committing: run `coderev .` and fix all blockers.
```

**CI (GitHub Actions)** — the Docker image runs coderev as an isolated step:
```yaml
docker run --rm -v "$GITHUB_WORKSPACE:/src" -e GH_TOKEN \
  ghcr.io/srivastava-ami/coderev:latest \
  --diff "origin/${BASE_REF}" --annotate-pr \
  --repo "${PR_REPO}" --pr "${PR_NUMBER}" /src
```

The agent posting a PR still runs in its own process. coderev posts inline comments independently via `gh api` — the two never share a context window.

---

## What coderev does NOT do

- It does not call any LLM or external API.
- It does not stream output into the agent context.
- It does not require Claude or any AI to function — it runs as a plain binary.

The agent's role is only: trigger the command, read the report file, act on findings.
