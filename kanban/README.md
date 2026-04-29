# Kanban Board

## Columns

| Folder | Meaning |
|--------|---------|
| `todo/v1/`, `todo/v2/`, `todo/v3/`, `todo/stretch/` | Prioritized and ready to start, grouped by milestone |
| `todo/meta/` | Workflow and tooling improvements (meta-tickets about the dev process itself) |
| `in-progress/<version>/` | Being worked on right now (move file here when you start) |
| `done/<version>/` | Finished — move file here, add a completion note at the bottom |

## Ticket format

Each ticket is a markdown file. Filename: `T<n>-short-name.md`, stored under the versioned subfolder matching its milestone.

Edit the `**Status:**` field (`todo` / `in-progress` / `done`) — the kanban-move hook moves the file into the matching folder. Do not rename or `git mv` manually.

## v1 — Linter engine + CLI

v1 critical path is complete: parser, validator, report rendering, lint facade, CLI, scan tests, and corpus integration tests have all landed. Only T24 (CLI polish — color, multi-file, `--strict`) remains as cleanup.

| Ticket | What | Size |
|--------|------|------|
| T0 ✓ | Project scaffold — rename cmd/, create internal/lint stub | small |
| T1 ✓ | Parser — implement cursor helpers + WalkEntities | large |
| T2 ✓ | Validator — array count/index rules | large |
| T3 ✓ | Report rendering — human-pretty + JSON output | medium |
| T4 ✓ | Lint facade — single engine API, file classification, binary sniff | medium |
| T5 ✓ | CLI — `voidslice lint <file> [--json]` | medium |
| T6 ✓ | Testdata — broken .decl fixture files + expected output snapshots | small |
| T7 ✓ | Integration tests — run linter against void-files/ corpus | small |
| T15 ✓ | Scan package — TDD test coverage expansion for internal/scan | large |
| T24 | CLI polish — ANSI color, multi-file args, `--strict` flag | small |

---

## v2 — HTTP server + React playground

Backend, frontend, container, CI/CD, WASM compile spike, and resource profile have all landed (merged from `worktree-v2-dev`). T12 (production deploy to Cloudflare) and T13 (docs polish) remain — see `cloudflare.md` at the repo root for the T12 handover doc.

| Ticket | What | Size |
|--------|------|------|
| T8 ✓ | HTTP server — `POST /lint`, CORS, rate limits, health check | medium |
| T9 ✓ | Frontend playground — React + Vite + CodeMirror | large |
| T10 ✓ | Containerization — Dockerfile (distroless, <20MB) + docker-compose | small |
| T11 ✓ | CI/CD — GitHub Actions + Wrangler config | medium |
| T12 | Production deploy — Cloudflare Pages + Worker + Container, custom domain | small |
| T13 | Docs polish — README → landing page, architecture page, LSP design doc | small |
| T25 ✓ | WASM compile spike — determine if linter compiles to wasip1/wasm for Workers | small |
| T26 ✓ | Linter resource profile — benchmark memory + wall-clock on void-files/ corpus | small |

---

## v3 — LSP server

v3 is complete: the JSON-RPC LSP server, subprocess integration tests, and a buildable VS Code extension all shipped (merged via `worktree-v3-dev`). Follow-up gaps (Neovim/Helix/Zed client docs, VS Code e2e smoke test, server stderr logging, release pipeline for binary + .vsix) are tracked in `todo/stretch/T-v3-followups.md`.

| Ticket | What | Size |
|--------|------|------|
| T14 ✓ | LSP server — JSON-RPC 2.0 over stdio, wraps internal/lint | large |
| T27 ✓ | LSP integration tests — subprocess harness against the corpus | small |
| T28 ✓ | VS Code extension — `voidslice-vscode/` packaged extension | small |

---

## Stretch / backlog

Evaluate after the relevant milestone; start only if there is velocity buffer.

| Ticket | What | Gate |
|--------|------|------|
| T-null-ref | NULL; reference validation — third lint rule | after T7 corpus sweep shows low false-positive rate |
| T-k3d | k3d Kubernetes lab | after end-of-week-1 velocity check |
| T-v3-followups | LSP umbrella: editor-setup docs, VS Code e2e test, stderr logging, release pipeline | evaluate after v2 lands |
