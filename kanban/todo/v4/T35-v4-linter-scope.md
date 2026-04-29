# T35 · v4 linter scope — analysis carry-forward & refinement

**Status:** todo
**Version:** v4
**Size:** medium (provisional — re-size after the refinement conversation)

## What

Park the diagnostic-coverage analysis surfaced during the T29/T34 cleanup window so a future v4 effort starts with the findings already on hand instead of re-discovering them. The analysis is preserved verbatim in **§ Reference** below; the **§ Open questions** capture work parked for later judgement.

This ticket is intentionally underspecified. v4 scope cannot be locked down until (a) the Cloudflare playground (T12) is live and producing real-world diagnostics on user input, and (b) the user makes a deliberate call about which of the open questions belong in v4 versus stretch versus "won't-fix." The first scope item is therefore a **mandatory refinement conversation** — no implementation begins until the placeholder bullets below are replaced with concrete deliverables.

## Open questions

These are parked, not committed. Each one is a v4 candidate:

1. **`PARSE_UNEXPECTED_TOKEN` false-positive class.** Non-`component` `.decl` sub-types (iggyfile, activeragdoll, renderprog, prefab, …) emit ~1900 of these against the 6-file corpus-mini. Labelled "fix before T5" in [internal/lint/clean_sweep_test.go:67-74](internal/lint/clean_sweep_test.go#L67-L74) but never fixed. Need to determine whether this is a parser-model gap (T1 didn't cover those grammars) or a deliberate exclusion that should be silenced rather than diagnosed.
2. **Semantic / schema awareness.** Today's linter is a grammar checker, not a contract checker — it doesn't know that `cpntPlayerFxManager` has field `m_maxValue` and not `m_maxValu`. Adding this would be a fundamental scope shift (likely a separate `internal/schema/` package, plus a per-component-type schema corpus). Cost-of-implementation vs. modder value is unclear.
3. **`VALIDATE_NULL_REFERENCE` revival.** [kanban/todo/stretch/T-null-ref.md](kanban/todo/stretch/T-null-ref.md) was scoped to add a 14th rule but gated on a corpus-sweep prerequisite that pointed at the now-deleted `void-files/` tree. The gate is stale; the rule itself may still be worth landing. Decide: absorb into v4, rewrite the gate against `testdata/corpus-mini/`, or close as won't-do.
4. **Array-rule hits on "valid" files.** `TestCoverageAudit` reports 7× `VALIDATE_ARRAY_COUNT_MISMATCH` and 1× `VALIDATE_ARRAY_MISSING_NUM` against the corpus-mini files, which are nominally valid game exports. Either the rules have false positives on real data, or the data has actual bugs that ship in-game. Uninvestigated. Worth one focused session.
5. **Live re-lint in the playground.** The Cloudflare path is one-shot upload → diagnostics. The LSP server in [internal/lsp/](internal/lsp/) already supports incremental re-lint but isn't wired to the web frontend. Worth a UX-cost/value sketch before scoping.

## Scope

> ⚠️ **Mandatory first step.** The bullets below are placeholders. No code changes until the refinement conversation has happened and these have been rewritten as concrete, verifiable deliverables.

- **Refinement conversation with user.** Walk through § Open questions, decide which belong in v4. Rewrite this Scope section in place (replace this bullet and the placeholders below with the agreed deliverables). Re-size the ticket if scope changes materially.
- _(placeholder)_ Resolution for open question 1 — `PARSE_UNEXPECTED_TOKEN` false positives
- _(placeholder)_ Resolution for open question 2 — semantic / schema awareness
- _(placeholder)_ Resolution for open question 3 — `VALIDATE_NULL_REFERENCE`
- _(placeholder)_ Resolution for open question 4 — array-rule hits on valid corpus
- _(placeholder)_ Resolution for open question 5 — live re-lint UX

## Dependencies

- T12 (Cloudflare deploy) — need real-world playground feedback before committing to rule-set changes
- May absorb [kanban/todo/stretch/T-null-ref.md](kanban/todo/stretch/T-null-ref.md) depending on the refinement outcome

## Verification

- Refinement conversation has occurred and § Scope contains concrete deliverables (no remaining `_(placeholder)_` bullets)
- Each landed sub-item has its own acceptance criterion appended to that bullet
- `go test ./...` and `go vet ./...` clean
- A `## Completion` section summarises which open questions were addressed, which were deferred to stretch, and which were closed as won't-do

## Reference — analysis carry-forward (2026-04-28)

Captured during the post-T34 cleanup conversation. The intent is to preserve enough detail that a future session can pick up without rerunning the investigation.

### Implemented diagnostic codes (13 total)

| Layer | Code | Fires when… |
|---|---|---|
| scan | `VOID_SCAN` | unterminated quote / number, unknown byte |
| scan | `VOID_SCAN_STRUCTURE` | nested structural lex error |
| parse | `PARSE_UNEXPECTED_TOKEN` | token doesn't fit grammar at that position |
| parse | `PARSE_EXPECTED_SYMBOL` | missing `{`, `}`, `=` |
| parse | `PARSE_EXPECTED_IDENTIFIER` | identifier expected, got something else |
| parse | `PARSE_EXPECTED_SEMICOLON` | missing `;` after assignment |
| parse | `PARSE_UNTERMINATED_OBJECT` | `{` never closed |
| validate | `VALIDATE_ARRAY_COUNT_MISMATCH` | `num=3` but only 2 `item[]` entries |
| validate | `VALIDATE_ARRAY_INDEX_OOB` | `item[5]` when `num=2` |
| validate | `VALIDATE_ARRAY_DUP_INDEX` | two `item[0]` entries |
| validate | `VALIDATE_ARRAY_MISSING_NUM` | array has items, no `num=` |
| lint facade | `LINT_BINARY_FILE` | extension `.bwm`/`.tome`/etc. or null bytes detected |
| lint facade | `LINT_VE_INCONSISTENCY` | warning attached to every `.entities` / `.cfg` upload |

Source-of-truth: [internal/scan/scan_constants.go:49-55](internal/scan/scan_constants.go#L49-L55), [internal/parse/parse_constants.go:5-17](internal/parse/parse_constants.go#L5-L17), [internal/validate/validate_constants.go:5-15](internal/validate/validate_constants.go#L5-L15), [internal/lint/lint.go:39-105](internal/lint/lint.go#L39-L105).

### Why `TestCoverageAudit` only reports 4 codes

Corpus-mini contains 6 nominally valid game files. The unobserved 9 codes are not unimplemented — they simply require broken inputs:

- `VOID_SCAN`, `VOID_SCAN_STRUCTURE`, all four `PARSE_EXPECTED_*` and `PARSE_UNTERMINATED_OBJECT` only fire on lex/parse errors → no instances in valid files.
- `VALIDATE_ARRAY_INDEX_OOB`, `VALIDATE_ARRAY_DUP_INDEX` aren't present in the corpus-mini files.
- `LINT_BINARY_FILE` requires a binary extension; covered separately by `TestBinarySweep` against `testdata/binary/sample.bwm`.
- `PARSE_UNEXPECTED_TOKEN` fires 1900× because of the false-positive class flagged in open question 1.

The five [testdata/broken/](testdata/broken/) fixtures (count-mismatch, dup-index, index-oob, missing-semicolon, unterminated-object) cover the most common error modes for manual playground testing.

### Backlog inventory (todo / stretch)

| Ticket | Adds new lint code? | Notes |
|---|---|---|
| [T12-deploy.md](../v2/T12-deploy.md) | no | The Cloudflare deploy itself |
| [T13-docs.md](../v2/T13-docs.md) | no | Docs polish, blocked on T12 |
| [T24-cli-polish.md](../v1/T24-cli-polish.md) | no | ANSI color, multi-file args, `--strict` — CLI ergonomics only |
| [T-null-ref.md](../stretch/T-null-ref.md) | `VALIDATE_NULL_REFERENCE` | Stretch; gate references the deleted `void-files/` corpus |
| [T-v3-followups.md](../stretch/T-v3-followups.md) | no | LSP polish (T29-T32 children) |
| [T-k3d.md](../stretch/T-k3d.md) | no | Superseded by Cloudflare decision; effectively dead |

### Honest UX framing

Strong value today: catching the 5 most common edit-time syntax breaks; the four array-bookkeeping rules; refusing binary uploads cleanly; warning on `.entities`/`.cfg` round-trip risk.

Known weak spots: no schema awareness; the `PARSE_UNEXPECTED_TOKEN` false-positive class; no live re-lint in the playground.

The truthful playground headline is: *"Paste a `.decl` or `.entities` file and find structural errors before launching the game."* Don't overstate.
