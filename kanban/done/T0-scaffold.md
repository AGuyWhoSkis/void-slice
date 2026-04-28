# T0 · Project Scaffold

**Status:** done  
**Size:** small  
**Blocks:** T5

## What

Rename `cmd/void-slice` → `cmd/voidslice` and verify the Go module is ready to add `internal/lint`.

## Scope

- Rename directory `cmd/void-slice/` → `cmd/voidslice/`; update any import paths or build references
- Confirm `go.mod` module name (`void-slice`) — note whether it needs updating if the binary name changes
- Create stub `internal/lint/` directory with an empty `lint.go` (`package lint`) so T4 has a landing zone
- `go build ./...` passes cleanly after changes

## Dependencies

None — do this first.

## Verification

```
go build ./...   # no errors
go test ./...    # no regressions in scan tests
```
