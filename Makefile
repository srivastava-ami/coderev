BIN     := coderev
PKG     := ./cmd/coderev
BINDIR  := bin

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"

PLATFORMS := \
  darwin/arm64 \
  darwin/amd64 \
  linux/amd64 \
  linux/arm64 \
  windows/amd64

.PHONY: build install install-deps install-hooks install-all test dev-scan clean dist docker-build docker-run docker-push $(PLATFORMS)

## build — compile for the current platform into ./bin/coderev
build:
	@mkdir -p $(BINDIR)
	go build $(LDFLAGS) -o $(BINDIR)/$(BIN) $(PKG)

## install — build and copy to /usr/local/bin (or GOPATH/bin as fallback)
install: build
	@if [ -w /usr/local/bin ]; then \
	  cp $(BINDIR)/$(BIN) /usr/local/bin/$(BIN); \
	  echo "installed → /usr/local/bin/$(BIN)"; \
	else \
	  go install $(LDFLAGS) $(PKG); \
	  echo "installed → $$(go env GOPATH)/bin/$(BIN)"; \
	fi

## install-deps — install external scanner dependencies (gitleaks, semgrep, madge)
install-deps: build
	$(BINDIR)/$(BIN) install-deps

## install-hooks — install pre-commit and pre-push git hooks
install-hooks: build
	$(BINDIR)/$(BIN) install-hooks

## install-all — full onboarding: build + install + deps + hooks
install-all: install install-deps install-hooks

## test — run all unit tests
test:
	go test ./...

## dev-scan — rebuild then scan this repo (development workflow)
##            for normal use: coderev .
dev-scan: build
	./$(BINDIR)/$(BIN) .

## clean — remove build artefacts
clean:
	rm -rf $(BINDIR)

## dist — cross-compile release binaries for all platforms into ./bin/dist/
dist:
	@mkdir -p $(BINDIR)/dist
	$(foreach PLATFORM,$(PLATFORMS), \
	  $(eval OS   := $(word 1,$(subst /, ,$(PLATFORM)))) \
	  $(eval ARCH := $(word 2,$(subst /, ,$(PLATFORM)))) \
	  $(eval EXT  := $(if $(filter windows,$(OS)),.exe,)) \
	  GOOS=$(OS) GOARCH=$(ARCH) go build $(LDFLAGS) \
	    -o $(BINDIR)/dist/$(BIN)-$(VERSION)-$(OS)-$(ARCH)$(EXT) $(PKG);)
	@echo "built $(words $(PLATFORMS)) binaries in $(BINDIR)/dist/"
	@ls -lh $(BINDIR)/dist/

## docker-build — build the coderev Docker image locally
docker-build:
	docker build -t coderev:local .

## docker-run — scan current directory via Docker
docker-run: docker-build
	docker run --rm -v "$(PWD):/src" coderev:local /src

## docker-push — push image to GHCR (requires: docker login ghcr.io)
docker-push: docker-build
	docker tag coderev:local ghcr.io/srivastava-ami/coderev:latest
	docker push ghcr.io/srivastava-ami/coderev:latest
