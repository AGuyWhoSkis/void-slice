# T1 · Complete Parser

**Status:** done  
**Version:** v1  
**Size:** large  
**Blocks:** T2, T4

## What

Implement `WalkEntities()` and the cursor helper methods in `internal/parse/parse.go`. The Handler interface, data types, and grammar pseudocode are already documented in the file.

## Scope

**Cursor fixes (do first — existing stubs have bugs):**
- `cursor.i` must be initialized to `-1` (zero value skips `toks[0]`); verify or fix initialization
- `cursor.n` is used as token count in `eof()` but as source-byte length in `lexeme()` — they are different things; separate into `nToks int` and use `len(c.src)` in `lexeme`
- `cursor.isIdent()` bounds check is inverted (comparison happens only when out-of-bounds); fix logic
- Implement the declared-but-empty cursor methods: `matchKind`, `expectKind`, `matchSym`, `expectSym`, `matchIdent`, `expectIdent`, `syncTo`

**Walk implementation:**
- Implement `walkEntities(h Handler)` private method and wire it from the public `WalkEntities()`
- Handle: `Version <number>`, `component { <TypeIdent> <NameIdent> { <body> } }`, assignment (`Key = Value ;`), nested objects, typed block headers (`Ident Ident { ... }`)
- Emit `OnVersion`, `OnComponentBegin/End`, `OnComponentDecl`, `OnObjectBegin/End`, `OnAssignment`, `OnDiag` at the documented points
- Emit diagnostics for unterminated objects, unexpected tokens, missing semicolons; recover via `syncTo`

**Naming note:** `WalkEntities` is a legacy name from when `.entities` was primary. Rename to `Walk` or `WalkDecl` if it makes sense once the file type story is clear — but don't block on it.

**New diagnostic codes (add to `scan_constants.go` or a new `parse_constants.go`):**
- `PARSE_UNEXPECTED_TOKEN`, `PARSE_EXPECTED_SYMBOL`, `PARSE_EXPECTED_IDENTIFIER`, `PARSE_EXPECTED_SEMICOLON`, `PARSE_UNTERMINATED_OBJECT`

**Tests (`internal/parse/parse_test.go`):**
- One test per Handler callback type (Version, ComponentBegin/End, ComponentDecl, ObjectBegin/End, Assignment)
- Error path: unterminated object, missing semicolon, unexpected token
- Integration: run against existing clean golden `.decl` files; expect zero diagnostics

## Dependencies

T0 (module must be clean first, though T1 doesn't touch `internal/lint`)

## Verification

```
go test ./internal/parse/...   # all tests pass
go test ./internal/scan/...    # no regressions
```
