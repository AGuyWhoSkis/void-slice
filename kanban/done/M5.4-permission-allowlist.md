# T19 · Permission Allow-List for Safe Operations

**Status:** done  
**Version:** meta  
**Size:** small

## What

Pre-approve commonly-used read-only and test-runner commands in Claude Code settings so they run without an approval prompt. Destructive operations (file deletion, force-push, etc.) stay gated.

## Scope

Add `allow` entries to `.claude/settings.json` for:

- `go test ./...` and `go test <pkg>`
- `go build ./...`
- `go vet ./...`
- `grep`, `find`, `ls` (read-only shell ops)

Keep approval gates on:

- `rm`, `git reset --hard`, `git push --force`, `git branch -D`
- Any command writing outside the repo directory

## Dependencies

None

**Note:** Overlaps T18 — both modify `.claude/settings.json`. Do T19 first, then layer the test-runner hook (T18) into the same file. Once implemented, add the pre-approved command list to T20's **Tooling** section so every session starts with that context.

## Verification

Open a new Claude Code session; invoke `go test ./...` via the Bash tool; confirm no approval prompt appears.

## Completion

Created `.claude/settings.json` with a `permissions.allow` list covering `go test`, `go build`, `go vet`, `grep`, `find`, and `ls`. Destructive commands (`rm`, force-push, hard-reset, branch-D) have no allow entry and remain gated. T18 and T23 hooks registered in the same file.
