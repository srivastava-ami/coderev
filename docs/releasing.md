# Release process

## Overview

Releases are triggered by pushing a `v*` tag (e.g. `v1.2.3`). The `release.yml` GitHub Actions workflow:

1. Runs `make dist` — cross-compiles for darwin/linux/windows × amd64/arm64
2. Creates a GitHub Release with all binaries attached
3. Updates `Formula/coderev.rb` — replaces placeholder `SHA256_*` values with real ones
4. The Homebrew tap `srivastava-ami/coderev` picks up the updated formula automatically via GitHub's formula tracking

Docker images are published to `ghcr.io/srivastava-ami/coderev` on tag push via a separate `docker-publish.yml` workflow.

---

## Step-by-step

```bash
# 1. Ensure main is up to date and clean
git checkout main
git pull origin main
make dist         # verify all platforms build
go test ./...     # all tests pass
coderev .         # 0 blockers — dogfood is mandatory

# 2. Tag and push
VERSION="v1.2.3"          # ← use next semantic version
git tag -a "${VERSION}" -m "Release ${VERSION}"
git push origin "${VERSION}"
```

Then:

1. Open https://github.com/srivastava-ami/coderev/actions — verify `release.yml` completes (≈2 min)
2. Open https://github.com/srivastava-ami/coderev/releases — confirm binaries are attached
3. Open https://github.com/srivastava-ami/coderev/blob/main/Formula/coderev.rb — confirm SHAs are updated
4. Open https://github.com/srivastava-ami/srivastava-ami-homebrew-tap (or similar) — confirm formula is synced
5. Test the install:
   ```bash
   brew tap srivastava-ami/coderev
   brew upgrade coderev
   coderev --version   # should show v1.2.3
   ```

---

## What each distribution channel does

### Homebrew

- Formula at `Formula/coderev.rb`
- Contains version and platform-specific SHA256 checksums
- `release.yml` overwrites `SHA256_DARWIN_ARM64`, `SHA256_DARWIN_AMD64`, etc. with real values after `make dist`
- The tap repo must update its copy of the formula — this is automatic if the tap is a submodule or GitHub tracks it

### curl installer

- Script at `scripts/install.sh`
- Fetches latest release from GitHub API, downloads matching platform binary, installs to `~/.local/bin/`
- No SHA verification — users trust HTTPS + GitHub TLS
- Always installs from latest stable release, so no per-version changes needed

### Docker / GHCR

- Built by `.github/workflows/docker-publish.yml`
- Builds multi-arch images (`linux/amd64`, `linux/arm64`)
- Published under `ghcr.io/srivastava-ami/coderev`
- Tagged `latest` and with the git tag

### `make install`

- From source: requires Go 1.22+
- `go build` + copy to `/usr/local/bin/`
- No version check; uses whatever tree you have checked out

---

## Prerequisites checklist

- [ ] Go 1.22+ installed
- [ ] `git tag` pushed (must be `v*` pattern)
- [ ] GitHub Actions token (`GITHUB_TOKEN`) with write access to releases
- [ ] Docker credentials for `ghcr.io` (if publishing Docker image)
- [ ] Homebrew tap repo exists at `https://github.com/srivastava-ami/homebrew-coderev` (or equivalent)
- [ ] `coderev .` passes with 0 blockers

---

## Versioning

Follow [semantic versioning](https://semver.org/):

- **Major** — breaking changes to the CLI, output format, or TOML config schema
- **Minor** — new rules, new languages, new adapters, new flags (backward-compatible)
- **Patch** — bug fixes, performance improvements, dependency updates

The binary's version string is set by `git describe --tags --always --dirty` and baked in at build time via `-ldflags`.
