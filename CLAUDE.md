# void-slice

Linter for Dishonored 2 / DOTO game files (`.entities`, `.decl`, `.entitydef`). Tokenizes, validates structural invariants, and renders diagnostics.

## Project layout

| Path | Purpose |
|------|---------|
| `cmd/voidslice/` | CLI entry point |
| `internal/scan/` | Tokenizer: `[]byte → []Token + []Diagnostic + []int (newline offsets)` |
| `internal/parse/` | Structural parser: token stream → events via `Handler` |
| `internal/validate/` | Semantic validator: implements `parse.Handler`, emits `VALIDATE_*` |
| `internal/report/` | Renderer: human-pretty + JSON |
| `internal/lint/` | Lint facade |
| `kanban/` | Markdown task board — see § Kanban |
| `testdata/` | Fixtures; binaries under `binary/`, real game files under `corpus-mini/` |

## Key abstractions

- **`scan.Scan(src) → ([]Token, []Diagnostic, []int)`** — tokens in source order, lex diagnostics, newline byte offsets. Spans are half-open `[Start, End)`.
- **`scan.Token{Kind, Span}`** — `SYMBOL` / `IDENTIFIER` / `QUOTE_LITERAL` / `NUMBER_LITERAL` / `COMMENT_BLOCK` / `COMMENT_LINE`.
- **`scan.Diagnostic{Code, Span, Message}`** — `VALIDATE_*` codes render as warning, everything else as error.
- **`parse.Handler`** — visitor: `OnAssignment`, `OnObjectBegin/End`, `OnComponentBegin/End`, `OnTypedBlock`, `OnVersion`, `OnDiag`.
- **`parse.WalkEntities(src, toks, h) → []Diagnostic`** — streams events. `validate.ValidateEntities` wraps it.
- **`report.Render` / `report.RenderJSON`** — pretty (with carets) and JSON outputs.

## Conventions

- **Option A punctuation:** `Kind == SYMBOL`; caller reads `src[t.Span.Start]` for the byte. Don't subtype.
- **Streaming parse:** no full-file AST; `WalkEntities` processes one component at a time.
- **Golden-file tests:** expected output under `testdata/golden/*.txt`; diff with `testify/assert`. `*_test.go` co-located with package.
- **No external deps** beyond `github.com/stretchr/testify`.
- **Commit messages:** subject only. Body only if the user asks or the *why* can't be inferred from the diff. Bar = `git log --oneline -10`.

## What not to touch

- `testdata/binary/` — committed binary fixtures. Replace only when updating expected scanner output for a known change.

## Kanban

Tickets live in `kanban/{todo,in-progress}/`. Each ticket's filename starts with its goal ID. Status folders are flat. Goal index in [`kanban/goals/`](kanban/goals/).

Tickets are self-contained: creating, editing, or deleting a ticket should never require updating any other file — including the goal file. Goal files do not maintain ticket tables; the kanban folder listing is the source of truth.

To change a ticket's status, edit the `**Status:**` field (`todo` / `in-progress`) using Edit/Write. The kanban-move hook runs `git mv` to the matching folder. Bash writes bypass the hook. Closed tickets are deleted (`git rm`); `git log` is the closure record.

Use `/crud-ticket <ticket>` and `/implement-ticket <ticket>`.

## Tooling

- **Hooks** (`PostToolUse` on Edit/Write):
  - `*.go` → `go test ./...`.
  - `kanban/{todo,in-progress}/*.md` → `git mv` based on `**Status:**`. `kanban/goals/*.md` excluded.
- **Slash commands:** `/crud-ticket`, `/implement-ticket`, `/goal-define`, `/goal-slice`.
- **Pre-approved Bash:** `go test`, `go build ./...`, `go vet ./...`, `grep`, `find`, `ls`. Still gated: `rm`, `git reset --hard`, `git push --force`, `git branch -D`.
- **Lint:** `make lint` runs `golangci-lint` at the version pinned in `Makefile` (self-installs to `$(go env GOPATH)/bin`). Keep `Makefile`'s `GOLANGCI_LINT_VERSION` and the `version:` in [.github/workflows/ci.yml](.github/workflows/ci.yml) in sync.
- **Harnesses:** `make harnesses` runs the per-layer harnesses (M8.1–M8.3) plus the differential oracle (M8.4); CI gates merges and deploys on it alongside `go test ./...`. Use `make layer-harnesses` to skip the oracle. Keep green when touching `worker/`, `cmd/voidslice-wasm/`, or `internal/report`.
- **Worktrees:** `claude --worktree <branch>` (auto setup/teardown), or `git worktree add ../void-slice-<branch> <branch>`. `.claude/` is tracked per branch.
- **Subagents:** ~3× speedup on file-disjoint writes. Self-contained agents only — no shared files, no git ops during parallel phase. Worktrees must live inside `/workspaces/void-slice/` (subagents can't reach `/tmp/`).

## Dev setup

- **Devcontainer:** `.devcontainer/` is tracked. Fresh clone works out of the box.
  - [Dockerfile](.devcontainer/Dockerfile) — base image, Go / Claude Code / UV / chezmoi.
  - [devcontainer.json](.devcontainer/devcontainer.json) — extensions, bind-mounts, port forwards, post-start firewall.
  - [init-firewall.sh](.devcontainer/init-firewall.sh) — iptables/ipset allowlist (GitHub, npm, Anthropic, VS Code marketplace, Go module proxy, Copilot).
  - Add new container deps to the Dockerfile, not Makefile self-bootstrap.
  - Upstream template: [anthropics/claude-code/tree/main/.devcontainer](https://github.com/anthropics/claude-code/tree/main/.devcontainer). The `# Changes from upstream …` header in each file is a load-bearing sync log — append, don't rewrite.
- **Container auth:** `~/.claude/` from the WSL2 host bind-mounts to `/home/node/.claude`; OAuth persists across rebuilds. If creds expire, re-auth on the host.

## Corpus

Real game-file fixtures under `testdata/corpus-mini/` (~7.7 MB committed). The six files mirror `goldenFileNames` in [internal/scan/scan_test.go](internal/scan/scan_test.go) (paths under `d2/game1/…` and `doto/game1/…`); scan tests fail hard if any are missing. The same tree feeds `internal/lint/`'s `TestCleanSweep` / `TestCoverageAudit`; `testdata/binary/sample.bwm` feeds `TestBinarySweep`. To add a golden: drop the file under the matching subpath, add to `goldenFileNames`, commit.
