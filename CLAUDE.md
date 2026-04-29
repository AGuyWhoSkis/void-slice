# void-slice

Linter for Dishonored 2 / DOTO game files (`.entities`, `.decl`, `.entitydef`). Reads game-exported text archives, tokenizes them, validates structural invariants, and renders diagnostics.

## Project layout

| Path | Purpose |
|------|---------|
| `cmd/voidslice/` | CLI entry point (`main.go`) — T5 pending |
| `internal/scan/` | Tokenizer: `[]byte → []Token + []Diagnostic + []int (newline offsets)` |
| `internal/parse/` | Structural parser: token stream → events via `Handler` interface |
| `internal/validate/` | Semantic validator: implements `parse.Handler`, emits `VALIDATE_*` diagnostics |
| `internal/report/` | Report renderer: human-pretty and JSON output from `[]scan.Diagnostic` |
| `internal/lint/` | Lint facade (T4 — pending) |
| `kanban/` | Markdown task board — see § Kanban workflow |
| `testdata/` | Fixture files for unit tests; committed binaries under `testdata/binary/`; committed mini corpus of real game files under `testdata/corpus-mini/` |

## Key abstractions

**`scan.Scan(src []byte) ([]Token, []Diagnostic, []int)`** — tokenizer. Returns tokens in source order, diagnostics for lexical errors, and raw newline byte offsets. Spans are half-open `[Start, End)` byte offsets.

**`scan.Token{Kind Kind, Span Span}`** — one lexical unit. Kind values: `SYMBOL`, `IDENTIFIER`, `QUOTE_LITERAL`, `NUMBER_LITERAL`, `COMMENT_BLOCK`, `COMMENT_LINE`.

**`scan.Diagnostic{Code DiagnosticCode, Span Span, Message string}`** — one lint finding. Code prefix determines report severity: `VALIDATE_*` → warning, everything else → error.

**`parse.Handler`** — visitor interface. Implement to receive structural events from `WalkEntities`: `OnAssignment`, `OnObjectBegin`, `OnObjectEnd`, `OnComponentBegin`, `OnComponentEnd`, `OnTypedBlock`, `OnVersion`, `OnDiag`.

**`parse.WalkEntities(src []byte, toks []scan.Token, h Handler) []Diagnostic`** — streams events to `Handler`. `validate.ValidateEntities` wraps this.

**`report.Render(filename, src, diags, opts)`** — human-pretty multi-line output with source context and caret underlines.

**`report.RenderJSON(filename, src, diags)`** — JSON output (`{file, diagnostics: [{line, col, severity, code, message}]}`).

## Conventions

- **Option A punctuation:** token `Kind == SYMBOL`; caller reads `src[t.Span.Start]` for the actual byte (`{`, `}`, `=`, `;`, etc.). Do not split `SYMBOL` into subtypes.
- **Streaming parse:** no full-file AST. `WalkEntities` processes one component at a time so the validator can process and discard state as it goes.
- **Golden-file tests:** expected output lives under `testdata/golden/` as `.txt` files; diff with `testify/assert`.
- **Test file placement:** `*_test.go` co-located with the package under test.
- **No external deps** beyond `github.com/stretchr/testify`.
- **Commit messages:** keep them concise — a short subject line is usually enough. Reach for a body only when the *why* genuinely needs explanation. The log is for skimming, not reading.

## Definition of done

A ticket is complete when all of the following are true:

1. All scope items in the ticket are addressed.
2. `go test ./...` passes with no failures.
3. `go vet ./...` reports no issues.
4. The ticket's `**Status:**` field is set to `done` and a `## Completion` section is appended.

## What not to touch

- `testdata/binary/` — committed binary fixtures. Replace only when updating expected scanner output for a known change.

## Kanban workflow

Tickets live in `kanban/todo/<version>/`, `kanban/in-progress/<version>/`, or `kanban/done/<version>/`.

**To change a ticket's status:** edit only the `**Status:**` field in the ticket file (valid values: `todo`, `in-progress`, `done`). The kanban-move hook automatically runs `git mv` to move the file to the correct folder. Do not manually move ticket files. **Always use the Edit or Write tool for any kanban file change** — Bash writes (e.g. `cat >>`) bypass the hook and leave the file in the wrong folder.

Use `/crud-ticket <ticket>` and `/implement-ticket <ticket>` for common operations (see § Tooling).

Version subfolders: `meta`, `v1`, `v2`, `v3`, `stretch`.

## Tooling

**Test-runner hook (T18):** `PostToolUse` on Edit/Write of `*.go` files → runs `go test ./...` automatically. Result appears in the conversation. Takes effect in a new session after `.claude/settings.json` was created.

**Kanban-move hook (T23):** `PostToolUse` on Edit/Write of `kanban/**/*.md` → reads the `**Status:**` field and runs `git mv` to move the ticket to the matching folder. Takes effect in a new session.

**Slash commands (T17):**
- `/crud-ticket <ticket>` — scaffold for creating, updating, or closing a ticket
- `/implement-ticket <ticket>` — full plan → implement → verify loop for a ticket

**Pre-approved commands (T19):** run without an approval prompt: `go test ./...`, `go test <pkg>`, `go build ./...`, `go vet ./...`, `grep`, `find`, `ls`. Still gated: `rm`, `git reset --hard`, `git push --force`, `git branch -D`.

**Lint (T33):** `make lint` runs `golangci-lint run --timeout=5m` against the same pinned version the CI `lint` job uses. On first invocation, the target self-installs the binary to `$(go env GOPATH)/bin` if it isn't already on PATH, so a fresh checkout needs no extra setup. The pinned version is the `GOLANGCI_LINT_VERSION` variable at the top of the `lint` target in `Makefile`; if you bump it, also update the `version:` arg in [.github/workflows/ci.yml](.github/workflows/ci.yml) to keep local and CI aligned.

**Worktrees (T16):** to run two Claude Code sessions in parallel on separate branches:
```bash
# Preferred — automatic setup/teardown:
claude --worktree <branch>

# Manual:
git worktree add ../void-slice-<branch> <branch>
git worktree remove ../void-slice-<branch>
git worktree prune
```
Worktrees land as siblings to the repo root (`../void-slice-<branch>`). `.claude/` settings are git-tracked; each branch has its own copy.

**v2/v3 parallel session setup:**
```bash
# Create milestone branches from main
git checkout -b v2-dev && git checkout main
git checkout -b v3-dev && git checkout main

# Terminal 1 — works T8→T9→T10→T11→T12→T13:
claude --worktree v2-dev

# Terminal 2 — works T14:
claude --worktree v3-dev
```
Merge order: v3-dev first (smaller delta), then v2-dev. Expected conflict: `cmd/voidslice/main.go` — both branches add a new subcommand (`serve` and `lsp`). Resolve by keeping both cases.

**Subagents (T22):** **Adopt with caveats.** Parallel subagents provide ~3× speedup on file-disjoint write tasks with zero merge friction. Restrictions:
- Only for tasks with truly file-disjoint subtasks (no shared files, no git operations during parallel phase).
- Subagents cannot access worktrees outside the project directory (`/tmp/` is blocked); place worktrees inside `/workspaces/void-slice/` or skip isolation and write directly to distinct files.
- Each agent must be self-contained — no coordination or shared state between agents.
- Budget for post-agent integration overhead (testing, corrections) in wall-clock estimates.

## Dev setup

**Devcontainer (T34):** `.devcontainer/` is tracked in git. A fresh clone gets a working devcontainer out of the box — no manual bootstrap from the upstream Anthropic template required. The three files are:

- [.devcontainer/Dockerfile](.devcontainer/Dockerfile) — base image, system packages, Go/Claude Code/UV/chezmoi installs
- [.devcontainer/devcontainer.json](.devcontainer/devcontainer.json) — VS Code extensions, host bind-mounts, port forwards, post-start firewall hook
- [.devcontainer/init-firewall.sh](.devcontainer/init-firewall.sh) — iptables/ipset firewall with an allowlist of public domains (GitHub, npm, Anthropic, VS Code marketplace, Go module proxy, Copilot)

**Syncing upstream template updates:** the upstream Anthropic devcontainer lives at [github.com/anthropics/claude-code/tree/main/.devcontainer](https://github.com/anthropics/claude-code/tree/main/.devcontainer). To take an update, diff our tracked copy against the upstream tree and cherry-pick what's wanted. The leading `# Changes from upstream …` header in each file is a load-bearing sync log — append new deltas to it whenever you edit and preserve the existing entries so the next sync is mechanical. Do not silently re-base onto upstream; the log is how a future maintainer reconstructs why each line diverges.

**Adding new container infra:** prefer putting the install in `.devcontainer/Dockerfile` (where Go, UV, chezmoi, and Claude Code already live) rather than routing it through `Makefile` self-bootstrap. Self-bootstrap (e.g. T33's `make lint`) is still useful as a safety net for contributors who run outside the devcontainer, but the Dockerfile is now the primary delivery path for container-resident tools.

**Container auth (T21):** `~/.claude/` from the WSL2 host is bind-mounted into the container at `/home/node/.claude` via `devcontainer.json`. OAuth credentials persist across container restarts and rebuilds automatically.

If credentials expire: re-authenticate on the **host** (`claude auth login` in a WSL2 terminal outside the container). The container reads credentials from the bind-mount — no container rebuild needed.

**Corpus:** real game-file fixtures live under `testdata/corpus-mini/` (committed, ~7.7 MB). The six files mirror the `goldenFileNames` list in `internal/scan/scan_test.go` (paths under `d2/game1/...` and `doto/game1/...`); the scan tests fail hard if any are missing. The same tree also feeds the lint sweeps (`internal/lint/` `TestCleanSweep`, `TestCoverageAudit`), and `testdata/binary/sample.bwm` feeds `TestBinarySweep`. To add a new golden, drop the file under the matching subpath, add it to `goldenFileNames`, and commit. The previous `void-files/` corpus (~2.2 GB of extracted game data) has been retired — the larger sweep was redistribution-blocked and CI couldn't see it anyway.
