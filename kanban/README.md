# Kanban Board

## Columns

| Folder | Meaning |
|--------|---------|
| `todo/` | Prioritized and ready to start |
| `in-progress/` | Being worked on right now (move file here when you start) |
| `done/` | Finished — move file here, add a completion note at the bottom |

## Ticket format

Each ticket is a markdown file. Filename: `T<n>-short-name.md`.

Move the file between folders to track status. Do not rename it.

## Ticket order (v1 critical path)

```
T0 (scaffold) → T1 (parser) → T2 (validator) ─┐
                                                ├→ T4 (lint facade) → T5 (cli)
               T3 (report) ────────────────────┘
               T6 (testdata, can run alongside T1)
```

T3 and T6 have no upstream dependencies within v1 and can be started immediately.

## Scope

**v1:** T0–T6 (linter engine + CLI only)
**v2:** HTTP server + React playground
**v3:** LSP server
