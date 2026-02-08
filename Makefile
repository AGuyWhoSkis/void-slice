# Makefile for a basic Go project (Go 1.23.5)

SHELL := /usr/bin/bash
.ONESHELL:
.SHELLFLAGS := -euo pipefail -c

GO      ?= go
BIN_DIR ?= bin
CMD ?=./cmd/void-slice

.PHONY: help tidy fmt vet test race cover build run clean

help:
	@echo "Targets:"
	@echo "  tidy   - go mod tidy"
	@echo "  fmt    - gofmt on all .go files"
	@echo "  vet    - go vet ./..."
	@echo "  test   - go test ./..."
	@echo "  race   - go test -race ./..."
	@echo "  cover  - run tests with coverage and open a local HTML report"
	@echo "  build  - build binaries (defaults to ./cmd/* if CMD not set)"
	@echo "  run    - run (requires CMD=./cmd/<app> or adjust to your layout)"
	@echo "  clean  - remove build artifacts"

tidy:
	$(GO) mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

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
