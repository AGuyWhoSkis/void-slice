# T20 · Project Root CLAUDE.md

**Status:** done  
**Version:** meta  
**Size:** large

## What

Create `CLAUDE.md` at the project root to capture architectural context that currently lives only in the developer's head. Claude Code loads this file automatically at every session start, eliminating the need to re-establish context from scratch each time.

## Scope

Sections to include:

- **Project layout** — purpose of `cmd/`, `internal/scan`, `internal/lint`, `internal/report`, `kanban/`, `void-files/`, `testdata/`
- **Key abstractions** — `Linter` interface, `Diagnostic` struct, `scan.Span`, severity mapping (`VALIDATE_*` → Warning, everything else → Error)
- **Conventions** — file naming, package boundaries, test file placement, golden-file pattern
- **Definition of done** — what it means for a ticket to be complete (tests passing, `go vet` clean, integration test not regressing)
- **What not to touch** — `void-files/` corpus (read-only real game data), committed binary testdata under `testdata/binary/`
- **Kanban workflow** — folder structure, how to move tickets, ticket format summary
- **Tooling** *(add entries as each meta ticket closes)*:
  - Active hooks: test-runner (T18) and kanban auto-move (T23) — one-line description of each trigger and command
  - Slash commands: `/crud-ticket`, `/implement-ticket` (T17) — one-line description of each
  - Pre-approved shell commands (T19) — what runs without a prompt vs. what stays gated
  - Worktree convention (T16) — where worktree dirs land, how to clean up
  - Subagent pattern (T22) — adopt/reject/caveats decision
- **Dev setup** *(add after T21)*:
  - Container auth solution (bind-mount path or API key env var) and how to re-authenticate if credentials expire

Keep it factual and terse — reference material, not a tutorial.

**Strategy:** Write an initial version immediately covering architecture, abstractions, and conventions (those don't change). Treat it as a living document — add **Tooling** and **Dev setup** entries as T16–T19, T21–T23 are completed rather than waiting for all of them.

## Dependencies

Soft deps on T16, T17, T18, T19, T21, T23 — the architecture/conventions sections can be written immediately; the Tooling and Dev setup sections should be filled in as each tooling ticket closes.

## Verification

Start a fresh Claude Code session in this repo; confirm Claude can answer questions about project layout without reading any files first.

## Completion

Created `CLAUDE.md` at the project root. Covers: project layout, key abstractions (`scan.Scan`, `scan.Token`, `scan.Diagnostic`, `parse.Handler`, `parse.WalkEntities`, `report.Render`/`RenderJSON`), conventions (Option A punctuation, streaming parse, golden-file tests), definition of done, what not to touch, kanban workflow, and a Tooling section reflecting all meta tickets (T16–T23). Dev setup section documents the T21 bind-mount fix. Subagents entry marked pending T22 trial.
