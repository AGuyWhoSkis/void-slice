# Goals

Goals capture durable intent — the why and the scope of a body of work. They live at `kanban/goals/M<N>.md`, drafted via the `/goal-define` skill. Goals are the only persistent cross-session artifact; small and medium work runs from conversation context without a goal file.

| Goal | File |
|------|------|
| [M1 — Linter engine + CLI](goals/M1.md) | `goals/M1.md` |
| [M2 — Cloud playground](goals/M2.md) | `goals/M2.md` |
| [M3 — Editor integration (LSP)](goals/M3.md) | `goals/M3.md` |
| [M4 — Linter scope refinement](goals/M4.md) | `goals/M4.md` |
| [M5 — Dev tooling & workflow](goals/M5.md) | `goals/M5.md` |
| [M6 — Preview environments](goals/M6.md) | `goals/M6.md` |
| [M7 — Production hardening](goals/M7.md) | `goals/M7.md` |
| [M8 — Playground bisection ladder](goals/M8.md) | `goals/M8.md` |
| [M9 — Concurrent agent workflow](goals/M9.md) | `goals/M9.md` |
| [M10 — Nit capture](goals/M10.md) | `goals/M10.md` |
| [M12 — Lint engine hardening](goals/M12.md) | `goals/M12.md` |
| [M13 — Whitespace-invariance as a hard property](goals/M13.md) | `goals/M13.md` |
| [M14 — Declared-typing spike](goals/M14.md) | `goals/M14.md` |
| [M17 — Edit→lint surprise](goals/M17.md) | `goals/M17.md` |

## `Paths:` field

Active goals declare a `**Paths:**` block between `**Scope:**` and `**Status:**` — a bullet list of repo-relative globs naming the surface the goal will edit. Lets `/goal-define` surface overlap with in-flight goals at definition time.

```markdown
**Paths:**
- internal/scan/**
- internal/parse/**
```

- **Glob syntax.** Doublestar globs (`**`, `*`, `?`), repo-relative, no leading slash. Same conventions as Go tooling.
- **Overlap.** Two `Paths:` fields overlap iff some concrete file in the working tree matches a pattern in both — set-intersection of matched paths, not pattern-prefix or shared-directory.
- **Cross-cutting files always declared.** No implicit allowlist for things like `go.mod`, `Makefile`, `CLAUDE.md`, `kanban/README.md`. If two goals both edit `Makefile`, they overlap.
- **Closed goals stay bare.** Overlap checking only matters for in-flight goals; M1–M8 don't carry `Paths:` retroactively.

## Lifecycle

A goal opens with `**Status:** active` (or `parked` / `ongoing`). Work commits onto the goal branch (`goal/M<N>-<slug>`, created with `git checkout -b goal/M<N>-<slug> origin/main`). A single PR per goal targets `main` and is opened early. The PR is merged by a human when the goal closes; branch protection on `main` enforces the gate. Closing a goal is just flipping its `**Status:**` field. A retrospective on the goal file is optional, not mandatory; [goals/M2.md](goals/M2.md) and [goals/M6.md](goals/M6.md) serve as exemplars.
