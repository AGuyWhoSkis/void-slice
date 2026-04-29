# Void Slice

A modding linter for Dishonored 2 and DOTO, shipped as a CLI and a web playground.

## What it does

[Void Explorer](https://www.nexusmods.com/dishonored2/mods/9) enables patch-based modding of code and game assets — Export, Import, Generate Mod. No tooling exists yet for parsing, linting, or verifying the compiled id Tech files those workflows produce. Void Slice fills that gap.

In v1, Void Slice lints individual game files (`.entities`, `.decl`, `.entitydef`) and reports diagnostics:

- **Bracket and quote parity** — unmatched `[] {} ()` and `"" ''`
- **Count/index validation** — `count = X;` doesn't match the number of entries below it
- **File reference checks** — invalid references and misuse of `NULL;`

Verification (linting all files in a mod) and D2/DOTO comparison are roadmapped for later milestones, not v1.

## Architecture

```
            ┌────────────────────┐
            │    lint package    │   pure Go library, no transport
            │    (the engine)    │
            └─────────┬──────────┘
                      │
        ┌─────────────┼─────────────┐
        ▼             ▼             ▼
   ┌─────────┐  ┌──────────┐  ┌──────────┐
   │   CLI   │  │  server  │  │   LSP    │   ← v2, designed for now
   │voidslice│  │ POST     │  │ (later)  │
   │  lint   │  │ /lint    │  │          │
   └─────────┘  └────┬─────┘  └──────────┘
                     │
                     ▼
            ┌─────────────────┐
            │ Frontend        │
            │ React +         │
            │ CodeMirror      │
            └─────────────────┘

  Deploy:  Pages (frontend)  →  Worker (router)  →  Container (Go binary)
```

Three transport layers share one engine. The LSP server (v2) is not a rewrite — it's a new file that imports `lint` and speaks JSON-RPC instead of HTTP.

## Stack

- **Go** — core lint engine, CLI, HTTP server (single binary)
- **React + Vite + CodeMirror** — web playground frontend
- **Cloudflare Pages + Workers + Containers** — deploy target
- **docker-compose** — full local stack (`docker compose up`)

## Status

v1 (linter engine + CLI) is in progress. See [`kanban/`](kanban/README.md) for current ticket status and the v2/v3 roadmap.

## Contributing

Non-code contributions (broken fixture files, bug reports, doc fixes) are just as valuable as code. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to get started and what's ready to pick up.

To discuss contributing or the project in general, reach out on [Nexus Mods](https://www.nexusmods.com/profile/kleptobismal).

