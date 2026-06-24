# ── Stage 1: build coderev ───────────────────────────────────────────────────
FROM golang:1.22-bookworm AS builder

WORKDIR /src

# Cache module downloads separately from source
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=1 required for tree-sitter; bookworm ships gcc
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-s -w" -o /coderev ./cmd/coderev

# ── Stage 2: runtime ─────────────────────────────────────────────────────────
FROM debian:bookworm-slim

ARG GITLEAKS_VERSION=8.30.0
ARG GH_VERSION=2.65.0

# git (--diff mode), Python (semgrep), Node (madge), curl (gitleaks + gh download)
RUN apt-get update && apt-get install -y --no-install-recommends \
      git \
      python3 python3-pip \
      nodejs npm \
      curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# gh CLI — required for --annotate-pr to post inline PR comments
RUN curl -fsSL \
      "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" \
    | tar -xz --strip-components=2 -C /usr/local/bin \
        "gh_${GH_VERSION}_linux_amd64/bin/gh"

# gitleaks — secrets & credential scanning
RUN curl -fsSL \
      "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz" \
    | tar -xz gitleaks \
    && mv gitleaks /usr/local/bin/gitleaks

# semgrep — OWASP / security pattern scanning
RUN pip3 install --no-cache-dir --break-system-packages semgrep

# madge — circular dependency detection for NX / TypeScript
RUN npm install -g madge --quiet

# coderev binary
COPY --from=builder /coderev /usr/local/bin/coderev

# Run as non-root — required for container security best practices
RUN groupadd --gid 1001 coderev \
 && useradd --uid 1001 --gid coderev --shell /bin/bash --create-home coderev

USER coderev
WORKDIR /src

ENTRYPOINT ["coderev"]
CMD ["."]
