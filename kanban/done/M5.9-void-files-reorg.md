# T29 · Reorganize void-files corpus

**Status:** done
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

## Completion

- Deleted `void-files/big-Export/`, `void-files/trimmed-Export/`, `void-files/trimmed-Export-2/`. `void-files/` now contains exactly `d2/`, `doto/`, and the new `README.md`.
- Added `void-files/README.md` (one paragraph) describing the two-set layout (`d2/` = set 1 D2, `doto/` = set 2 DOTO), the read-only nature of the corpus, the `game0/`–`game3/` subtree structure consumed by `internal/scan`, `internal/parse`, and `internal/lint`, and the gitignore exception for the README itself.
- Updated `.gitignore` to keep `void-files/*` ignored while allowing `void-files/README.md` to be tracked (`!void-files/README.md`). This was the minimum gitignore change needed to make the README the only tracked file under the corpus root, as scoped.
- Doc audit: `CLAUDE.md` and `README.md` did not mention the deleted folders by name, so no edits were needed there. The only remaining grep hits for `big-Export`/`trimmed-Export` outside of this ticket file are in `kanban/done/v1/T15` and `kanban/done/v2/T25`, which are completion notes and were left untouched per scope.

**Verification:** `ls void-files/` → `README.md d2 doto`; `grep -rn 'big-Export\|trimmed-Export' .` → no matches outside `kanban/done/`; `go test ./...` → all packages pass; `go vet ./...` → clean.

**Decisions:**
- Chose `!void-files/README.md` over moving the README outside the corpus tree (e.g. a top-level `void-files.md`). The ticket explicitly asked for `void-files/README.md`, and the gitignore exception is the standard idiom for this case.
- The kanban-move hook did not auto-move this file when its `**Status:**` field was edited (likely because the hook activates in a new session per `CLAUDE.md`). Used `git mv` manually to land the file in `kanban/in-progress/meta/` for the work, and the close edit will move it to `kanban/done/meta/` via the same path. Not a scope deviation, just a session-level note.

**Follow-ups:** none
