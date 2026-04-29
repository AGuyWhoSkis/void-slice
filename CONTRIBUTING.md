# Contributing to Void Slice

Void Slice is a modding tool — contributors who know Dishonored 2 / DOTO file formats are just as valuable as contributors who write Go. You don't need to write code to contribute.

## What kind of help is needed

**Non-code (start here):**
- Broken `.decl`, `.entitydef`, or `.entities` fixture files — real game files that trigger lint rules are the most valuable thing you can add
- Bug reports — unexpected output, false positives, crashes
- Documentation fixes

**Code:**
- Ready tickets live in [`kanban/todo/`](kanban/todo/) under milestone subfolders (`v1/`, `v2/`, `v4/`, `stretch/`). Each is a self-contained markdown file with acceptance criteria.

## Finding work

Browse [`kanban/todo/`](kanban/todo/) for tickets that are ready to start. v1 is essentially done (one cleanup ticket left); the most active areas now are v2 docs polish and stretch follow-ups. Pick one, edit its `**Status:**` field to `in-progress` (a hook will move the file), and open a PR when you're ready.

Not sure where to start? Reach out on [Nexus Mods](https://www.nexusmods.com/profile/kleptobismal) and we can find something that fits.

## Local setup

```
go 1.23+
make test    # runs the test suite
make build   # builds the binary
make fmt     # format before pushing
```

Full local stack (frontend + server): `docker compose up`

## PR expectations

- Small is fine — one broken fixture file is a real contribution
- For fixture/doc PRs: tests are encouraged but not required
- For code PRs: `make test` and `make vet` should pass; run `make fmt` before pushing
- Draft PRs are welcome — open early and ask questions in the thread

## What not to worry about

- No CLA, no sign-off, no DCO
- No style guide beyond `make fmt`
- No penalty for asking questions or submitting imperfect first attempts