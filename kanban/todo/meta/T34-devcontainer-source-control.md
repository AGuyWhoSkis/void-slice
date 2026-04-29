# T34 · Decide on .devcontainer/ source control strategy

**Status:** todo
**Version:** meta
**Size:** medium
**Origin:** T33

## What

`.devcontainer/*` is fully gitignored ([.gitignore:37](.gitignore#L37)), so the ~5 KB of customizations the repo layers onto the upstream Anthropic devcontainer template (firewall script, host bind-mounts, Go and Claude Code installs, pre-baked Claude settings, OAuth port forwarding) lives only in the maintainer's working tree. Other contributors cloning the repo get no devcontainer at all — they have to bootstrap one from upstream and re-apply every customization by hand.

T33 surfaced this concretely. The natural place to install `golangci-lint` for local pre-flight was `.devcontainer/Dockerfile` (mirroring how Go itself is installed there), but those edits were source-control invisible, so T33 had to re-route the install into a `Makefile` self-bootstrap target. That worked for `golangci-lint` because it is a single binary with a published install script, but the same pattern won't generalize cheaply — every future infra change either has to be re-implemented as a Makefile/script, or it will silently apply only to the maintainer's container.

The decision is structural: either track `.devcontainer/` (with a documented strategy for syncing upstream template updates) or deliberately keep moving infra concerns into tracked alternatives. Either is defensible; staying in the current state (gitignored, with growing drift between maintainer and contributor environments) is not.

## Scope

Pick one path and apply it; record the rationale in the ticket completion notes.

- **Option A — track `.devcontainer/` and version a sync strategy.** Remove the `.devcontainer/*` line from `.gitignore`, commit the current state, and document in `CLAUDE.md` § Dev setup: how to take an upstream template update (e.g. "diff against `github.com/anthropics/claude-code/tree/main/.devcontainer`, cherry-pick what's wanted, preserve our `# Changes from upstream` headers"). The headers already in [.devcontainer/Dockerfile:1-14](.devcontainer/Dockerfile#L1-L14) and [.devcontainer/devcontainer.json:1-16](.devcontainer/devcontainer.json#L1-L16) are explicitly a sync log — Option A makes that log load-bearing.
- **Option B — keep `.devcontainer/` gitignored and route infra into tracked alternatives.** Document the rule in `CLAUDE.md` (e.g., "container-level installs go in `Makefile` or `scripts/`, not Dockerfile") and audit the existing customizations: which of them (firewall, Claude settings) genuinely need to be inside the image, vs. which (Go install, golangci-lint) can move to `Makefile`-style self-bootstrap. Migrate what cleanly can move; leave the rest in the gitignored Dockerfile and accept that those bits won't propagate.
- **Option C — split the difference.** Track `.devcontainer/` but only the project-specific customizations (e.g., firewall allowlist additions, repo-relative bind-mounts), not the upstream template body. This requires factoring the Dockerfile into a tracked overlay plus an untracked upstream base, which is more infrastructure than the project probably wants. Listed for completeness.

**Implementation, regardless of choice:**

- If Option A: `git rm` is not needed (the files are untracked, not tracked-and-ignored). Just remove `.gitignore:37`, `git add .devcontainer/`, commit, and update CLAUDE.md.
- If Option B: enumerate the current customizations in CLAUDE.md so a contributor knows what they're missing, and migrate at least one (the Go install is a natural candidate) to prove the pattern. T33's `make lint` self-bootstrap is the template.
- Either way: confirm the firewall allowlist (`init-firewall.sh`) is sane to share publicly — no host-specific IPs or secrets — before committing.

## Dependencies

None directly. T21 (devcontainer auth via host bind-mount) and T33 (lint pre-flight) both layer customizations into the gitignored Dockerfile; this ticket retroactively decides whether those layers should have been tracked.

## Verification

- A fresh `git clone` on a different machine produces a working devcontainer (Option A) or a documented bootstrap path (Option B) — no tribal knowledge required to get a contributor unblocked.
- `CLAUDE.md` § Dev setup explains what gets tracked and how to sync upstream updates (Option A) or which infra concerns belong outside the Dockerfile (Option B).
- No secrets, host-specific paths, or personal credentials are committed if Option A is chosen.
- T33's `make lint` continues to work; if Option A, the Makefile self-install becomes a redundant safety net rather than the primary delivery path, which is fine.
