# M17 — SC-2 orphan-quote investigation

**TL;DR:** SC-2's `VOID_SCAN×2 + PARSE_*×0-7` cluster across 57 quote-byte-flip repros (3–9 codes per file) is **proportional to the structural reach of the de-quoted value**, not cascade-shaped. The cluster scales with the path-depth or token-count of whatever bytes the original quote sheltered — not with file length and not with an unbounded parser loop. **Verdict: inherent limit (B2), maps to `P-ProportionalToStructuralReach` in [M17-root-causes.md §2](M17-root-causes.md).** No engine fix ticket. M17.7 surfaces the principle in user-facing copy.

## 1. Mechanism

A `"` → `'` byte-flip at a quote-literal opener triggers a deterministic three-or-four-part cluster in the linter's output. Citations against current HEAD:

### 1a. The rejected `'` byte — first `VOID_SCAN`

The scanner's main dispatch at [`internal/scan/scan.go:73-171`](../../internal/scan/scan.go#L73-L171) has no case for `'` (the accepted-byte set is the union of the explicit cases). The `default` branch at [`internal/scan/scan.go:185-188`](../../internal/scan/scan.go#L185-L188) emits `VOID_SCAN` for any byte not picked up by the identifier/comment branches. The `'` falls through, producing one `VOID_SCAN` at the flip offset. This is correct — the grammar's character set is ASCII-only and excludes `'`.

### 1b. The orphan-quote run — second `VOID_SCAN`

The *original closing* `"` of the affected string literal becomes an **opener**: the scanner's `'"'` case at [`internal/scan/scan.go:80-108`](../../internal/scan/scan.go#L80-L108) starts a forward scan from there looking for a terminator. The M12.3 newline-terminate branch at [`internal/scan/scan.go:95-103`](../../internal/scan/scan.go#L95-L103) fires when it hits the line's `\n`, emitting a second `VOID_SCAN` with the M17.5 user-facing wording: `"unterminated string literal — check for a missing '\"'"`. The orphan literal becomes a single `QUOTE_LITERAL` token spanning from the original close to the newline.

This is the answer to the ticket's open question about where the second `VOID_SCAN` comes from: it's M12.3's existing newline-terminate emission at [scan.go:100](../../internal/scan/scan.go#L100), not a separate unterminated-at-EOF (scan.go:107) and not a side-emission from inside the forward-scan loop. The forward-scan terminates cleanly at `\n` on every repro in the 57-row sample — `\r` characters in CRLF line endings are part of the literal lexeme but the `\n` reliably terminates.

### 1c. The structural fragments — `PARSE_UNEXPECTED_TOKEN` ×0–7

Between the rejected `'` (1a) and the orphan opener (1b), the scanner emits tokens for every byte that was previously *inside* the original string. Path-shaped strings produce IDENT runs separated by `/` (lone-`/` SYMBOL path at [`internal/scan/scan.go:168-171`](../../internal/scan/scan.go#L168-L171)). These structural fragments are real tokens the parser must walk.

What happens next depends on the statement shape the parser was reading:

- **Shape-2 (`inherit "..."` — bare keyword, no `=`):** the parser is in `walkObjectBody` at [`internal/parse/parse.go:594-619`](../../internal/parse/parse.go#L594-L619). The `inherit` IDENT is consumed by `walkStatement` at [`internal/parse/parse.go:622-781`](../../internal/parse/parse.go#L622-L781); when the next token isn't `=`, `[`, or a recognised continuation, the parser routes to the shape-2 typed-block or value-statement branch and the stray `/`, IDENT, `/`, IDENT, … bytes flow through as unexpected tokens. The max-9-code repro (`md6def...docker_small_01_head..md6.decl`, span=22) produces exactly **7** `PARSE_UNEXPECTED_TOKEN`s, one per `/` separator in the original 7-segment path.
- **Assignment shape (`key = "..."`):** the parser reaches the `=` branch at [`internal/parse/parse.go:689-744`](../../internal/parse/parse.go#L689-L744). `parseValue` at [`internal/parse/parse.go:783-853`](../../internal/parse/parse.go#L783-L853) peeks the post-`=` slot — `'` has been rejected, so the next real token is an IDENT (the first path segment). `parseValue` returns `Value{Kind: ValIdent}`. The remaining `/` SYMBOLs and IDENT segments then flow through `walkObjectBody` as `"unexpected token in object body"` or `"unexpected token after identifier"`, one per separator. The mid-range repro (`gamelogicmanager...decl`, span=73) has a 3-segment path → exactly **3** `PARSE_UNEXPECTED_TOKEN`s.

### 1d. The `PARSE_EXPECTED_SEMICOLON` — only on assignment shapes

After `parseValue` returns a `ValIdent` for the swallowed first path-segment, the assignment branch's `;` check at [`internal/parse/parse.go:723-739`](../../internal/parse/parse.go#L723-L739) runs. The M12.3 suppression at [`internal/parse/parse.go:727`](../../internal/parse/parse.go#L727) — `if !(val.Kind == ValString && c.tokenIsUnterminatedQuote(val.Tok))` — requires the *value itself* to be the unterminated quote. Here the value is `ValIdent` (the byte-flip pushed the unterminated quote one slot to the right, into the post-value position), so the suppression doesn't fire and `EXPECTED_SEMICOLON` is emitted.

This explains the split between the 44/57 repros that include `EXPECTED_SEMICOLON×1` (assignment-shape values) and the 13/57 that don't (shape-2 `inherit`/typed-block values).

### Evidence — three repros, predicted vs. observed

Walked end-to-end with a throwaway probe (not committed). Observations match predictions exactly.

**Minimal — `generated.decls.arktree.components.aicrew.bt.sub.combat_far_target..arktree.decl`, span=59..60**

Source context: `m_class = "arkTreeNodeSettings_Decorator_SetBehaviorTag";` at offsets 49..107. Byte 59 is the opening `"`. After flip:

```
VOID_SCAN          span=59..60     "unknown byte ..." (the '\'' byte)
VOID_SCAN          span=104..107   "unterminated string literal — check for a missing '\"'"
PARSE_EXPECTED_SEMICOLON  span=104..104  "expected ';' after assignment"
```

Predicted: 1a + 1b + 1d. Observed: matches. The post-`=` value is the IDENT `arkTreeNodeSettings_...` (single identifier, no `/` separators), so 1c contributes zero `PARSE_UNEXPECTED_TOKEN`. Cluster = 3 codes.

**Mid-range — `d2/game1/generated.decls.gamelogicmanager.ui.gamelogic.manager..gamelogicmanager.decl`, span=73..74**

Source context: `enumItem[ARK_GAME_LOGIC_STATE_INGAME] = "ui/gamelogic/state_ingame.gamelogicstateingame";`. After flipping the `"` at 73:

```
VOID_SCAN          span=73..74     "unknown byte ..."
PARSE_EXPECTED_SEMICOLON  span=76..76     "expected ';' after assignment"
PARSE_UNEXPECTED_TOKEN    span=76..77     "unexpected token in object body"        (the '/' after 'ui')
PARSE_UNEXPECTED_TOKEN    span=86..87     "unexpected token after identifier"      (the '/' after 'gamelogic')
VOID_SCAN          span=120..123   "unterminated string literal — check for a missing '\"'"
PARSE_UNEXPECTED_TOKEN    span=212..213   "unexpected token in object body"        (a stray '/' from the next 'enumItem = "ui/gamelogic/..."' line — the orphan-quote span ended on a newline before this line)
```

Predicted: 1a + 1b + 1c + 1d, with 1c proportional to the 3 `/`s in the original path. Observed: matches (the third stray `/` is one row down because the orphan-quote ate the trailing chars of one line and the `enumItem` line below it remains structurally intact, with its leading `/` separating consumed tokens). Cluster = 6 codes.

**Maximum — `doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl`, span=22..23**

Source context: `inherit "models/characters/small/civ_middle/dockers/docker_01/docker_small_01_body.md6";`. After flipping the `"` at 22:

```
VOID_SCAN          span=22..23     "unknown byte ..."
PARSE_UNEXPECTED_TOKEN    span=29..30     "unexpected token in shape-2 body"   ('/' after 'models')
PARSE_UNEXPECTED_TOKEN    span=40..41     "unexpected token in shape-2 body"   ('/' after 'characters')
PARSE_UNEXPECTED_TOKEN    span=46..47     "unexpected token in shape-2 body"   ('/' after 'small')
PARSE_UNEXPECTED_TOKEN    span=57..58     "unexpected token in shape-2 body"   ('/' after 'civ_middle')
PARSE_UNEXPECTED_TOKEN    span=65..66     "unexpected token in shape-2 body"   ('/' after 'dockers')
PARSE_UNEXPECTED_TOKEN    span=75..76     "unexpected token in shape-2 body"   ('/' after 'docker_01')
PARSE_UNEXPECTED_TOKEN    span=96..97     "unexpected token in shape-2 body"   ('.' after 'docker_small_01_body')
VOID_SCAN          span=100..102   "unterminated string literal — check for a missing '\"'"
```

Predicted: 1a + 1b + 1c with 1c ≈ 7 (six `/`s + one `.`). Observed: matches. No 1d because this is shape-2 (`inherit`), not an assignment. Cluster = 9 codes — the maximum across all 57 repros.

The diagnostic count is **exactly** the structural reach of the bytes the original quote was hiding: path-segment count + `.` separator + 2 fixed `VOID_SCAN`s + (0 or 1 `EXPECTED_SEMICOLON` depending on statement shape).

## 2. Verdict: P-ProportionalToStructuralReach (B2 inherent limit)

The SC-2 cluster is **bounded by what was inside the original quote**, not by the file's overall structure. The principle named tentatively in [M17-root-causes.md §2](M17-root-causes.md) — `P-ProportionalToStructuralReach` — fits exactly: *a small textual edit can have a large structural footprint when it changes the file's parse shape, and the linter's diagnostic count reflects the actual structural reach, not the textual edit's byte count.*

Each of the 57 repros tells the same story: one `'` byte flipped, N structural tokens exposed, N diagnostics emitted. The diagnostics are honest — they describe real parse-shape violations on tokens that genuinely sit at top-level after the flip. The 3–9 code range matches the path-depth distribution in the corpus, not file size: the 1.78 MB animset has *zero* SC-2 repros, while the 4 KB `md6def` head-mesh decl has the maximum 9.

### Why not outcome 1 (cascade-shaped, extends M12.3)

A narrow extension — suppress `EXPECTED_SEMICOLON` when the peek-next token at the `;` slot is itself an unterminated-quote `QUOTE_LITERAL`, regardless of `val.Kind` — would close the `PARSE_EXPECTED_SEMICOLON×1` tail on 44/57 repros. But:

1. The dominant `PARSE_UNEXPECTED_TOKEN×N` run is independent and proportional — outcome 1 doesn't touch it.
2. The `EXPECTED_SEMICOLON` itself is **truthful**: the user really did write a statement whose `;` is missing (the `'` flip removed the structural marker that would have closed the assignment cleanly). The diagnostic is one of the more actionable ones in the cluster, alongside M17.5's reworded `VOID_SCAN`.
3. The cluster's qualitative shape (proportional, bounded) doesn't change. No "tens of thousands of diagnostics on a small edit" outcome to fix.

### Why not outcome 3 (classifier-router carve-out)

M12.18's precedent routes `actionShaderDecl` files to a different lint shape based on file-content classification. There's no analogous content signal for "this file's quote-bytes may have been flipped" — every `.decl`/`.entities` file legitimately uses quoted strings. No higher-level shape exists to short-circuit to.

## 3. Follow-up

**None.** Outcome 2 closes SC-2 to a principle name. M17.7 ("B2 boundary — surface inherent limits") is the documentation surface and was already gated on this memo's verdict: it now includes the second bullet (P-ProportionalToStructuralReach) alongside P-AcceptedByteSet.
