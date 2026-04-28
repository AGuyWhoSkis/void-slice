# T5 · CLI (`cmd/voidslice`)

**Status:** todo  
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
./voidslice lint void-files/binwalk\ test/boat_curator_p.bwm # exit 1, binary error
go test ./cmd/voidslice/...
```
