# Makefile for a basic Go project (Go 1.23.5)

SHELL := /usr/bin/bash
.ONESHELL:
.SHELLFLAGS := -euo pipefail -c

GO      ?= go
BIN_DIR ?= bin
CMD ?=./cmd/voidslice

.PHONY: help tidy fmt vet lint test race cover build run clean

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
	@echo "  clean        - remove build artifacts"

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
