# M17 — Edit→delta surprise taxonomy

**TL;DR:** edit→delta surprise across the corpus collapses to a small, mostly cascade-shaped set — **one fatal class** (single-byte flip of the file-root `{` produces O(N) `PARSE_UNEXPECTED_TOKEN` cascades, up to **223,305** codes on a 1.78 MB animset), **two minor classes** (quote-byte flip emits a bounded `VOID_SCAN`+`PARSE` cluster; line-comment unwrap on the no-trailing-newline fixture cascades 8 codes), and **one curiosity** (uncommenting a single byte position in a `.material.decl` surfaces `VOID_SCAN×2`). The negative results are louder than the positive: inter-token whitespace, blank-line jitter, line-split/join in token gaps, EOF byte append, and consistent identifier rename are all **delta-zero** across all 62 lintable corpus files. M13's inter-token-whitespace gate is doing the job it claims to.

## 1. Probing method

Probes live in [`testdata/m17-probes/`](../../testdata/m17-probes/) as a single Go program at [`testdata/m17-probes/run/`](../../testdata/m17-probes/run/) dispatching on `-family=<name>`. Each family is one `.go` file in the same package; each program walks `testdata/golden/` (62 lintable files: `.decl`, `.entities`, `.entitydef`, `.cfg`), applies a deterministic transform per file, lints both versions via `lint.New().Lint(path, src)`, and emits a finding line on any diagnostic-code-multiset diff. Findings are committed under [`testdata/m17-probes/findings/`](../../testdata/m17-probes/findings/) for reproducibility. Counts are point-in-time on the M17 goal branch.

Per the M17.1 ticket Scope table, the edit-class families probed and their substrates:

| Family | Substrate | Transform |
|---|---|---|
| Inter-token whitespace | `TestLexinvarianceHardGate` (`internal/harness/lexinvariance/lexinvariance_gate_test.go:27`) | already a hard-gate property; re-confirmed delta-zero |
| Blank-line jitter | [`run/blankline.go`](../../testdata/m17-probes/run/blankline.go) | drives existing `lexinvariance.TransformBlankLineJitter` (triage-only today) |
| Single-char punctuation flip | [`run/punctuation.go`](../../testdata/m17-probes/run/punctuation.go) | per file, flip first scanner-token byte: `{→(`, `;→,`, `"→'` |
| Line split / join | [`run/linesplit.go`](../../testdata/m17-probes/run/linesplit.go) | insert `\n` into first inter-token gap containing a space (V0); join first `\n\n` run (V1) |
| Comment toggle | [`run/commenttoggle.go`](../../testdata/m17-probes/run/commenttoggle.go) | prefix first assignment-bearing line with `// ` (V0); strip `// ` from first `// `-prefixed line (V1) |
| Identifier rename | [`run/idrename.go`](../../testdata/m17-probes/run/idrename.go) | rename every occurrence of the file's most-frequent IDENTIFIER token to `<name>_R` |
| Byte append / delete | [`run/bytemut.go`](../../testdata/m17-probes/run/bytemut.go) | append trailing space (V0); append trailing newline (V1); delete last byte (V2) |

No lexinvariance hard-gate-eligible transform was promoted by this work; the existing `BlankLineJitter` triage posture is unchanged. No engine code under `internal/scan/`, `internal/parse/`, `internal/validate/`, `internal/lint/`, `internal/report/`, `worker/`, or `cmd/voidslice-wasm/` was modified.

## 2. Surprise classes observed

Each surprise class below names: a minimal repro (file, byte offset, transform), the diagnostic-multiset diff (post-mutation `+CODE×N` only — all baselines are zero-diag), and the surface on which it bites users. Magnitudes are exact from [`testdata/m17-probes/findings/`](../../testdata/m17-probes/findings/).

### SC-1. Root-brace flip cascades to O(N) parse errors

A single-byte flip of `{` → `(` at offset 0 of a file (where the file's root token is `{`) yields a `+PARSE_UNEXPECTED_TOKEN×N` finding where N scales with file size. 41 of 62 lintable corpus files matched the precondition (first scanner-emitted SYMBOL token is `{`); all 41 produced a cascade. Magnitude ranges from **22** (a 614-byte `md6.decl`) to **223,305** (a 1.78 MB `animset.decl`).

- **Repro (worst case):** [`testdata/golden/generated.decls.animset.models.characters.small._animations_.small_generic..animset.decl`](../../testdata/golden/generated.decls.animset.models.characters.small._animations_.small_generic..animset.decl), byte offset 0: `{` → `(`. Baseline diags: 0. Variant diags: `+PARSE_UNEXPECTED_TOKEN×223305, +PARSE_EXPECTED_SYMBOL×1`.
- **Repro (typical):** [`testdata/golden/d2/game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl`](../../testdata/golden/d2/game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl), byte offset 0: `{` → `(`. Variant diags: `+PARSE_UNEXPECTED_TOKEN×832`.
- **Mechanism (suspected):** the file-root structure recognizer in the curly-shape parser sees a non-`{` opener; rather than emitting one structural error and aborting, it continues into the body and emits one `PARSE_UNEXPECTED_TOKEN` per remaining token. Generalization of the M12 cascade frame (M12 fixed three unterminated-token shapes; this is a structural-byte shape that wasn't covered).
- **Surface:** CLI prints all 223k diagnostics; WASM playground attempts to render the same list. Both are catastrophic UX on a single-byte edit at the file head.
- **Repro line:** `go run ./testdata/m17-probes/run -family=punctuation` (see findings file for the full 41-row list).

### SC-2. Quote-byte flip emits bounded VOID_SCAN+PARSE cluster

A single-byte flip of `"` → `'` at the offset of a QUOTE_LITERAL opener yields a fixed-shape cluster: `+VOID_SCAN×2, +PARSE_UNEXPECTED_TOKEN×1-7, +PARSE_EXPECTED_SEMICOLON×1` (occasionally without the semicolon term). 38 of 62 files matched the precondition; magnitude is 3-9 codes per file.

- **Repro:** [`testdata/golden/generated.decls.arktree.components.aicrew.bt.sub.combat_hideout..arktree.decl`](../../testdata/golden/generated.decls.arktree.components.aicrew.bt.sub.combat_hideout..arktree.decl), byte offset 59: `"` → `'`. Variant diags: `+PARSE_EXPECTED_SEMICOLON×1, +VOID_SCAN×2`.
- **Mechanism (suspected):** `'` is not in the scanner's accepted-byte set (one `VOID_SCAN`), and the byte at offset 59 was the *opening* `"` of a QUOTE_LITERAL — its previous closing `"` becomes a new orphan opener, which scans forward to the next `"` and consumes content that was previously two assignments. The second `VOID_SCAN` is the orphan opener's downstream byte. Bounded by the next `"` in the file (so magnitude stays single-digit even on multi-MB files).
- **Surface:** CLI + WASM playground. The magnitude is small enough to be acceptable UX, but `VOID_SCAN` reads as a scanner-internal complaint rather than the actionable "you flipped a quote" the user expects.
- **Repro line:** `go run ./testdata/m17-probes/run -family=punctuation` (38 dquote-to-squote rows in findings file).

### SC-3. Line-comment unwrap on no-trailing-newline fixture cascades

Stripping the `// ` prefix from the single line of [`testdata/golden/eof.line-comment-no-newline.decl`](../../testdata/golden/eof.line-comment-no-newline.decl) — whose content is exactly `// normal line comment, no newline at EOF` — yields `+PARSE_UNEXPECTED_TOKEN×8`. The fixture is intentionally degenerate (it exists to pin scanner behaviour at EOF inside an unterminated line comment), but the surprise stands: 3 bytes removed → 8 diagnostics.

- **Repro:** [`testdata/golden/eof.line-comment-no-newline.decl`](../../testdata/golden/eof.line-comment-no-newline.decl), byte offset 0: strip `// ` prefix.
- **Mechanism (suspected):** the file's word tokens were entirely inside a line comment; uncommenting promotes them all to identifiers in an unstructured top-level position, and each unbracketed identifier becomes an unexpected token. May be an inherent limit (a deliberately-pathological 40-byte fixture) rather than an engine bug — M17.2 to decide.
- **Surface:** CLI + WASM playground, but the affected file shape is unlikely to be authored by hand.
- **Repro line:** `go run ./testdata/m17-probes/run -family=commenttoggle`.

### SC-4. Comment unwrap surfaces hidden non-ASCII bytes

Stripping a `// ` line-comment prefix at byte offset 694 of [`testdata/golden/generated.decls.material.models.environment.archi.karnaca.nature.rock.cliff_bottom_01..material.decl`](../../testdata/golden/generated.decls.material.models.environment.archi.karnaca.nature.rock.cliff_bottom_01..material.decl) yields `+VOID_SCAN×2`. The commented-out content presumably contains a byte the scanner's accepted-byte set rejects (a common case: a non-ASCII character in a comment that becomes a scan error when uncommented).

- **Mechanism (suspected):** scanner-level — the comment was a syntactic shield for bytes that aren't otherwise allowed in token positions. Uncommenting exposes them as `VOID_SCAN`. Not cascade-shaped (delta is exactly 2), but the surprise is that a "remove three bytes" edit produces two scanner errors at a span the user didn't edit.
- **Surface:** CLI + WASM playground.
- **Repro line:** `go run ./testdata/m17-probes/run -family=commenttoggle`.

### Negative-result H3 (edit classes probed that produced no surprise)

The following families and sub-classes were probed and produced **zero** diagnostic-multiset delta across the entire 62-file lintable corpus. Each is a real datapoint about the linter's current invariants, not a silent skip.

- **Inter-token whitespace** (Reindent, TabSpace, InterTokenPadding hard-gate transforms): zero findings under `TestLexinvarianceHardGate`. M13's hard property holds.
- **Blank-line jitter** ([`run/blankline.go`](../../testdata/m17-probes/run/blankline.go)): zero findings. The transform's "top-level boundary" precondition (a non-whitespace-first-byte line, after at least one earlier such line) is not satisfied by the corpus — most lintable files are a single top-level construct rooted at `{`. The transform is correctly triage-only, and no hard-gate promotion is warranted from this corpus.
- **Line split in token gap** ([`run/linesplit.go`](../../testdata/m17-probes/run/linesplit.go) V0): inserting `\n` into the first inter-token gap containing a space produced zero findings. Stronger evidence for the M13 invariant than the gate transforms alone — a *novel* whitespace edit shape still produces delta-zero.
- **Line join from `\n\n` run** ([`run/linesplit.go`](../../testdata/m17-probes/run/linesplit.go) V1): zero findings. Same reasoning.
- **Append trailing space at EOF** ([`run/bytemut.go`](../../testdata/m17-probes/run/bytemut.go) V0): zero findings across 62 files. Confirms EOF-padding insensitivity.
- **Append trailing newline at EOF** ([`run/bytemut.go`](../../testdata/m17-probes/run/bytemut.go) V1): zero findings. The linter does not have a "missing newline at EOF" rule, and adding one is a no-op.
- **Consistent identifier rename** ([`run/idrename.go`](../../testdata/m17-probes/run/idrename.go)): zero findings. The linter is structure-only — identifier names carry no semantic load today.
- **Punctuation flip `;` → `,`** ([`run/punctuation.go`](../../testdata/m17-probes/run/punctuation.go) note=semi-to-comma): 46 findings, all `+PARSE_EXPECTED_SEMICOLON×1, +PARSE_UNEXPECTED_TOKEN×1` (magnitude 2). Proportional to the edit; not surprise.
- **Delete last byte** ([`run/bytemut.go`](../../testdata/m17-probes/run/bytemut.go) V2): 56 findings, all `+PARSE_EXPECTED_SYMBOL×1` (magnitude 1). Trimming the trailing `}` correctly reports one missing close-symbol. Proportional; not surprise.
- **Comment-out first `=` line on curly-shape `.decl` files** ([`run/commenttoggle.go`](../../testdata/m17-probes/run/commenttoggle.go) V0): zero findings on the 40 curly-shape `.decl` files in the corpus. This is itself an observation worth noting in M17.2: the parser is permissive of orphan `}` (the comment removes the matching `{`, but the orphan close-brace is tolerated). 2 findings on `.entities` files were proportional (magnitude 1).

## 3. Draft taxonomy

| Class | Edit family | Repro pointer | Preliminary suspicion |
|---|---|---|---|
| SC-1 | punctuation flip (`{` at file root) | [`findings/punctuation.txt`](../../testdata/m17-probes/findings/punctuation.txt) (41 rows with note=brace-to-paren and offset 0) | cascade-shaped (generalization of M12 cascade frame to structural-byte mutations at the parser entry point) |
| SC-2 | punctuation flip (`"` → `'`) | [`findings/punctuation.txt`](../../testdata/m17-probes/findings/punctuation.txt) (38 rows with note=dquote-to-squote) | cascade-shaped, bounded (unterminated-quote shape from M12, surfacing here because the byte flip *creates* an unterminated quote) |
| SC-3 | comment toggle (V1 unwrap on no-trailing-newline file) | [`findings/commenttoggle.txt`](../../testdata/m17-probes/findings/commenttoggle.txt) line 3 | unknown — possibly inherent limit (degenerate fixture), possibly cascade |
| SC-4 | comment toggle (V1 unwrap, mid-comment non-ASCII byte) | [`findings/commenttoggle.txt`](../../testdata/m17-probes/findings/commenttoggle.txt) line 4 | classifier-routing / scanner — comment was shielding bytes the scanner rejects; not a cascade |

## 4. Handoff to M17.2

M17.2 must root-cause **4 surprise classes** (SC-1 through SC-4) and classify each as **cascade-shaped (fixable)** or **inherent limit (B2)**. SC-1 is the highest-leverage class — a single byte at offset 0 producing a 223k-diagnostic cascade is the largest UX gap on the corpus today, and the M12 precedent (cascade gates at `M12.16/17` and the classifier carve-out at `M12.18`) is a credible fix shape. SC-2 looks like the same shape, smaller. SC-3 and SC-4 may resolve as inherent limits — both surfaced on degenerate inputs that no hand-authoring workflow produces — but M17.2 makes that call after looking at the engine.

No surprise class was inconclusive enough to require additional probing in M17.1. All 62 lintable corpus files were swept by every relevant probe.

**Probes committed under [`testdata/m17-probes/`](../../testdata/m17-probes/) and their re-run invocations:**

| Probe | Family | Re-run from repo root |
|---|---|---|
| [`run/blankline.go`](../../testdata/m17-probes/run/blankline.go) | blank-line jitter | `go run ./testdata/m17-probes/run -family=blankline` |
| [`run/punctuation.go`](../../testdata/m17-probes/run/punctuation.go) | single-char punctuation flip | `go run ./testdata/m17-probes/run -family=punctuation` |
| [`run/linesplit.go`](../../testdata/m17-probes/run/linesplit.go) | line split / join in token gap | `go run ./testdata/m17-probes/run -family=linesplit` |
| [`run/commenttoggle.go`](../../testdata/m17-probes/run/commenttoggle.go) | line-comment wrap / unwrap | `go run ./testdata/m17-probes/run -family=commenttoggle` |
| [`run/idrename.go`](../../testdata/m17-probes/run/idrename.go) | consistent identifier rename | `go run ./testdata/m17-probes/run -family=idrename` |
| [`run/bytemut.go`](../../testdata/m17-probes/run/bytemut.go) | EOF byte append / delete | `go run ./testdata/m17-probes/run -family=bytemut` |

Pre-collected findings: [`testdata/m17-probes/findings/`](../../testdata/m17-probes/findings/) (six `<family>.txt` files, one per family above).
