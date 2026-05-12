# M14 — Declared-typing investigation

**TL;DR:** _<filled by M14.3>_

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

_<filled by M14.2>_

---

_<footer filled by M14.3: methodology footnote, gameN-bucket gap, named next-goal proposal>_
