# Makefile for a basic Go project (Go 1.23.5)

SHELL := /usr/bin/bash
.ONESHELL:
.SHELLFLAGS := -euo pipefail -c

GO      ?= go
BIN_DIR ?= bin
CMD ?=./cmd/voidslice

.PHONY: help tidy fmt vet test race cover build run clean corpus-mini

help:
	@echo "Targets:"
	@echo "  tidy         - go mod tidy"
	@echo "  fmt          - gofmt on all .go files"
	@echo "  vet          - go vet ./..."
	@echo "  test         - go test ./..."
	@echo "  race         - go test -race ./..."
	@echo "  cover        - run tests with coverage and open a local HTML report"
	@echo "  build        - build binaries (defaults to ./cmd/* if CMD not set)"
	@echo "  run          - run (requires CMD=./cmd/<app> or adjust to your layout)"
	@echo "  clean        - remove build artifacts"
	@echo "  corpus-mini  - refresh testdata/corpus-mini/ from a local void-files/ tree"

tidy:
	$(GO) mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	$(GO) vet ./...

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

# Repopulate testdata/corpus-mini/ from a full void-files/ tree (local-only,
# gitignored). Use this only when adding a new golden to scan_test.go's
# goldenFileNames or when the source files in void-files/ change. The mini
# corpus is committed; CI does not run this target.
CORPUS_FILES := \
	d2/game1/generated.decls.gamelogicmanager.ui.gamelogic.manager..gamelogicmanager.decl \
	d2/game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl \
	d2/game1/generated.decls.greatestmomentsmanager.greatestmoments.manager.manager..greatestmomentsmanager.decl \
	d2/game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities \
	doto/game1/generated.decls.physicsmaterial.contactsystem.weapons.decl \
	doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl

corpus-mini:
	@if [[ ! -d void-files ]]; then
		echo "ERROR: void-files/ not present. The full corpus is local-only and not hosted publicly (game data)."
		echo "       Place an extracted void-files/ tree at the repo root, then re-run."
		exit 2
	fi
	@for f in $(CORPUS_FILES); do
		src="void-files/$$f"
		dst="testdata/corpus-mini/$$f"
		if [[ ! -f "$$src" ]]; then
			echo "MISSING in void-files/: $$f"
			exit 2
		fi
		mkdir -p "$$(dirname "$$dst")"
		cp "$$src" "$$dst"
		echo "refreshed $$dst"
	done
