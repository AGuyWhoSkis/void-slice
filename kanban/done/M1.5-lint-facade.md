# T4 · Lint Facade (`internal/lint`)

**Status:** done  
**Version:** v1  
**Size:** medium  
**Blocks:** T5

## What

Create `internal/lint/lint.go` as the single public engine API. All transport layers (CLI, and later HTTP server, LSP) import only this package. Includes binary detection and file-type classification.

## Scope

**Public types:**
```go
type Severity int
const (
    Error   Severity = iota
    Warning
)

type Diagnostic struct {
    Severity Severity
    Code     string        // e.g. "PARSE_UNTERMINATED_OBJECT"
    Span     scan.Span     // or promote Line/Col fields directly — decide here
    Message  string
}

type Linter interface {
    Lint(filename string, src []byte) ([]Diagnostic, error)
}

func New() Linter
```

**File type classification (run before scan/parse):**
- Check extension: `.decl`, `.entitydef` → proceed normally
- `.entities`, `.cfg` → prepend a `Warning` diagnostic: `"editing via Void Explorer is inconsistent — results may not reflect in-game"`; then proceed with lint
- `.tome`, `.bwm`, `.navmesh`, `.mapresources`, `.soundpropa`, `.bnavmesh`, `.bphysworld`, `.maprscreusechunk0` → return immediately with single `Error` diagnostic: `"binary map file — cannot lint"`
- Any other input → run binary sniff (see below); if binary, return error

**Binary sniff:**
- Read first 512 bytes; if any null byte (`0x00`) is present, treat as binary
- A small helper `isBinary(data []byte) bool` in `lint.go`; test it independently

**Wiring:**
```
Lint(filename, src):
  1. classifyFile(filename) → decide action or prepend warning
  2. isBinary(src[:min(512, len(src))]) → error if true
  3. scanDiags, toks := scan.Scan(src)
  4. validateDiags := validate.ValidateEntities(src, toks)
  5. allDiags := scanDiags + validateDiags → convert to lint.Diagnostic with severity
  6. return sorted by Span.Start
```

**Severity mapping from scan diagnostic codes:**
- `VALIDATE_*` codes → `Warning`
- Everything else → `Error`

**Tests:**
- Clean `.decl` → `[]Diagnostic{}`
- Binary fixture (small committed file under `testdata/binary/`, e.g. a file with a null byte) → single binary error
- `.entities` input → first diagnostic is the VE-inconsistency warning
- Known-broken `.decl` → diagnostics match expected codes

## Dependencies

T1, T2, T3, T15 (all must be complete)

T15 is required because `Lexeme`, `Sym`, `EqIdent`, and `ParseIntLiteral` are unimplemented stubs in `internal/scan` until T15 stage 4e. T4's wiring calls these utilities.

## Verification

```
go test ./internal/lint/...
go test ./...    # full suite, no regressions
```

## Completion

Implemented `internal/lint/lint.go` with the full public API (`Severity`, `Diagnostic`, `Linter`, `New()`), `classifyFile`, `isBinary`, and `Lint()` wiring through `scan.Scan` + `validate.ValidateEntities`. Tests in `internal/lint/lint_test.go` cover clean `.decl`, binary fixture, binary sniff on unknown extension, `.entities` VE-inconsistency warning, and known-broken `.decl`. Binary fixture committed at `testdata/binary/sample.bwm`.

Devcontainer setup changes in the same session: Go 1.23.5 added to `.devcontainer/Dockerfile` (installed as root to `/usr/local/go`), module cache pre-populated via `go mod download` during Docker build, and `proxy.golang.org`/`sum.golang.org` added to `init-firewall.sh` allowlist.

`go test ./...` and `go vet ./...` both pass clean.
