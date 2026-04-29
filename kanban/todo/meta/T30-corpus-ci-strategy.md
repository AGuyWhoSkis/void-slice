# T30 · Decouple CI from the void-files corpus

**Status:** todo
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
