# Convenience wrapper around scripts/ and the Go toolchain. The scripts stay
# authoritative; every target here invokes one, or the command CI runs.
#
#   make config     submodules and module downloads
#   make build      both binaries: the coordinator and scpd
#   make help       list every target
#
# Not needed for a local devnode: hog-devnode is a single process that the
# groundhog CLI supervises directly. chuckd and scpd are the multi-process
# validator's coordinator and SCP daemon.

SHELL := /bin/bash
.DEFAULT_GOAL := help
.NOTPARALLEL:

# Reaches config, test, lint and fmt only. The build targets hand off to
# scripts/build_chuckd.sh, which invokes `go build` directly, so overriding this
# does not change which toolchain builds the binaries.
GO ?= go

# The module and its dependencies live in private repos, so the toolchain must
# skip the public proxy and checksum database. scripts/setup_go.sh sets the
# same three plus a repo-local GOPATH; source it instead if you want the
# module cache kept inside the checkout.
export GOPRIVATE ?= github.com/groundhogchain/*
export GONOSUMDB ?= github.com/groundhogchain/*
export GONOPROXY ?= github.com/groundhogchain/*

# Release builds set this; scripts/build_chuckd.sh passes it to the linker as
# -X github.com/groundhogchain/chuckd.distributionVersion, which is the Go
# equivalent of the generated version.c that hogd compiles in.
export CHUCKD_DISTRIBUTION_VERSION ?=

# The build scripts drop their output at the repo root with a .dev suffix.
CHUCKD_BIN := chuckd.dev
SCPD_BIN := scpd.dev

PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
DESTDIR ?=

.PHONY: help
help:
	@echo "Setup"
	@echo "  config        git submodules (SSH-only) plus go mod download"
	@echo "  submodules    scp and hogx, needed for scpd"
	@echo ""
	@echo "Build"
	@echo "  build         both binaries"
	@echo "  coordinator   just chuckd -> ./$(CHUCKD_BIN)"
	@echo "  scpd          just the SCP daemon, needs the scp submodule -> ./$(SCPD_BIN)"
	@echo ""
	@echo "Verify"
	@echo "  check         what CI enforces: the test suite"
	@echo "  test          go test ./..."
	@echo "  lint          go vet ./..."
	@echo "  fmt           rewrite formatting; fmt-check reports instead"
	@echo "  version       distribution version, commit, and built binaries"
	@echo ""
	@echo "Install"
	@echo "  install       copy built binaries to $(BINDIR) under their operator names"
	@echo ""
	@echo "Clean"
	@echo "  clean         remove the built binaries"
	@echo ""
	@echo "Variables: PREFIX BINDIR DESTDIR CHUCKD_DISTRIBUTION_VERSION"
	@echo "  GO covers config, test, lint and fmt; the build scripts call go themselves"

# Both submodule remotes are git@github.com, so this needs an SSH agent holding
# a key GitHub accepts. scpd cannot build without the scp submodule.
.PHONY: submodules
submodules:
	git submodule update --init --recursive

.PHONY: config
config: submodules
	$(GO) mod download

.PHONY: build
build: coordinator scpd

.PHONY: coordinator
coordinator:
	./scripts/build_chuckd.sh

.PHONY: scpd
scpd:
	./scripts/build_scpd.sh

# CI runs go test with coverage and nothing else (scripts/schedules/
# build_test_cover.toml), so check is the test suite. lint and fmt-check are
# here for parity with the sibling repos and both report pre-existing findings
# in generated code, which is why check does not depend on them.
.PHONY: check
check: test

.PHONY: test
test:
	$(GO) test ./...

.PHONY: lint
lint:
	$(GO) vet ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

# git ls-files scopes this to the repo's own sources; a plain `gofmt -l .`
# walks into the scp and hogx submodules and reports their formatting too.
.PHONY: fmt-check
fmt-check:
	@out=$$(gofmt -l $$(git ls-files '*.go')); \
	if [ -n "$$out" ]; then echo "$$out"; exit 1; fi

.PHONY: version
version:
	@echo "source tree"
	@echo "  go module          $$(head -1 go.mod | awk '{print $$2}')"
	@echo "  go directive       $$(grep -m1 '^go ' go.mod | awk '{print $$2}')"
	@echo "  CHUCKD_DISTRIBUTION_VERSION  $${CHUCKD_DISTRIBUTION_VERSION:-(unset)}"
	@echo "  commit             $$(git rev-parse --short=12 HEAD)$$(git update-index --refresh >/dev/null 2>&1; git diff-index --quiet HEAD -- 2>/dev/null || echo '+dirty')"
	@echo "binaries"
	@for b in $(CHUCKD_BIN) $(SCPD_BIN); do \
		if out=$$(./$$b --version 2>/dev/null); then \
			echo "  $$b: $$out"; \
		else \
			echo "  $$b: not built"; \
		fi; \
	done

# Installed under the operator-facing names, without the .dev suffix the build
# scripts use to mark a local build.
.PHONY: install
install:
	@installed=0; \
	mkdir -p "$(DESTDIR)$(BINDIR)"; \
	if [ -f $(CHUCKD_BIN) ]; then \
		install -m 755 $(CHUCKD_BIN) "$(DESTDIR)$(BINDIR)/chuckd" || exit 1; \
		echo "  $(DESTDIR)$(BINDIR)/chuckd"; installed=1; \
	fi; \
	if [ -f $(SCPD_BIN) ]; then \
		install -m 755 $(SCPD_BIN) "$(DESTDIR)$(BINDIR)/scpd" || exit 1; \
		echo "  $(DESTDIR)$(BINDIR)/scpd"; installed=1; \
	fi; \
	if [ $$installed -eq 0 ]; then \
		echo "nothing built to install; run 'make build' first" >&2; exit 1; \
	fi

.PHONY: clean
clean:
	rm -f $(CHUCKD_BIN) $(SCPD_BIN)
