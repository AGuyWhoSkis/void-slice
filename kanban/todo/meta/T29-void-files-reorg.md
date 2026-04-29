# T29 · Reorganize void-files corpus

**Status:** todo
**Version:** meta
**Size:** small

## What

`void-files/` currently contains a mix of structured and ad-hoc folders: `d2/`, `doto/`, plus `big-Export/`, `trimmed-Export/`, and `trimmed-Export-2/`. The ad-hoc exports were one-off scratch dumps from earlier experiments and are no longer referenced by any test or tool. Consolidate the corpus around the two canonical golden sets so the layout is self-documenting and downstream work (T30 corpus hosting) has a stable surface to package.

After this ticket, `void-files/` contains exactly two top-level trees:

- `d2/` — **set 1** of golden files (known-valid Dishonored 2 exports).
- `doto/` — **set 2** of golden files (known-valid Death of the Outsider exports).

## Scope

- Delete `void-files/big-Export/`, `void-files/trimmed-Export/`, `void-files/trimmed-Export-2/`. These folders are not referenced anywhere in the codebase (`grep -rn "big-Export\|trimmed-Export"` returns no hits in Go, YAML, TOML, or JSON).
- Confirm `void-files/d2/` and `void-files/doto/` are the only remaining top-level entries. Each preserves its existing `game0/`, `game1/`, `game2/`, `game3/` subtree structure — those are referenced by `internal/scan/scan_test.go`, `internal/parse/parse_test.go`, and the lint sweeps in `internal/lint/`.
- Update any documentation that mentions the ad-hoc folders. Quick audit pass over `CLAUDE.md`, `README.md`, and any `*.md` under `kanban/done/` that may explain the corpus layout (do not rewrite completion notes; only add a forward-pointer if needed).
- Add a short `void-files/README.md` (one paragraph) describing the two-set layout: `d2/` = set 1 known-valid, `doto/` = set 2 known-valid. This is the only file in `void-files/` that should be tracked by git — everything else stays gitignored.

## Dependencies

None. Should land before T30 so the CI/corpus-hosting work has a stable layout to package.

## Verification

- `ls void-files/` shows only `d2/` and `doto/` (plus the new `README.md`).
- `grep -rn "big-Export\|trimmed-Export" .` returns no matches outside `kanban/done/`.
- `go test ./...` still passes locally (the existing scan/parse/lint tests only reference `d2/game1/` and `doto/game1/`, both of which are unchanged).
