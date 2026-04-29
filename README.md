# Void Slice

A modding linter for Dishonored 2 and DOTO, shipped as a CLI and a web playground.

## What it does

[Void Explorer](https://www.nexusmods.com/dishonored2/mods/9) enables patch-based modding of code and game assets — Export, Import, Generate Mod. No tooling exists yet for parsing, linting, or verifying the compiled id Tech files those workflows produce. Void Slice fills that gap.

Void Slice lints individual game files (`.entities`, `.decl`, `.entitydef`) and reports diagnostics:

- **Bracket and quote parity** — unmatched `[] {} ()` and `"" ''`, unterminated literals
- **Grammar errors** — missing `;`, missing `{` / `}` / `=`, identifier expected, unterminated objects
- **Array bookkeeping** — `num = X;` doesn't match the number of `item[]` entries; out-of-bounds, duplicate, or missing indices
- **Binary-file refusal** — `.bwm`/`.tome`/etc. and any null-byte content rejected with a clear error
- **`.entities` / `.cfg` round-trip warning** — flags upload of file types Void Explorer cannot reliably re-import

Verification (linting all files in a mod) and D2/DOTO comparison are roadmapped for later milestones.

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
   │   CLI   │  │  server  │  │   LSP    │
   │voidslice│  │  POST    │  │ JSON-RPC │
   │  lint   │  │  /lint   │  │ over io  │
   └─────────┘  └────┬─────┘  └──────────┘
                     │
                     ▼
            ┌─────────────────┐
            │ Frontend        │
            │ React +         │
            │ CodeMirror      │
            └─────────────────┘

  Deploy:  Pages (frontend)  →  Worker (Go compiled to WASM, runs in-isolate)
```

Three transport layers share one engine. The LSP server is not a rewrite — it's a thin entry point that imports `lint` and speaks JSON-RPC.

## Stack

- **Go** — core lint engine, CLI, HTTP server (single binary)
- **React + Vite + CodeMirror** — web playground frontend
- **Cloudflare Pages + Workers + Containers** — deploy target
- **docker-compose** — full local stack (`docker compose up`)

## Status

- **M1** (linter engine + CLI) — shipped; only [M1.10](kanban/todo/M1.10-cli-polish.md) (CLI ergonomics — color, multi-file, `--strict`) remains.
- **M2** (HTTP server + Cloudflare playground) — deployed via Pages + Worker/WASM. [M2.6](kanban/todo/M2.6-docs.md) (landing-page rewrite + LSP design doc) is the last open item.
- **M3** (LSP server + VS Code extension) — shipped; follow-up polish tracked in [stretch.3-v3-followups](kanban/todo/stretch.3-v3-followups.md).
- **M4** (linter scope refinement) — parked; see [M4.1](kanban/todo/M4.1-linter-scope.md).

See [`kanban/`](kanban/README.md) for the full board.

## Contributing

Non-code contributions (broken fixture files, bug reports, doc fixes) are just as valuable as code. See [CONTRIBUTING.md](CONTRIBUTING.md) for how to get started and what's ready to pick up.

To discuss contributing or the project in general, reach out on [Nexus Mods](https://www.nexusmods.com/profile/kleptobismal).

