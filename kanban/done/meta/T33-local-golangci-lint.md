# T33 · Make golangci-lint runnable locally in the devcontainer

**Status:** done
**Version:** meta
**Size:** small
**Origin:** T32

## What

T30 added a `golangci-lint` job to [.github/workflows/ci.yml](.github/workflows/ci.yml) that gates the `deploy` job (`needs: [test, lint, build]`), but it did not provide any way to run the same linter locally. While implementing T32 (fixing the two `unused` findings the lint job surfaced), the verification step had to fall back to "push to a feature branch and wait for CI" because `golangci-lint` is not installed in the devcontainer (`which golangci-lint` → not found). The ticket's Verification section had to accept this as a known limitation.

This means today's contributor loop for any future lint-only failure is: push → wait ~30s for CI → read the annotation → fix → push again. There's no pre-flight. The fix is to install (or make trivially installable) `golangci-lint` inside the devcontainer so `go vet` and `golangci-lint run` can both be invoked locally before push.

## Scope

- Pick the simplest delivery mechanism and apply it. Options, in rough order of weight:
  - **Add to the devcontainer image.** Append a `RUN` line to `.devcontainer/Dockerfile` (or `postCreateCommand` in `devcontainer.json`) that fetches the same `golangci-lint` version CI uses. Pinned, not `latest`, so the local version matches CI.
  - **Add a `make lint` target.** Have the target `go install` (or `curl | sh`) `golangci-lint` at a pinned version on first run, then exec it. Doesn't require a container rebuild but adds a one-time install delay per fresh checkout.
- Pin the version. CI currently uses `version: latest` in `golangci/golangci-lint-action@v6`; align CI and local on a single explicit version (e.g., `v1.64.8`, which is what the failing CI run reported running) so they cannot drift.
- Document the local invocation in `CLAUDE.md` § Tooling alongside the existing `go test ./...` / `go vet ./...` notes.
- Confirm the local run reproduces the CI behavior on a known-bad commit (e.g., temporarily revert T32's deletions and verify both helpers are flagged locally).

## Dependencies

None. T30 introduced the gap, T32 was the first ticket to feel it; T33 closes the loop independently of either.

## Verification

- `golangci-lint run --timeout=5m` runs successfully inside a freshly built devcontainer with no extra setup beyond what `claude --worktree` / a regular container start already do.
- The local version matches what `ci.yml` invokes (no version drift between local pre-flight and CI).
- `CLAUDE.md` § Tooling lists the local lint command alongside `go test` and `go vet`.
- Reverting T32's two deletions locally and re-running `golangci-lint run` reproduces the same two `unused` findings CI reported on T30's PR.

## Completion

**What shipped:**

- `Makefile` gained a `lint` target with `GOLANGCI_LINT_VERSION ?= v1.64.8` at the top. On first invocation the target self-installs `golangci-lint` to `$(go env GOPATH)/bin` if it isn't on PATH, then runs `golangci-lint run --timeout=5m`. Subsequent runs use the cached binary.
- `.github/workflows/ci.yml`'s `golangci/golangci-lint-action@v6` step pinned from `version: latest` to `version: v1.64.8`, with an inline comment instructing future maintainers to bump it together with the Makefile var so local and CI stay aligned.
- `CLAUDE.md` § Tooling gained a "Lint (T33)" entry alongside the existing test-runner / kanban-move hook notes, documenting `make lint` and the version-bump pairing.
- `internal/parse/parse.go` was untouched (T32 already cleaned the unused helpers; the verification step here just exercised them).

**Verification:**

- Bootstrapped `golangci-lint v1.64.8` into `~/go/bin` and confirmed `golangci-lint run --timeout=5m` exits 0 against the post-T32 tree.
- Cold-start simulation: `rm ~/go/bin/golangci-lint && make lint` — target detected the missing binary, installed v1.64.8 from the official install script, then ran the lint cleanly. Single command, no extra setup, satisfying the ticket's "freshly built devcontainer with no extra setup" criterion.
- Warm-run: `make lint` on the cached binary — exits 0.
- Regression detection: temporarily reintroduced the `(*cursor).sym` helper that T32 deleted, ran `make lint`, got `internal/parse/parse.go:87:18: func (*cursor).sym is unused (unused)` — byte-identical to the CI annotation. Reverted the helper, re-ran, clean.
- `go vet ./...` clean, `go test -count=1 ./...` passes across all 8 packages.

**Decisions / deviations:**

- The plan favored installing `golangci-lint` into `.devcontainer/Dockerfile` (mirroring how Go itself is installed there). I implemented that, then discovered `.devcontainer/*` is fully gitignored ([.gitignore:37](.gitignore#L37)) — so my Dockerfile + devcontainer.json edits would have helped only this maintainer's local container and would have evaporated on any upstream-template re-sync. Reverted those edits and re-routed the install into a `Makefile` self-bootstrap, which has the additional benefit of working for non-devcontainer setups (native Go installs, other CI runners) with no infrastructure changes. The "freshly built devcontainer with no extra setup" verification criterion still holds — `make lint` becomes the no-extra-setup path rather than the image bake-in.
- Version pinned to `v1.64.8` (the version reported in the failing CI run). Future bumps require a coordinated edit to two places (`Makefile`'s `GOLANGCI_LINT_VERSION` and `ci.yml`'s `version:` arg); both files have a comment pointing at the other. Considered factoring the version into a single source-of-truth file (e.g., `.golangci-version`) but that's over-engineered for two callers.
- `CLAUDE.md` was updated under § Tooling, not § Dev setup. Tooling already groups test-runner / kanban-move hooks and pre-approved commands, which is the right semantic neighborhood for a lint pre-flight. Dev setup covers credential / corpus bootstrapping, which is a different concern.
- The locale warning (`bash: warning: setlocale: LC_ALL: cannot change locale (en_US.UTF-8)`) on every Makefile target run is preexisting and not introduced by this ticket.

**Follow-ups:** T34 (Decide on `.devcontainer/` source control strategy) — the deviation above surfaced that gitignored devcontainer config is a structural issue affecting any infra change, not just this one. T34 captures the decision needed.
