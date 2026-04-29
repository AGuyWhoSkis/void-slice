# T30 · Decouple CI from the void-files corpus

**Status:** done
**Version:** meta
**Size:** medium

## What

Two related problems with the current setup:

1. **CI runs tests that require gitignored files.** `void-files/` is `.gitignore`d (correctly — see problem 2), but `internal/scan/scan_test.go`'s `TestMain` (lines 32–48) hard-fails with `os.Exit(1)` when the goldens listed in `goldenFileNames` aren't readable. On GitHub Actions the corpus is absent, so `go test ./...` cannot succeed. The `internal/lint/` sweeps and `internal/parse/parse_test.go:397` already handle this correctly via `t.Skip` when `void-files/` is missing — the scan tests are the outlier.
2. **The corpus is enormous and doesn't belong on GitHub.** `void-files/` is ~2.2 GB of extracted game assets. It's already gitignored (good), but there's currently no documented or automated way for CI — or a fresh contributor — to obtain it, which is why CI works around the issue by having tests fail or skip silently.

Solve both at once: pick a hosting strategy for the corpus and wire CI (and local dev) to fetch it on demand, so the full test suite — including the scan goldens — runs green in CI without checking the corpus into git.

## Scope

**Decide and document the corpus hosting strategy.** Pick one and record the rationale in the ticket completion notes:

- **Option A — release artifact / external object store.** Upload a tarball (`void-files-corpus-vN.tar.zst` or similar) to a GitHub Release on a dedicated `corpus` repo, Cloudflare R2, or S3. CI downloads + extracts it before `go test`. Local dev runs a `make corpus` / `scripts/fetch-corpus.sh` helper.
- **Option B — git LFS in a sibling repo.** A `void-files-corpus` companion repo using Git LFS. CI checks it out as a submodule or via a separate clone step. More git-native; LFS bandwidth quotas may bite at 2.2 GB.
- **Option C — minimal CI corpus, full corpus local-only.** Curate a few-MB subset containing just the files referenced by `goldenFileNames` (six files, per `scan_test.go:23`), commit that subset to the main repo under `testdata/corpus-mini/`, and keep the full 2.2 GB corpus as a local-only resource for the broader sweeps (`TestCleanSweep`, `TestBinarySweep`, `TestCoverageAudit`). The sweeps continue to `t.Skip` in CI; the scan goldens get a stable, version-controlled fixture.

Option C is the least-friction path if the goal is just unblocking CI; Options A/B are the right call if the broader sweeps need to run in CI too. Pick based on what the project actually needs CI to verify.

**Implementation, regardless of choice:**

- Fix `internal/scan/scan_test.go` so it does not `os.Exit(1)` when goldens are missing. If Option C is chosen, the goldens come from `testdata/corpus-mini/` and are always present, so the hard-fail becomes correct again. If Option A or B is chosen, the test should `t.Skip` (consistent with the rest of the suite) when the path resolves to a missing file, with a clear log message pointing at the fetch script.
- Update `.github/workflows/ci.yml` to either fetch the corpus (Options A/B) or simply rely on the in-repo mini fixture (Option C). The `test` job must end with `go test ./...` exiting 0.
- Add a `scripts/fetch-corpus.sh` (or equivalent `Makefile` target) for local dev, with a one-line invocation documented in `CLAUDE.md` under § Dev setup.
- Confirm `void-files/` stays in `.gitignore`. The mini fixture under `testdata/corpus-mini/` (Option C) is the only corpus content that should be tracked.

## Dependencies

T29 (corpus reorg) — should land first so the hosted/curated artifact reflects the cleaned-up `d2/` + `doto/` layout, not the legacy ad-hoc folders.

## Verification

- A fresh checkout on a machine without `void-files/` runs `go test ./...` and gets a green result — no `os.Exit(1)`, no missed packages.
- A push to a feature branch on GitHub Actions: the `test` job in `ci.yml` runs `go test ./...` and passes, exercising the scan goldens (not just skipping them).
- `internal/scan/scan_test.go` either reads goldens from a path that's reliably present, or skips cleanly with an actionable message.
- `CLAUDE.md` documents how a contributor fetches the corpus locally.
- `void-files/` (the full 2.2 GB) remains gitignored; nothing of meaningful size is added to the main repo's git history.

## Completion

**Strategy chosen: Option C (minimal in-repo corpus, full corpus local-only).** Rationale: the project already had `void-files/` correctly gitignored as private game data with no public hosting source, and the only test that hard-failed on its absence was the scan goldens. Hosting the full 2.2 GB corpus (Options A/B) would either require redistributing copyrighted Bethesda/Arkane content or pointing CI at a private endpoint that no fresh contributor could reach — neither defensible. Option C is also what the ticket flagged as the least-friction path, and matches what the codebase actually needs CI to verify (scan goldens green, sweeps may skip).

**What shipped:**

- Six golden files referenced by `internal/scan/scan_test.go:goldenFileNames` were copied from `void-files/` into `testdata/corpus-mini/`, preserving the `d2/game1/...` and `doto/game1/...` subpaths. Total size ~7.7 MB; the dominant contributor is the 7.4 MB `maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities` fixture, which the spot-check at byte offset 25980 in `TestScannerGoldenFiles` requires.
- `internal/scan/scan_test.go:20` `testDir` now points at `../../testdata/corpus-mini/`. The `os.Exit(1)` hard-fail in `TestMain` is now correct: the goldens are committed to the repo, so a missing file is a genuine misconfiguration rather than a CI environment issue.
- `internal/parse/parse_test.go:398` integration test (`TestIntegration_EntitiesGoldenFile`) now reads from `testdata/corpus-mini/` as well, so it exercises the entities golden in CI instead of skipping. The `t.Skipf` fallback was kept as a harmless safety net.
- `Makefile` gained a `corpus-mini` target that refreshes the in-repo fixture from a local `void-files/` tree. The file list is encoded as a `CORPUS_FILES` variable so adding a new golden is a one-line edit on top of an edit to `goldenFileNames`. The target is local-only — CI does not invoke it.
- `CLAUDE.md` § Dev setup documents the two corpus tiers (mini = committed, full = gitignored, no fetch script) and updates the project-layout table to reflect that `void-files/` is gitignored and `testdata/corpus-mini/` is the committed counterpart.
- `.github/workflows/ci.yml` was deliberately left untouched — once the goldens are in-repo, the existing `go test ./...` step exercises them. Adding a fetch step would be dead weight.

**Verification:**

- `go vet ./...` — clean.
- `go test ./...` (corpus present) — all packages pass, including `TestScannerGoldenFiles` reading from the mini corpus and `TestIntegration_EntitiesGoldenFile` no longer skipping.
- Fresh-checkout simulation: `mv void-files void-files.hidden && go test -count=1 ./...` — all packages pass, with the lint sweeps (`internal/lint/`) skipping cleanly as designed.
- `.gitignore` still has `void-files/*` + `!void-files/README.md`; no path under `void-files/` is staged.

**Decisions / deviations:**

- No `scripts/fetch-corpus.sh` was added. The ticket scope said "or equivalent `Makefile` target"; `make corpus-mini` is that equivalent for the mini fixture, and a fetch script for the full corpus has no public source to fetch from. CLAUDE.md explicitly tells contributors that the full corpus is BYO (extracted from their own game install).
- The lint sweeps (`TestCleanSweep`, `TestBinarySweep`, `TestCoverageAudit`) continue to `t.Skip` in CI when `void-files/` is absent. The ticket explicitly accepted this for Option C ("the sweeps continue to `t.Skip` in CI; the scan goldens get a stable, version-controlled fixture").
- Committing a 7.4 MB text fixture to git is a one-time cost; `.git` was 3.6 MB pre-T30. Considered truncating the entities file to a smaller representative slice but rejected — the existing spot-check at offset 25980 and the "syntactically valid file produces zero diagnostics" assertion both depend on the file being intact, and the test's stated purpose is to exercise large real-world files.

**Follow-ups:** none
