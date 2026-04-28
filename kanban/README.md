# Kanban Board

## Columns

| Folder | Meaning |
|--------|---------|
| `todo/v1/`, `todo/v2/`, `todo/v3/`, `todo/stretch/` | Prioritized and ready to start, grouped by milestone |
| `todo/meta/` | Workflow and tooling improvements (meta-tickets about the dev process itself) |
| `in-progress/` | Being worked on right now (move file here when you start) |
| `done/v1/` | Finished — move file here, add a completion note at the bottom |

## Ticket format

Each ticket is a markdown file. Filename: `T<n>-short-name.md`, stored under the versioned subfolder matching its milestone.

Move the file between `todo/<version>/` → `in-progress/` → `done/<version>/` to track status. Do not rename it.

## v1 — Linter engine + CLI (`todo/v1/`)

Critical path:

```
T0 (scaffold) → T1 (parser) → T2 (validator) ─┐
                T15 (scan tests) ──────────────┤
                                                ├→ T4 (lint facade) → T5 (cli) → T7 (integration tests)
               T3 (report) ────────────────────┘
               T6 (testdata, can run alongside T1)
```

T3, T6, and T15 are done. The remaining critical path is T4 → T5 → T7.  
T7 runs last — it's the end-of-v1 quality gate against real game files.

| Ticket | What | Size |
|--------|------|------|
| T0 ✓ | Project scaffold — rename cmd/, create internal/lint stub | small |
| T1 ✓ | Parser — implement cursor helpers + WalkEntities | large |
| T2 ✓ | Validator — array count/index rules | large |
| T3 ✓ | Report rendering — human-pretty + JSON output | medium |
| T4 | Lint facade — single engine API, file classification, binary sniff | medium |
| T5 | CLI — `voidslice lint <file> [--json]` | medium |
| T6 ✓ | Testdata — broken .decl fixture files + expected output snapshots | small |
| T7 | Integration tests — run linter against void-files/ corpus | small |
| T15 ✓ | Scan package — TDD test coverage expansion for internal/scan | large |

---

## v2 — HTTP server + React playground (`todo/v2/`)

Starts after v1 is complete and deployed locally.

| Ticket | What | Size |
|--------|------|------|
| T8 | HTTP server — `POST /lint`, CORS, rate limits, health check | medium |
| T9 | Frontend playground — React + Vite + CodeMirror | large |
| T10 | Containerization — Dockerfile (distroless, <20MB) + docker-compose | small |
| T11 | CI/CD — GitHub Actions + Wrangler config | medium |
| T12 | Production deploy — Cloudflare Pages + Worker + Container, custom domain | small |
| T13 | Docs polish — README → landing page, architecture page, LSP design doc | small |
| T25 | WASM compile spike — determine if linter compiles to wasip1/wasm for Workers | small |
| T26 | Linter resource profile — benchmark memory + wall-clock on void-files/ corpus | small |

---

## v3 — LSP server (`todo/v3/`)

Starts after v2. Does not depend on any v2 ticket except T4 (lint facade).

| Ticket | What | Size |
|--------|------|------|
| T14 | LSP server — JSON-RPC 2.0 over stdio, wraps internal/lint | large |

---

## Stretch / backlog (`todo/stretch/`)

Evaluate after the relevant v1 milestone; start only if there is velocity buffer.

| Ticket | What | Gate |
|--------|------|------|
| T-null-ref | NULL; reference validation — third lint rule | after T7 corpus sweep shows low false-positive rate |
| T-k3d | k3d Kubernetes lab | after end-of-week-1 velocity check |
