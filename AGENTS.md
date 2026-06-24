# Adding coderev to an AI agent workflow

`coderev` is a standalone binary — it runs as a shell command and produces a structured report file. It does not call any LLM, consume agent tokens, or require network access.

---

## Drop into any agent's instruction file

Add this to your `AGENTS.md`, `CLAUDE.md`, `.cursorrules`, or equivalent:

```
## Quality gate
Before every commit: run `coderev .`
Report: coderev-report.md
Fix all blockers. Advisory findings must be addressed or justified in the PR.
```

The agent runs `coderev .` as a shell command. The binary does the entire analysis and writes the report. The agent reads the file when it needs to act on findings — no token streaming.

---

## Reading findings as an agent

```bash
# incremental — only files changed since main
coderev --diff main . --output /tmp/findings.md

# agent reads /tmp/findings.md
# format: rule ID | file | line | severity | remediation
```

Every finding in the Markdown report is a self-contained record. The agent reads the report in one shot and acts on it.

---

## Inline PR comments (no agent involvement needed)

```bash
coderev --annotate-pr --diff main .
```

coderev posts inline comments on the PR directly via the `gh` CLI. The agent does not need to relay findings — they go straight to GitHub.

In CI, pass `--repo` and `--pr` explicitly (the checkout may be detached):

```bash
coderev --annotate-pr --diff "origin/${BASE_REF}" \
  --repo "${PR_REPO}" --pr "${PR_NUMBER}" .
```

---

## Installing the pre-commit hook

```bash
coderev install-hooks
```

After this the agent never needs to remember to run coderev — the hook fires on every `git commit` and blocks the commit if blockers are found.

---

## What the agent should never do

- Do not pipe `coderev` output directly into the context window — read the report file instead.
- Do not re-run coderev for every file individually — one `coderev .` or `coderev --diff <ref> .` covers everything.
- Do not pass `--no-verify` to skip the pre-commit hook unless explicitly asked by the user.
