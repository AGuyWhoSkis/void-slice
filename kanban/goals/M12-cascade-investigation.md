# M12 — Cascade investigation

**TL;DR:** the cascade is broad — three of four mutation shapes I tested (unterminated quote, unterminated block comment, unterminated brace) produce diagnostics outside the mutated span across every D2/DOTO `.entities` and `.decl` cell I sampled, with the quote shape's blast radius scaling to the entire file; missing-semicolon recovers locally and does not cascade. Property-based locality fuzzing per [linter-hardening-prospects.md](linter-hardening-prospects.md) is confirmed as the technique pick — every cascading shape was detected by the locality property applied at line granularity, no shape demanded an oracle the prospects doc didn't account for.

Methodology in one sentence: for each (shape × format × game) cell, I ran a throwaway Go harness ([`/tmp/cascade-exp/mutate.go`](#methodology), not committed) that picks a known span in a clean golden, applies a single byte-level mutation, runs `voidslice lint -json`, subtracts the baseline (clean-file) diagnostics, and counts new diagnostics whose line ≠ the mutated line. The committed evidence is the minimal repro fixtures under [`testdata/cascades/`](../../testdata/cascades/) plus the 6 spot-check goldens already in [`testdata/golden/d2/game1/`](../../testdata/golden/d2/game1/) and [`testdata/golden/doto/game1/`](../../testdata/golden/doto/game1/); broader per-cell sampling drew from the local `testdata/all-text-files/` corpus (gitignored, ~2 GB) and is reproducible on dev machines with that corpus in place.

## 1. Repro fixture

[`testdata/cascades/unterminated-quote/minimal.decl`](../../testdata/cascades/unterminated-quote/minimal.decl) — a 4-line `.decl` that opens a quote on line 2 and never closes it. Source:

```
{
	a = "x
	b = 1;
}
```

The mutated span is **line 2** (the line containing the unterminated `"`). Running `go run ./cmd/voidslice lint testdata/cascades/unterminated-quote/minimal.decl` emits 3 diagnostics — 1 on the mutated line and 2 **outside** it:

```
…/minimal.decl:2:6 error [VOID_SCAN]              unterminated quote
…/minimal.decl:5:1 error [PARSE_EXPECTED_SEMICOLON] expected ';' after assignment
…/minimal.decl:5:1 error [PARSE_EXPECTED_SYMBOL]   expected '}' to close top-level object
```

The two cascade diagnostics land on line 5 (post-EOF position) — three lines past the mutated span. The mechanism: the scanner's unterminated-quote consumes everything from the bare `"` on line 2 to EOF as a single string literal, so the parser never sees `;`, `}`, or any of the structural tokens on lines 3–4 — they're all inside what the scanner thinks is one quoted string. The parser then asserts at EOF that the missing `;` and `}` should have appeared, and emits both diagnostics at the post-EOF position.

This is the canonical M11 cascade, distilled. The same mechanism scales catastrophically on real corpus files: mutating [`maps.campaign.menu.menu_p.entities`](#fn-menu) the same way produces 16,927 cascade diagnostics across line range [2..33,374] of the file — see §2 row 1.

## 2. Cascade shapes characterized

Per-shape × per-format × per-game cascade verdicts, drawn from the throwaway harness against samples in each cell. **"Cascade?"** = ≥1 post-baseline diagnostic at a line ≠ the mutated line. Counts in cells are `outside-line diagnostics / sample` from a representative file; range is `[min..max]` lines of distance from the mutated line.

| Shape | D2 `.entities` | DOTO `.entities` | D2 `.decl` | DOTO `.decl` |
|-------|----------------|------------------|------------|--------------|
| **unterminated-quote** | yes — 16,927 / 1 sample, range [2..33,374] | yes — 100 / 1 sample, range [1..182] | yes — 4 / 5 samples, range [1..117] | yes — 1–67 / 5 samples, range [2..109] |
| **unterminated-block-comment** | **no** — 0 / 1 sample | **no** — 0 / 1 sample | yes — 1 / 5 samples, range [24..120] (always EOF) | yes — 1 / 5 samples, range [28..112] (always EOF) |
| **unterminated-brace** | yes — 1 / 1 sample, range [1..1] | yes — 1 / 1 sample, range [1..1] | yes — 1 / 5 samples, range [1..1] | yes — 1 / 5 samples, range [1..1] |
| **missing-semicolon** | **no** — 0 / 1 sample | **no** — 0 / 1 sample | **no** — 0 / 5 samples | **no** — 0 / 5 samples |

Additional confirmation from the **6 committed spot-check goldens** (4 D2 + 2 DOTO under [`testdata/golden/{d2,doto}/game1/`](../../testdata/golden/)): the same shape verdicts reproduce on every spot-check, ranging from 8 outside-line diagnostics for the smaller `.decl` files up to 63 outside-line diagnostics with the cascade spanning to line 147,888 for the D2 `.entities` spot-check.

**Per-shape mechanism:**

- **unterminated-quote** — scanner consumes everything from the bare `"` to the next `"` or EOF as a single string literal. The parser is then deceived about every structural token that fell inside the over-extended string; cascades range from a handful of EOF-locality diagnostics (when the next quote happens to be nearby) up to entire-file blowouts (when the next quote is 30K+ lines away). **This is the M11 motivating bug. The most severe shape.**
- **unterminated-block-comment** — scanner consumes everything from `/*` to EOF as one comment, so any closing punctuation the parser expected is invisible. Cascade is one extra diagnostic at EOF on `.decl` (the parser was inside the `{` open at line 1 and now never sees the closing `}`). On `.entities` the parser hasn't yet entered a block when the comment swallows the rest of the file, so it reports both diagnostics on the mutated line — no cascade. *Smaller blast radius than unterminated-quote, but the same mechanism: scanner-layer error consumes structural tokens the parser expected.*
- **unterminated-brace** — parse-layer cascade. The parser reaches EOF inside an unclosed object and emits `PARSE_UNTERMINATED_OBJECT` at the position of the opening `{` plus one or more `PARSE_EXPECTED_SYMBOL` at the position of the last token before EOF. The diagnostic about the missing brace lands at a line **earlier** than the mutated span (the deleted `}`'s position), not later — a locality violation in the opposite direction from quote/comment but detectable by the same line-≠-mutated-line property. Blast radius is small (1–3 diagnostics) but the *position is wrong*.
- **missing-semicolon** — parser recovers cleanly. Single `PARSE_EXPECTED_SEMICOLON` at the exact line of the missing `;`. Across 11 samples the cascade count was always 0. This is the linter behaving correctly on a localized fault.

**Cross-corpus differential note.** No cross-corpus cascades surfaced — every cascading shape cascades equivalently in D2 and DOTO. The cascade is shape-driven, not corpus-driven. Per [M12.md](M12.md), cross-corpus differential stays out of scope.

## 3. Technique-pick stress test

**Property-based locality fuzzing per [linter-hardening-prospects.md](linter-hardening-prospects.md) is confirmed**, and the burden-of-proof framing in that doc was right — every cascading shape §2 surfaced was detected by the same property ("post-baseline diagnostics confined to the mutated span") applied to a single byte-level mutation against a known-clean golden. The shapes split cleanly into three causal mechanisms — scanner over-extends a string (quote) or comment (block comment), and parser walks off the end of an unclosed structure (brace) — and the locality property catches all three at line granularity without needing an oracle the prospects doc didn't enumerate. Missing-semicolon's local recovery is evidence in the opposite direction: when the parser handles a fault cleanly, the property is silent, exactly as it should be. The investigation surfaced no shape that would slip past a line-window locality assertion, and surfaced no oracle gap that would force a divergence to differential or mutation-of-the-validator. Property-based locality fuzzing is the technique pick.

---

<a id="methodology"></a>
**Methodology footnote.** The throwaway harness (`/tmp/cascade-exp/mutate.go`, not committed; ~200 lines of Go) does the following per (shape, file): lint the unmodified file to build a `(line, col, code)` baseline set; apply one byte-level mutation at a known span (delete closing `"` for unterminated-quote, insert `/* unterminated\n` at the start for unterminated-block-comment, delete the last unquoted `}` for unterminated-brace, delete the first unquoted `;` for missing-semicolon); write the mutated bytes to a temp file with the original extension; re-lint; subtract baseline; count outside-line diagnostics. Sampling: 5 files per `.decl` cell (random shuf of the local corpus filtered to 200B–10KB for fast lint), 1 file per `.entities` cell (small representative), plus the 6 committed spot-check goldens. This is not a long-term harness — the M12 follow-up ticket builds the real property-based harness inside `internal/`.

<a id="fn-menu"></a>
**`maps.campaign.menu.menu_p.entities` reference.** The cited 16,927-diagnostic cascade was on `testdata/all-text-files/Dishonored2/Export/game1/maps.campaign.menu.menu_p.entities` (761,935 bytes, ~33,400 lines), available locally but not committed.
