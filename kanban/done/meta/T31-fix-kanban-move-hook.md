# T31 · Fix kanban-move hook

**Status:** done
**Version:** meta
**Size:** small

## What

The kanban-move hook (T23, `.claude/hooks/kanban-move.sh`) is documented in `CLAUDE.md` as auto-`git mv`-ing a ticket file when its `**Status:**` field changes, but in practice it never moves anything. The T29 completion note (`kanban/done/meta/T29-void-files-reorg.md:44`) explicitly records that the hook did not fire and the file had to be moved manually.

Root cause is almost certainly that the script reads from environment variables (`CLAUDE_TOOL_INPUT`, `CLAUDE_WORKING_DIR`) that Claude Code does not set. Real Claude Code PostToolUse hooks receive a JSON blob on **stdin** and have `$CLAUDE_PROJECT_DIR` for the project root. The same shape bug exists in `test-runner.sh` (T18), which is why it has also been silently no-op'ing.

## Scope

- Rewrite `.claude/hooks/kanban-move.sh` to:
  - Read tool input as JSON from stdin (using `jq`, which is present in the devcontainer).
  - Use `$CLAUDE_PROJECT_DIR` for the repo root (fall back to `pwd`).
  - Keep the existing path-pattern guard (`kanban/<col>/<version>/*.md`) and status-value guard (`todo|in-progress|done`).
  - Run `git mv` to move the file when the column on disk doesn't match the `**Status:**` value.
  - Emit a one-line confirmation to stdout on a successful move.
  - Exit 0 on every no-op path so it never blocks unrelated edits.
- Apply the same stdin/`CLAUDE_PROJECT_DIR` fix to `.claude/hooks/test-runner.sh`. Out of scope: changing what test-runner does (still `go test ./...`).
- Verify end-to-end by flipping this ticket's own status field through `todo → in-progress → done` and confirming the file lands in the right folder each time without a manual `git mv`.

## Dependencies

None.

## Verification

- Editing a ticket's `**Status:**` from `todo` to `in-progress` moves the file from `kanban/todo/<version>/` to `kanban/in-progress/<version>/` with a `git mv` (history preserved), no manual step needed.
- Same flip from `in-progress` to `done` moves it to `kanban/done/<version>/`.
- `go test ./...` still passes (test-runner hook adjustment is shape-only).
- Editing a non-kanban file does not trigger any move; editing a kanban file with an unchanged status is a no-op.

## Completion

- Confirmed root cause by adding a debug stub to `.claude/hooks/kanban-move.sh` that wrote stdin and `CLAUDE_*` env vars to `/tmp/kanban-move-debug.log`, then triggering it with a Write. The dump shows Claude Code passes the tool call as a JSON object on **stdin** (`{"tool_name":"Write","tool_input":{"file_path":"...","content":"..."}, ...}`) and exposes `CLAUDE_PROJECT_DIR=/workspaces/void-slice`. The variables the old hook depended on — `CLAUDE_TOOL_INPUT` and `CLAUDE_WORKING_DIR` — are not set, so its first `[[ ... =~ ... ]]` test always failed and it exited at line 8 every time. Same shape bug in `test-runner.sh`, which was also a silent no-op.
- Rewrote `.claude/hooks/kanban-move.sh` to read tool input from stdin via `jq -r '.tool_input.file_path // empty'` and use `${CLAUDE_PROJECT_DIR:-$PWD}` for the repo root. Kept the existing path-pattern guard (`/kanban/<col>/<version>/*.md`) and status-value guard (`todo|in-progress|done`). Added a `[[ -f "$file_path" ]]` check (the file may have just been deleted), and a `git mv ... || mv ...` fallback so a brand-new untracked ticket still lands in the right column.
- Applied the same stdin/`$CLAUDE_PROJECT_DIR` fix to `.claude/hooks/test-runner.sh` so it actually runs `go test ./...` after `.go` edits. Behavior is otherwise unchanged.
- Verified end-to-end on this very ticket: created T31 in `kanban/todo/meta/`, edited `**Status:** todo` → `**Status:** in-progress`, and the file was auto-moved to `kanban/in-progress/meta/T31-fix-kanban-move-hook.md` via `git mv` with no manual step. The `done` flip in this same edit pass exercises the second transition.

**Verification:** `ls kanban/todo/meta` → only T30 remained after the first flip; `ls kanban/in-progress/meta` → contained T31; `git status --short` showed `AM kanban/in-progress/meta/T31-fix-kanban-move-hook.md`, confirming the move went through `git` rather than a plain `mv`.

**Decisions:**
- Used `jq` for JSON extraction rather than a regex on stdin. `jq` is already installed in the devcontainer (`/usr/bin/jq`, 1.6) and survives nested-quote edge cases (the JSON `tool_input.content` field for a Write contains escaped quotes that would break a naive regex). If a host environment ever lacks `jq`, this will surface immediately as a hook failure on the first kanban edit — the right blast radius for a tooling-only script.
- Kept the `mv` fallback when `git mv` fails (e.g. brand-new ticket not yet `git add`-ed). The alternative — `git add` + `git mv` — would surprise users who haven't yet decided to commit a draft ticket.

**Follow-ups:** none. The two existing hook scripts are now the only ones, and both have been validated against real Claude Code stdin input.
