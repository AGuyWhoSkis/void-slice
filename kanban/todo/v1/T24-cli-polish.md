# T24 · CLI polish (`cmd/voidslice`)

**Status:** todo
**Version:** v1
**Size:** small

## What

Three small follow-up gaps left by T5: ANSI color output (stubbed but unimplemented), multi-file argument support, and a `--strict` flag. Each sub-task is independent and can be done in order.

## Scope

**Sub-task A — ANSI color output:**
- Implement `RenderOptions.UseColor` in `internal/report/report.go`: wrap severity label and caret in ANSI escape codes when the flag is set (reset at end of each diagnostic)
- Add `--color` / `--no-color` flags to `voidslice lint`; default to auto-detect TTY (`os.Stdout.Fd()` + `term.IsTerminal` or equivalent without new deps — check `TERM` env or use a syscall)
- Update or add golden tests for the color variant; assert `--no-color` matches existing golden output

**Sub-task B — Multi-file linting:**
- Accept one or more file arguments: `voidslice lint <file> [<file>...]`
- Print diagnostics for each file in argument order; files with zero diagnostics produce no output
- Print a summary line to stdout: `N file(s): X error(s), Y warning(s)` (always printed when N > 1)
- Exit 1 if any file has an Error-severity diagnostic across all files; exit 0 otherwise
- Tests: run binary with two fixtures, assert combined output and exit code

**Sub-task C — `--strict` flag:**
- Add `--strict` to the `lint` subcommand; when set, treat Warning-severity diagnostics as errors for exit-code purposes only (rendering is unchanged)
- Tests: assert `count-mismatch.decl` exits 0 without `--strict` and exits 1 with `--strict`

## Dependencies

T5 (CLI exists)

## Verification

```bash
go build -o /tmp/voidslice ./cmd/voidslice

# A: color
/tmp/voidslice lint --color testdata/broken/count-mismatch.decl   # ANSI escapes visible
/tmp/voidslice lint --no-color testdata/broken/count-mismatch.decl  # matches existing golden

# B: multi-file
/tmp/voidslice lint testdata/broken/count-mismatch.decl testdata/broken/missing-semicolon.decl
# → diagnostics from both files, summary line, exit 1

# C: strict
/tmp/voidslice lint testdata/broken/count-mismatch.decl; echo $?          # 0
/tmp/voidslice lint --strict testdata/broken/count-mismatch.decl; echo $?  # 1

go test ./cmd/voidslice/... ./internal/report/...
go vet ./...
```
