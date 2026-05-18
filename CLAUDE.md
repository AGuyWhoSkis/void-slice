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
| `kanban/goals/` | Durable goal files — see § Goals |
| `testdata/` | Fixtures; binaries under `binary/`, real game files under `golden/` |

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
- **Co-located tests:** `*_test.go` lives with the package it tests; expected values are typically inline in the test file, diffed with `testify/assert`.
- **No external deps** beyond `github.com/stretchr/testify`.
- **Commit messages:** subject only. Body only if the user asks or the *why* can't be inferred from the diff. Bar = `git log --oneline -10`.

## What not to touch

- `testdata/binary/` — committed binary fixtures. Replace only when updating expected scanner output for a known change.

## Goals

Goal files at `kanban/goals/M<N>.md` are the only durable cross-session artifact. They capture *why* a body of work exists and the boundaries it stays within — drafted via `/goal-define`, the user lives at this layer. Small and medium tasks don't need a goal; work directly from conversation. A goal is warranted when work won't fit in a single session and needs the same `why` re-established next time.

### Branches and merge protocol

Each active goal has a branch `goal/M<N>-<slug>` and a single PR against `main`. Create the branch with `git checkout -b goal/M<N>-<slug> origin/main` when work begins. Work commits onto the goal branch; the goal lands as one human-reviewed PR when it closes. Agents push the goal branch and stop — they do not merge to `main`. Branch protection on `main` is the hard backstop: 1 non-author approval and green required checks, no agent bypass. See [`.github/rulesets/main.json`](.github/rulesets/main.json).

### CI-feedback loop

After pushing, stay in the loop until CI is green or you hit something out of scope. Subscribe to the PR via `mcp__github__subscribe_pr_activity` and read check status via `mcp__github__pull_request_read` (`method=get_check_runs`). On failure, classify each failing check as **in-scope** (caused by this session's diff — fix, re-push, re-read) or **out-of-scope** (pre-existing flake, infra, unrelated regression — surface to the user, do not fix). If the same failure shape recurs twice without progress, stop looping and escalate.

## Tooling

- **Hooks** (`PostToolUse` on Edit/Write):
  - `*.go` → `go test ./...`.
- **Slash commands:** `/goal-define`.
- **Pre-approved Bash:** `go test`, `go build ./...`, `go vet ./...`, `grep`, `find`, `ls`. Still gated: `rm`, `git reset --hard`, `git push --force`, `git branch -D`.
- **Lint:** `make lint` runs `golangci-lint` at the version pinned in `Makefile` (self-installs to `$(go env GOPATH)/bin`). Keep `Makefile`'s `GOLANGCI_LINT_VERSION` and the `version:` in [.github/workflows/ci.yml](.github/workflows/ci.yml) in sync.
- **Harnesses:** `make harnesses` runs the per-layer harnesses (M8.1–M8.3) plus the differential oracle (M8.4); CI gates merges and deploys on it alongside `go test ./...`. Use `make layer-harnesses` to skip the oracle. Keep green when touching `worker/`, `cmd/voidslice-wasm/`, or `internal/report`.
- **Lexinvariance gate:** `TestLexinvarianceHardGate` in [`internal/harness/lexinvariance/`](internal/harness/lexinvariance/) runs under `go test ./...` and fails on any finding from a hard-gate transform (Reindent, TabSpace, InterTokenPadding) across [`testdata/golden/`](testdata/golden/). New findings close by **either** an engine fix (M12.16/17 precedent) **or** an input-boundary router in [`internal/lint/lint.go`](internal/lint/lint.go)'s `classifyFile` (M12.18's `actionShaderDecl` precedent) — **never** a per-file skip list. The playground accepts pasted text without a filename, so a path-based skip would silently lie about coverage on the surface that serves users. BlankLineJitter findings stay triage-only via `make lexinvariance`.
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

Real game-file fixtures under `testdata/golden/` (~7.7 MB committed). Layout:

- `d2/game1/`, `doto/game1/` — files spot-checked by `goldenFileNames` in [internal/scan/scan_test.go](internal/scan/scan_test.go); scan tests fail hard if any are missing.
- Flat root files (`generated.decls.*.decl`, `eof.*.decl`) — broader corpus included alongside the subdirs in `internal/lint/`'s `TestCleanSweep` / `TestCoverageAudit` (which walk `testdata/golden/` recursively) for diagnostic-shape coverage. `*.decl.xml` files sit at the flat root too but are skipped by extension — they're XML, not the `.decl` grammar.

A file graduates from the flat root into `d2/game1/` or `doto/game1/` when it becomes a spot-checked golden — i.e. when we want byte-level scanner output pinned. `testdata/binary/sample.bwm` feeds `TestBinarySweep`.
