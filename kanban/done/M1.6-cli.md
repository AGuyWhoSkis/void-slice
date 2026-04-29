# T5 · CLI (`cmd/voidslice`)

**Status:** done  
**Version:** v1  
**Size:** medium

## What

Replace the `cmd/void-slice` stub with a real `voidslice lint <file>` command wired to `internal/lint` and `internal/report`.

## Scope

**Commands / flags:**
```
voidslice lint <file> [--json]
```
- Reads file from disk, calls `lint.New().Lint(filename, src)`
- Default output: human-pretty via `report.Render`
- `--json`: machine-readable via `report.RenderJSON`
- Exit code `1` if any `Error`-severity diagnostics; `0` if clean or warnings-only
- Write to stdout; any internal errors (file not found, etc.) to stderr

**Source cleanup:**
- Remove or archive `cmd/void-slice/` after `cmd/voidslice/` is working
- Confirm binary name produced by `go build` is `voidslice`

**Tests (`cmd/voidslice/`):**
- Golden-file tests: pipe a broken `.decl` fixture through the binary; compare stdout to expected output file
- Test both `--json` and default format
- Test exit codes (use `exec.Command`)

## Dependencies

T0, T4 (lint facade), T3 (report)

## Verification

```
go build -o voidslice ./cmd/voidslice
./voidslice lint void-files/d2/game1/some-clean.decl      # exit 0, no diagnostics
./voidslice lint testdata/broken/count-mismatch.decl       # exit 1, human-pretty diagnostics
./voidslice lint --json testdata/broken/count-mismatch.decl  # exit 1, JSON output
./voidslice lint testdata/binary/sample.bwm                  # exit 1, binary error
go test ./cmd/voidslice/...
```

## Completion

Replaced the `cmd/voidslice/main.go` stub with a real `voidslice lint <file> [--json]` CLI. The binary reads a file from disk, calls `lint.New().Lint`, converts `[]lint.Diagnostic` to `[]scan.Diagnostic` for rendering, outputs via `report.Render` (default) or `report.RenderJSON` (`--json`), and exits 1 on any Error-severity diagnostic.

Added `cmd/voidslice/main_test.go` with 6 tests using `exec.Command` + `TestMain`-built binary: exit code 0/1, human-format golden, JSON golden, binary file, and file-not-found. Golden files generated under `cmd/voidslice/testdata/golden/`.

Key decision: `count-mismatch.decl` only emits `VALIDATE_*` warnings (exit 0), so `TestLint_ExitCode_Error` uses `missing-semicolon.decl` which produces a `PARSE_EXPECTED_SEMICOLON` error (exit 1). All `go test ./...` and `go vet ./...` pass.
