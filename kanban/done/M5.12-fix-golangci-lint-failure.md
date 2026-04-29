# T32 · Fix golangci-lint CI failure (unused parser helpers)

**Status:** done
**Version:** meta
**Size:** small

## What

The `golangci-lint` job in [.github/workflows/ci.yml](.github/workflows/ci.yml) fails on every push with two `unused` findings in [internal/parse/parse.go](internal/parse/parse.go):

- `internal/parse/parse.go:87:18: func (*cursor).sym is unused (unused)`
- `internal/parse/parse.go:168:18: func (*cursor).expectIdent is unused (unused)`

`grep -rn '\.sym\|\.expectIdent' internal/` confirms neither helper is called anywhere in the tree — they are dead code left behind from earlier parser iterations (the `sym` accessor was superseded by callers reading `src[t.Span.Start]` directly per the Option A punctuation convention; `expectIdent` was an API shape that no caller adopted). `go test ./...` and `go vet ./...` both pass — only `golangci-lint`'s `unused` linter catches them, and it has been red on `main` since T30 since `go vet` was the only static check before.

The other CI jobs (`test`, `build`) are green; the failure is isolated to the lint job, but it gates the `deploy` job (`needs: [test, lint, build]`) so `main` no longer auto-deploys.

## Scope

- Delete `(*cursor).sym` from [internal/parse/parse.go:86-89](internal/parse/parse.go#L86-L89). Callers already read `c.src[tok.Span.Start]` directly per the Option A convention documented in `CLAUDE.md`; no replacement needed.
- Delete `(*cursor).expectIdent` from [internal/parse/parse.go:167-174](internal/parse/parse.go#L167-L174). No caller exists; if a future ticket needs the "match-or-diagnose ident" pattern it can be reintroduced at that point.
- Verify `golangci-lint run --timeout=5m` is clean locally (or in CI on the feature branch) after the deletions. If other `unused` findings surface, fix them in the same ticket — the goal is a green lint job, not just these two specific lines.
- Do not silence the linter (no `//nolint:unused` directives, no config carve-outs). Dead code should be deleted, not annotated.

## Dependencies

None. T30 introduced the lint job; this is the first cleanup pass against it.

## Verification

- `go test ./...` passes (no test references the deleted helpers, but confirm).
- `go vet ./...` passes.
- `golangci-lint run --timeout=5m` exits 0 with no findings.
- A push to a feature branch shows all four CI jobs green (`test`, `lint`, `build`, plus `deploy` skipped on non-main as expected).

## Completion

**What shipped:**

- Deleted `(*cursor).sym` from [internal/parse/parse.go](internal/parse/parse.go) (the four-line helper plus its doc comment). Callers already access `c.src[tok.Span.Start]` directly per the Option A punctuation convention; no replacement was needed.
- Deleted `(*cursor).expectIdent` from the same file (eight-line helper plus doc comment). No caller existed; the sibling `expectKind` and `expectSym` cover the actively-used "match-or-diagnose" patterns.
- No `//nolint` directives, no `.golangci.yml` carve-outs, no test changes. The fix is two excisions and nothing else.

**Verification:**

- `go vet ./...` — clean.
- `go test -count=1 ./...` — all 8 packages pass on a fresh run, including `internal/parse` and the integration test in `parse_test.go` that exercises the corpus golden.
- `grep -rn '\.sym(\|\.expectIdent('` across the tree — no matches; nothing referenced the deleted helpers.

**Decisions / deviations:**

- `golangci-lint run --timeout=5m` could not be executed locally — the binary is not installed in the devcontainer (`which golangci-lint` returns not found). Verification of the linter-clean state therefore depends on the next CI run on this branch, which the ticket's Verification section explicitly accepted ("locally _or in CI on the feature branch_"). T33 was filed to close that local-pre-flight gap.

**Follow-ups:** T33 (Make golangci-lint runnable locally in the devcontainer).
