# T23 · Self-Maintaining Kanban via Status Field

**Status:** done  
**Version:** meta  
**Size:** medium

## What

Add a `status:` field to ticket frontmatter and a hook that automatically moves the ticket file between `todo/` → `in-progress/` → `done/` when the field is updated. Eliminates manual file-moving as part of the ticket workflow.

## Scope

**Frontmatter field:**
- Allowed values: `todo`, `in-progress`, `done`
- Version subfolder is preserved on move (e.g., a `v1` ticket moving to done lands in `done/v1/`; a `meta` ticket lands in `done/meta/`)

**Hook:**
- Type: `PostToolUse`, triggered on `Edit` and `Write` tool calls matching `kanban/**/*.md`
- Parse the `**Status:**` line from the updated file
- Derive source and destination folder from current path and new status value
- Move the file with `git mv` to preserve history
- Hook script lives in `.claude/hooks/`; registered in `.claude/settings.json`

**Note:** Overlaps T18 and T19 — all three modify `.claude/settings.json`. Implement in sequence: T19 first, T18 second, T23 last.

**T17 alignment:** After T23 is complete, update T17's `/crud-ticket` command so it only edits the `**Status:**` field — the hook handles the `git mv`. Remove any manual file-move instructions from that command's prompt text. Together, T17 and T23 eliminate every manual ticket-moving step.

## Dependencies

None (T18 hook infrastructure is additive, not a prerequisite)

## Verification

```bash
# Edit a ticket's **Status:** line from "todo" to "in-progress"
# Confirm the file moves from kanban/todo/<version>/ to kanban/in-progress/<version>/
# without any manual mv command
```

## Completion

Created `.claude/hooks/kanban-move.sh`. Registered in `.claude/settings.json` as a `PostToolUse` hook on `Edit|Write` (same entry as the test-runner hook). Script extracts `file_path`, resolves relative paths, matches the `kanban/<col>/<version>/*.md` pattern, reads `**Status:**` from the file, derives current column from the path, and runs `git mv` if column ≠ status. Exits 0 on all no-op paths. Takes effect in the next Claude Code session. T17 `/crud-ticket` command aligned to only edit `**Status:**` from the start.
