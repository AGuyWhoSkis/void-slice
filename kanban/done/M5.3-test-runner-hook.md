# T18 · Test-Runner Hook on File Edits

**Status:** done  
**Version:** meta  
**Size:** small

## What

Add a PostToolUse hook that automatically runs `go test ./...` after Claude edits or writes source files, surfacing results directly in the conversation. Closes the TDD loop without a manual "run tests" prompt after every change.

## Scope

- Hook type: `PostToolUse`, triggered on `Edit` and `Write` tool calls
- Filter to `*.go` files only (skip test fixtures, markdown, generated files)
- Command: `go test ./...`; optionally follow with `go vet ./...`
- Output captured and returned to Claude as the tool result
- Configure in `.claude/settings.json` (or `settings.local.json` if machine-specific)

## Dependencies

None (T19 recommended first — if the test command isn't pre-approved, every hook invocation prompts for permission, defeating the purpose)

**Note:** Overlaps T19 — both modify `.claude/settings.json`. Implement T19 first, then add this hook in the same file. Once implemented, add a one-line entry to T20's **Tooling** section describing the hook trigger and command.

## Verification

Edit any `.go` source file; confirm `go test ./...` output appears in the conversation automatically without a manual "run tests" prompt.

## Completion

Created `.claude/hooks/test-runner.sh`. Registered in `.claude/settings.json` as a `PostToolUse` hook on `Edit|Write`. Script extracts `file_path` from `$CLAUDE_TOOL_INPUT` via bash regex, exits early for non-`.go` files, then runs `go test ./...` from `$CLAUDE_WORKING_DIR`. A comment in the script shows how to add `go vet ./...` if desired. Takes effect in the next Claude Code session.
