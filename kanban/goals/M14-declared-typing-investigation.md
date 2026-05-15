# M14 — Declared-typing investigation

**TL;DR:** Audit A's filename→declared-name recipe round-trips **12/12** of the `.decl.xml` `<ResourceInfo>` records, and audit B resolves **0/3,364** of the corpus's `inheritedDecl`/`inherit` references against that 12-record table — the table is structurally vacant for the cited kinds, not approximately broken. Audit C surveyed five intra-file type-suffix rule shapes and found **none qualifies** under the FP=0 + non-zero TP bar; no diagnostic shipped.

## 1. Filename ↔ declared-name round-trip (audit A)

**Corpus:** 12 `.decl.xml` records under `testdata/golden/` (flat root). Each is a one-line `<ResourceInfo name="..." classname="..." isCompressed="..." />` element giving the asset's declared canonical logical name and its kind. Classname distribution: 2 × `animBasic`, 7 × `speechScene`, 3 × `speechBarks`.

**Forward transform** `filename → expected-declared-name`, applied byte-by-byte to each `.decl.xml` basename, in order:

1. Strip suffix `.decl.xml`.
2. Strip prefix `generated.decls.`.
3. If the next dot-separated segment is `localized`, strip `localized.<lang>.` (drop two segments).
4. Strip the next dot-separated segment (the **class-bucket**; equals the lowercased `classname` attribute).
5. Replace every `.` with `/`.
6. Collapse any `//` back to `.` (this restores the `..<type-suffix>` doubled-dot grammar present in `animBasic` filenames, e.g. `..animbasic` → `.animbasic`).

**Per-file outcome** (all 12 rows, alphabetized by filename):

| # | classname | filename (basename) | expected (transformed) | declared (`<ResourceInfo name>`) | match |
|---|-----------|---------------------|------------------------|---------------------------------|-------|
| 1 | animBasic   | `generated.decls.animbasic.models.characters.dlc01.player.billie.additives_body..animbasic.decl.xml`                       | `models/characters/dlc01/player/billie/additives_body.animbasic`                | `models/characters/dlc01/player/billie/additives_body.animbasic`                | ✓ |
| 2 | animBasic   | `generated.decls.animbasic.models.interactive.doors.doors_structure.door_single..animbasic.decl.xml`                       | `models/interactive/doors/doors_structure/door_single.animbasic`                | `models/interactive/doors/doors_structure/door_single.animbasic`                | ✓ |
| 3 | speechScene | `generated.decls.localized.brazilian.speechscene.speech.scene.dlc01.rich_district_01.player.player_flowershop.decl.xml`    | `speech/scene/dlc01/rich_district_01/player/player_flowershop`                  | `speech/scene/dlc01/rich_district_01/player/player_flowershop`                  | ✓ |
| 4 | speechScene | `generated.decls.localized.brazilian.speechscene.speech.scene.dlc01.rich_district_01.player.player_tattooinks.decl.xml`    | `speech/scene/dlc01/rich_district_01/player/player_tattooinks`                  | `speech/scene/dlc01/rich_district_01/player/player_tattooinks`                  | ✓ |
| 5 | speechBarks | `generated.decls.localized.chinese.speechbarks.speech.speech_barks.patrol.onuse.dunwall.bk_onuse_dw_guard_m.decl.xml`      | `speech/speech_barks/patrol/onuse/dunwall/bk_onuse_dw_guard_m`                  | `speech/speech_barks/patrol/onuse/dunwall/bk_onuse_dw_guard_m`                  | ✓ |
| 6 | speechBarks | `generated.decls.localized.german.speechbarks.speech.speech_barks.detection.dlc01.bk_guards_mimicsuspicious.decl.xml`      | `speech/speech_barks/detection/dlc01/bk_guards_mimicsuspicious`                 | `speech/speech_barks/detection/dlc01/bk_guards_mimicsuspicious`                 | ✓ |
| 7 | speechBarks | `generated.decls.localized.german.speechbarks.speech.speech_barks.detection.dlc01.bk_worker_mimicsuspicious.decl.xml`      | `speech/speech_barks/detection/dlc01/bk_worker_mimicsuspicious`                 | `speech/speech_barks/detection/dlc01/bk_worker_mimicsuspicious`                 | ✓ |
| 8 | speechScene | `generated.decls.localized.japanese.speechscene.speech.scene.dlc01.rich_district_02.player.player_losesauction.decl.xml`   | `speech/scene/dlc01/rich_district_02/player/player_losesauction`                | `speech/scene/dlc01/rich_district_02/player/player_losesauction`                | ✓ |
| 9 | speechScene | `generated.decls.localized.polish.speechscene.speech.scene.dlc01.rich_district_01.player.player_tattoo_chair.decl.xml`     | `speech/scene/dlc01/rich_district_01/player/player_tattoo_chair`                | `speech/scene/dlc01/rich_district_01/player/player_tattoo_chair`                | ✓ |
| 10 | speechScene | `generated.decls.localized.polish.speechscene.speech.scene.dlc01.rich_district_02.player.player_losesauction.decl.xml`     | `speech/scene/dlc01/rich_district_02/player/player_losesauction`                | `speech/scene/dlc01/rich_district_02/player/player_losesauction`                | ✓ |
| 11 | speechScene | `generated.decls.localized.russian.speechscene.speech.scene.dlc01.rich_district_02.player.player_auction_mimic.decl.xml`   | `speech/scene/dlc01/rich_district_02/player/player_auction_mimic`               | `speech/scene/dlc01/rich_district_02/player/player_auction_mimic`               | ✓ |
| 12 | speechScene | `generated.decls.localized.russian.speechscene.speech.scene.dlc01.rich_district_02.player.player_sees_bloodfly.decl.xml`   | `speech/scene/dlc01/rich_district_02/player/player_sees_bloodfly`               | `speech/scene/dlc01/rich_district_02/player/player_sees_bloodfly`               | ✓ |

**Headline match rate: 12/12.** No failure shapes were observed — every filename round-trips exactly to its `<ResourceInfo name>` declared canonical name under the six-step recipe above. Two structural footnotes are worth recording even at 100%: (a) the doubled-dot `..<type-suffix>` grammar shows up only on the two `animBasic` filenames in this 12-record sample, but the same grammar appears in `.decl` files outside the `<ResourceInfo>` set (e.g. `testdata/golden/generated.decls.entitydef.models.characters.base.elite..def.decl`) so step 6 is load-bearing for the broader corpus; (b) declared names collide across language variants — row 8 (japanese) and row 10 (polish) produce the same declared name `speech/scene/dlc01/rich_district_02/player/player_losesauction`, confirming that the `localized.<lang>.` infix is non-load-bearing at the declared-name level.

## 2. Inherit-graph resolution per kind (audit B)

**Reference set.** Walking every `.decl`, `.entities`, and `.entitydef` file under `testdata/golden/` recursively yields:

- **`inheritedDecl = "..."` references:** **1,635** total occurrences.
- **`inherit = "..."` references (line-anchored, top-level form):** **1,729** total occurrences.
- **Combined: 3,364 references** to attempt to resolve against the 12-record declared-name table.

**Per-kind table.** *Kind* = the substring from the last `.` in the cited target up to the next `/` (or end-of-string), per the ticket's definition coarsened so `<path>.<ext>/<entity>` targets (e.g. `models/.../bucket_fire_01.prefab/book_01_10`) bucket as `prefab`, not `prefab/book_01_10`. The `inheritedDecl` kind universe is dominated by 46 distinct `cpnt<X>` kinds; the table shows the top kinds by occurrence count plus a roll-up of the long tail, the `(no-suffix)` bucket explicitly, and every kind that appears in the top-level `inherit` form.

| reference form  | kind                       | occurrences | resolved | rate  |
|-----------------|----------------------------|------------:|---------:|-------|
| `inheritedDecl` | `cpnthealth`               |         377 |        0 | 0.00% |
| `inheritedDecl` | `cpntmidnightproxy`        |         271 |        0 | 0.00% |
| `inheritedDecl` | `cpntaudio`                |         186 |        0 | 0.00% |
| `inheritedDecl` | `cpntaiattentionsource`    |         142 |        0 | 0.00% |
| `inheritedDecl` | `cpntloddriver`            |          99 |        0 | 0.00% |
| `inheritedDecl` | `cpntloot`                 |          66 |        0 | 0.00% |
| `inheritedDecl` | `cpntreferenceframe`       |          57 |        0 | 0.00% |
| `inheritedDecl` | `cpntanimbase`             |          54 |        0 | 0.00% |
| `inheritedDecl` | `cpnttrigger`              |          46 |        0 | 0.00% |
| `inheritedDecl` | `cpntpatrolwaypoint`       |          45 |        0 | 0.00% |
| `inheritedDecl` | other 36 `cpnt<X>` kinds   |         292 |        0 | 0.00% |
| `inherit`       | `def`                      |         954 |        0 | 0.00% |
| `inherit`       | `(no-suffix)`              |         423 |        0 | 0.00% |
| `inherit`       | `prefab`                   |         349 |        0 | 0.00% |
| `inherit`       | `fx`                       |           2 |        0 | 0.00% |
| `inherit`       | `cpntplayerpowersystem`    |           1 |        0 | 0.00% |

Across both forms: **0/3,364 references resolve** against the 12-record table. The kind universe observed across the corpus (52 distinct kinds including `(no-suffix)`) does not overlap with the three kinds the table actually covers (`animBasic`, `speechScene`, `speechBarks`) — the only file in the audit-B reference set whose target string matches a `<ResourceInfo>` declared-name would have to be a literal `inheritedDecl = "models/characters/dlc01/player/billie/additives_body.animbasic"` (or one of the other 11 declared names), and the corpus contains zero such strings.

**Suffix-tolerance list.** For any future resolver to bridge between corpus reference strings and the `<ResourceInfo>` table — assuming a `<ResourceInfo>` table that covered the kinds actually cited — the following transformations would have to be encoded. Each is illustrated by one concrete corpus path:

1. **Path-separator translation: `.` (filename) ↔ `/` (declared name).** A resolver matching a reference like `models/characters/dlc01/player/billie/additives_body.animbasic` against the file system would have to translate `.` to `/` for path-separating dots but not for the type-suffix dot. *Example:* `<ResourceInfo>` row 1 in §1 — declared name `models/characters/dlc01/player/billie/additives_body.animbasic` lives in `testdata/golden/generated.decls.animbasic.models.characters.dlc01.player.billie.additives_body..animbasic.decl.xml`.
2. **Class-bucket prefix stripping: `generated.decls.[localized.<lang>.]<classbucket>.` ∉ declared name.** The filename carries a leading class-bucket prefix (and, for localized assets, a language-bucket infix) that the declared name omits. *Example:* `testdata/golden/generated.decls.localized.chinese.speechbarks.speech.speech_barks.patrol.onuse.dunwall.bk_onuse_dw_guard_m.decl.xml` — the `localized.chinese.speechbarks.` prefix is absent from the declared name `speech/speech_barks/patrol/onuse/dunwall/bk_onuse_dw_guard_m`.
3. **Doubled-dot `..<type-suffix>` filename grammar.** The filename uses `..<suffix>` as a path-vs-type-suffix separator; the declared name has `.<suffix>`. *Example:* `testdata/golden/generated.decls.entitydef.models.characters.base.elite..def.decl` uses `..def` to terminate the `models.characters.base.elite` path before the `def` type-suffix.
4. **Classname case-folding between attribute and filename.** `<ResourceInfo classname="animBasic"/>` vs the lowercased `animbasic` class-bucket segment in the filename. *Example:* `testdata/golden/generated.decls.animbasic.models.interactive.doors.doors_structure.door_single..animbasic.decl.xml` (filename has `animbasic` twice; the record itself reads `classname="animBasic"`).
5. **`.def` optional on cited targets.** Top-level `inherit` strings mix kinds: `def`-suffixed (e.g. `ai/hideout_spot.def`) and no-suffix (e.g. `info/debug_text`, `blacksparrow/patrolroute`) both appear as inherit targets. A resolver could not assume suffix presence on the reference side. *Example:* both `inherit = "ai/hideout_spot.def"` and `inherit = "info/debug_text"` occur in `testdata/golden/d2/game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities`.
6. **`localized.<lang>.` filename infix is non-load-bearing — multiple files share a declared name.** Two of the 12 records (`...japanese.speechscene...player_losesauction` and `...polish.speechscene...player_losesauction`) declare the same name `speech/scene/dlc01/rich_district_02/player/player_losesauction`. A reference→file resolver would have to lookup across all `localized.<lang>` variants and either pick one or surface the ambiguity. *Example:* `testdata/golden/generated.decls.localized.japanese.speechscene.speech.scene.dlc01.rich_district_02.player.player_losesauction.decl.xml` and `testdata/golden/generated.decls.localized.polish.speechscene.speech.scene.dlc01.rich_district_02.player.player_losesauction.decl.xml`.

**Conclusion.** The observed resolution rate is **0.00% (0/3,364)** of all `inheritedDecl` and `inherit` references in `testdata/golden/`. The corollary holds: the 12-record `<ResourceInfo>` table is several orders of magnitude smaller than what a resolver would need to cover the corpus's reference set — even setting aside the suffix-tolerance list above (which assumes the table *would* cover the cited kinds), the table covers three kinds (`animBasic`, `speechScene`, `speechBarks`) while the corpus cites 52 distinct kinds, none of which overlap. Cross-file resolution against the present intra-workspace declared-name source is not approximately broken, it is structurally vacant.

## 3. Type-suffix check (audit C)

**Rule survey.** `inheritedDecl` in the corpus always appears as the immediate child of an `item_<verb>["HASH"] = { ... }` body — 1,635 of 1,635 entries match the two-line pattern `item_<verb>["HASH"] = {\n[ \t]*inheritedDecl = "...\.<suffix>"` — and every value's last-dot suffix is a `cpnt<X>` identifier (46 distinct, headed by `cpnthealth` × 377, `cpntmidnightproxy` × 271, `cpntaudio` × 186). The original goal framing assumed an enclosing `component { cpntFoo … }` block (M14.md §1); the corpus places these `item_*[H]` bodies inside `entity { entityDef name { ... m_componentDecls = { ... } } }` blocks, so "enclosing component" admits more than one definition. Five candidate rules were enumerated and measured.

| ID | Rule shape | Pros | Cons |
|----|------------|------|------|
| A  | same `item_*["H"]` hash → same `.cpnt<X>` suffix, **file-wide** | matches the hash-as-component-slot mental model | conflates hashes across entities |
| A1 | same `item_<verb>["H"]` hash → same suffix, **file-wide, per-verb** | tighter than A | same scope-collision risk |
| A2 | same `item_*["H"]` hash → same suffix, **scoped to enclosing `m_componentDecls = {}` block** | the verb's actual semantic boundary | depends on the corpus exercising the pattern |
| B  | inside `component { cpntFoo … }`: `inheritedDecl` suffix must equal `cpntFoo` | matches the original M14.md framing | frame doesn't apply — `inheritedDecl` doesn't live in `component {}` bodies in the corpus |
| C  | `inheritedDecl = "X.<suffix>"` where `<suffix>` appears as an identifier sibling key in the same object | structural, no scope state | siblings of `inheritedDecl` in the corpus are never `cpnt<X>` keys |

**Per-candidate measurement** (62 `.decl` / `.entities` / `.entitydef` files under [`testdata/golden/`](../../testdata/golden/), 1,635 `inheritedDecl` entries; scripts at `/tmp/m14-audit-c/audit.py` and `/tmp/m14-audit-c/audit2.py`):

| Rule | Fires on golden | TPs (real bugs) | FPs (legitimate constructs) | Sample |
|------|----------------:|----------------:|----------------------------:|--------|
| A    |              63 |               0 |                          63 | `maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities:182775` — `item_add["100000"]` is reused across 216 `entity { entityDef … }` blocks in one file, legitimately mapping to `cpntpatrolroute` in one entity and `cpntpatrolwaypoint` in another |
| A1   |              63 |               0 |                          63 | same |
| A2   |               0 |               0 |                           0 | corpus has zero duplicate hashes within any single `m_componentDecls = {}` block; rule is structurally empty against this corpus |
| B    |           1,635 |               0 |                       1,635 | every `inheritedDecl` in the corpus lives in an `entity { entityDef name { … m_componentDecls = { item_*[H] = { inheritedDecl } } } }` chain — no enclosing `component { cpnt<X> … }` block exists |
| C    |           1,635 |               0 |                       1,635 | siblings of `inheritedDecl` in `item_*[H]` bodies are `inlineDecl`, `mapComponentDecl`, etc. — never `cpnt<X>` identifiers |

**Verdict — none qualifies.** Two threshold checks: (1) **FP=0 against golden** is met only by A2 (and the unlisted D variant "duplicate hash within block, regardless of suffix" — same shape, also 0). (2) **Non-zero TP against golden**: zero for every candidate. The corpus is empirically clean of intra-file `inheritedDecl` type-suffix inconsistency; the rule shapes that survive the FP filter (A2, D) survive only by firing nowhere — they have no observable signal on real game files. Partial-context-graceful was a non-issue here — all five rules are pure functions of the parsed token stream — but it stops mattering once TP=0 forces the verdict.

**Ship outcome — no diagnostic shipped.** Precise blocker: **zero-TP across all candidates** on [`testdata/golden/`](../../testdata/golden/). No `VALIDATE_INHERITED_DECL_TYPE_SUFFIX` code added; no `internal/`, `cmd/`, `worker/`, or `testdata/broken/` files touched. The "type-suffix check" candidate that survived M14's original filter as the only intra-file-shippable rule does not survive contact with the corpus. What would change that: corpus acquisition of a file with a hand-edit bug of the A2 shape (duplicate `item_*[H]` with mismatched `cpnt<X>` suffix within one `m_componentDecls` block), or a different intra-file rule shape sourced from a class of construct this audit didn't enumerate.

---

<a id="methodology"></a>
**Methodology footnote.** Audits A and B were run by `/tmp/m14-audit/audit.go` (~234 lines of Go, not committed): walks `testdata/golden/` for `.decl.xml` records, applies §1's six-step filename→declared-name transform, then walks every `.decl`/`.entities`/`.entitydef` file for `inheritedDecl = "..."` and line-anchored `inherit = "..."` references and tabulates resolution against the declared-name set. Audit C's rule-survey and FP measurement were run by `/tmp/m14-audit-c/audit.py` (~187 lines) and `/tmp/m14-audit-c/audit2.py` (~148 lines), not committed: the first measures the four originally-enumerated rule shapes file-wide; the second adds the per-`m_componentDecls`-block-scoped variant. Both audit harnesses are reproducible on a dev machine with the golden corpus in place — they only read `testdata/golden/`.

**gameN-bucket gap.** The corpus under [`testdata/golden/`](../../testdata/golden/) covers `d2/game1` and `doto/game1` only — the `gameN` bucket beyond N=1 is absent for both titles. Extending audits A and B to a wider declared-name set would require acquiring per-bucket `.decl.xml` `<ResourceInfo>` files for the top-10 cited kinds from §2 (`cpnthealth`, `cpntmidnightproxy`, `cpntaudio`, `cpntaiattentionsource`, `cpntloddriver`, `cpntloot`, `cpntreferenceframe`, `cpntanimbase`, `cpnttrigger`, `cpntpatrolwaypoint`) plus their corresponding `.def`/`.entities`/`.entitydef` siblings to verify that the round-trip and resolution rates measured here generalise. The spike does not commit to that acquisition — it states the gap so a follow-up goal can scope it deliberately.

## Next goal

**M15 (proposed): Acquire `.decl.xml` fixtures for the top-N cited kinds and re-measure audits A and B at scale.** §1 establishes that the filename→declared-name forward transform is byte-exact on every `.decl.xml` record present (12/12), and §2 establishes that the 12-record table covers 3 of the 52 cited kinds — so a resolver built today against the current corpus would return zero hits on 100% of corpus references. The bottleneck measured by M14 is not engine capability (the transform works) and not rule design (every intra-file shape we surveyed is either zero-TP or zero-FP-violating, per §3); it is **declared-name coverage**. M15 would acquire `.decl.xml` `<ResourceInfo>` records and matching reference files for the top-10 cited kinds from §2's per-kind table, re-run the audit-A round-trip on the wider record set, and re-run audit B's resolution-rate measurement to learn whether the structural-vacancy finding is an artifact of the 12-record sample size or a deeper property of how declared names and references coexist across buckets. Diagnostic shippability is not in M15's scope — it is gated on what the re-measurement shows. The parked G-A (schema-synthesis) candidate-goal framing from [`kanban/goals/M14.md`](M14.md) is **retired** by this proposal: §1 already invalidates the synthesis trap — declared names are recorded, not synthesised. G-B (cross-file refs) is **narrowed, not retired**: §2 invalidates only the `<ResourceInfo>`-mediated resolver framing (the `<ResourceInfo>` target is empty at corpus scale), but file-tree-mediated resolution against the corpus's existing `.decl`/`.entitydef`/`.entities` files is an unmeasured second path — M16 isolates that path.
