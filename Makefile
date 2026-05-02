# Makefile for a basic Go project (Go 1.23.5)

SHELL := /usr/bin/bash
.ONESHELL:
.SHELLFLAGS := -euo pipefail -c

GO      ?= go
BIN_DIR ?= bin
CMD ?=./cmd/voidslice

.PHONY: help tidy fmt vet lint test race cover build run clean wasm wasm-harness worker-harness web-harness harnesses

help:
	@echo "Targets:"
	@echo "  tidy         - go mod tidy"
	@echo "  fmt          - gofmt on all .go files"
	@echo "  vet          - go vet ./..."
	@echo "  lint         - golangci-lint run --timeout=5m (mirrors CI's lint job)"
	@echo "  test         - go test ./..."
	@echo "  race         - go test -race ./..."
	@echo "  cover        - run tests with coverage and open a local HTML report"
	@echo "  build        - build binaries (defaults to ./cmd/* if CMD not set)"
	@echo "  run          - run (requires CMD=./cmd/<app> or adjust to your layout)"
	@echo "  wasm           - build worker/voidslice.wasm via worker/build.sh"
	@echo "  wasm-harness   - run the WASM-boundary harness (M8.1) against a fresh wasm build"
	@echo "  worker-harness - run the Worker-glue harness (M8.2) in Miniflare against a fresh wasm build"
	@echo "  web-harness    - run the frontend-transport harness (M8.3) via vitest in web/"
	@echo "  harnesses      - run wasm-harness, worker-harness, and web-harness"
	@echo "  clean          - remove build artifacts"

tidy:
	$(GO) mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	$(GO) vet ./...

# Mirrors the `golangci-lint` job in .github/workflows/ci.yml. Bootstraps
# golangci-lint at the pinned CI version into $GOBIN on first run if missing,
# so a fresh checkout (devcontainer or otherwise) needs no extra setup.
# Keep GOLANGCI_LINT_VERSION aligned with the `version:` arg in
# .github/workflows/ci.yml so local and CI run the same binary.
GOLANGCI_LINT_VERSION ?= v1.64.8

lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then
		echo "golangci-lint not found — installing $(GOLANGCI_LINT_VERSION) to $$($(GO) env GOPATH)/bin ..."
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
			| sh -s -- -b "$$($(GO) env GOPATH)/bin" "$(GOLANGCI_LINT_VERSION)"
	fi
	golangci-lint run --timeout=5m

test:
	$(GO) test -v ./...

race:
	$(GO) test -race ./...

cover:
	$(GO) test ./... -coverprofile=$(BIN_DIR)/coverage.out
	$(GO) tool cover -html=$(BIN_DIR)/coverage.out -o $(BIN_DIR)/coverage.html
	@echo "Wrote $(BIN_DIR)/coverage.html"

build:
	mkdir -p "$(BIN_DIR)"
	if [[ -n "$(CMD)" ]]; then
		# Build exactly one main package
		name="$$(basename "$(CMD)")"
		$(GO) build -o "$(BIN_DIR)/$$name" "$(CMD)"
	else
		# Build all main packages under ./cmd/*
		for d in ./cmd/*; do
			[[ -d "$$d" ]] || continue
			name="$$(basename "$$d")"
			$(GO) build -o "$(BIN_DIR)/$$name" "$$d"
		done
	fi

run:
	@if [[ -z "$(CMD)" ]]; then \
		echo "ERROR: set CMD to your main package, e.g.: make run CMD=./cmd/<app>"; \
		exit 2; \
	fi
	$(GO) run "$(CMD)"

clean:
	rm -rf "$(BIN_DIR)"

wasm:
	bash worker/build.sh

# WASM-boundary harness (M8.1). Owns its own toolchain (Node + wasm_exec.js)
# and is intentionally NOT wired into `go test ./...` — the Go suite stays
# self-contained.
wasm-harness: wasm
	node worker/harness/harness.mjs

# Install npm deps for the Worker-glue harness (M8.2). Tracks package-lock.json
# so subsequent `make worker-harness` runs are no-ops.
node_modules: package.json package-lock.json
	npm ci

# Worker-glue harness (M8.2). Boots worker/index.js inside Miniflare 3 and
# asserts every branch of the router and handleLint. Offline; no Cloudflare
# account required. Like wasm-harness, NOT part of `go test ./...`.
worker-harness: wasm node_modules
	node worker/harness/worker-harness.mjs

# Install npm deps for the frontend-transport harness (M8.3). Tracks
# web/package-lock.json so subsequent runs are no-ops.
web/node_modules: web/package.json web/package-lock.json
	npm --prefix web ci

# Frontend-transport harness (M8.3). Pins web/src/api.ts's lintFile() against
# a stubbed globalThis.fetch via vitest. No React, no DOM, no network.
# Like the other harnesses, NOT part of `go test ./...`.
web-harness: web/node_modules
	npm --prefix web test

# Run all middle-layer harnesses end-to-end.
harnesses: wasm-harness worker-harness web-harness
