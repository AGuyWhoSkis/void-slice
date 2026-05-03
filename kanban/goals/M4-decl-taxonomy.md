# M4 — `.decl` format taxonomy & dispatch decision

The `.decl` extension covers five distinct top-level grammars, plus an XML sidecar variant. This doc is the inventory + dispatch decision that future M4 grammar tickets cite. It is the deliverable of M4.1.

All counts were derived by content-sniff (not filename) over `testdata/golden/` — the flat root plus the `d2/game1/` and `doto/game1/` spot-check subpaths. Universe: 57 `.decl` + 12 `.decl.xml` + 1 `.entities`.

## Corpus inventory

| Shape | Description | Count | Status |
|-------|-------------|-------|--------|
| 1 | curly inherit/edit (`= ;` separated) | 44 | in scope |
| 2 | animset (`previewmd6`, whitespace + quoted) | 7 | in scope |
| 3 | material (`m_PhysicsMaterial`, tab-aligned, no `=`) | 3 | in scope |
| 4 | renderprog (`newstyle` + embedded HLSL) | 2 | **out of scope (G-C)** |
| 5 | md6def (`init { … }` whitespace + quoted) | 1 | in scope |
| sidecar | `.decl.xml` Void Explorer metadata | 12 | in scope |

The existing `.entities` file (`maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities`) is already supported by the M1 parser and is not part of the dispatch problem.

## Shape details

### Shape 1 — curly inherit/edit (44 files)

Curly-braced root, `=` and `;` separated, identifier keys, scalar/quoted/object values. Sub-types include `arktree`, `cpntplayer*`, `entitydef`, `fx`, `kiscule`, `mapinfo`, `midnightscene`, `particledt`, `physicsmaterial`, `rulehandler`, `localized.*.speechbarks`, `localized.*.speechscene`. Representative: [d2/game1/generated.decls.cpntplayerfxmanager.…decl](../../testdata/golden/d2/game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl).

```
{
    inherit = "components/characters/player/base/powers/playerpowersystem.cpntplayerpowersystem";
    edit = {
        m_someField = "value";
    }
}
```

### Shape 2 — animset (7 files)

Curly-braced root, **no** `=`, **no** `;`, whitespace-separated. Top-level `previewmd6 "…"` and `skeleton "…"`, nested `groups { "…" }`, `aliases { group N alias { name "…" … } }`. Representative: [golden/generated.decls.animset.…timeshift..animset.decl](../../testdata/golden/generated.decls.animset.models.weapons.timeshift_device.timeshift..animset.decl).

```
{
    previewmd6 "models/weapons/timeshift_device/timeshift_device.md6"
    skeleton "models/weapons/timeshift_device/timeshift_device.md6skl"
    groups { "_animations_" }
    aliases {
        group 0
        alias { name "timeshift_scan_loop_add" … }
    }
}
```

### Shape 3 — material (3 files)

Curly-braced root, no `=`, no `;`, tab-aligned key-value pairs. `m_PhysicsMaterial "…"` at top, then `version N`, `state { }`, `options { … }`. Representative: [golden/generated.decls.material.…void_rock_01..material.decl](../../testdata/golden/generated.decls.material.models.environment.voidhouse.rock_set.void_rock_01..material.decl).

```
{
    m_PhysicsMaterial    "contactsystem/env.stone"
    version    5
    state { }
    options {
        hasBumpMap        1
        hasSpecularMap    0
    }
}
```

### Shape 4 — renderprog (2 files, **out of scope per G-C**)

Curly-braced root, `newstyle` token, `state { … }`, `hlsl_prefix { … }` containing HLSL preprocessor directives + shader source. Representative: [golden/generated.decls.renderprog.arksssblur.decl](../../testdata/golden/generated.decls.renderprog.arksssblur.decl). Counted here so dispatch routes them to a no-op handler; grammar work belongs to a future G-C goal.

### Shape 5 — md6def (1 file)

**Not in M4.1's pre-classification — discovered during content-sniff.** Curly-braced root containing a top-level `init { … }` block. Whitespace-separated, quoted-string values, integer literals (e.g. `lod 0 { mesh "…" }`). Lexically near-identical to Shape 2; the Shape 2 grammar ticket should fold this in rather than carve a separate ticket. Representative: [doto/game1/generated.decls.md6def.…docker_small_01_head..md6.decl](../../testdata/golden/doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl).

```
{
    init {
        inherit "models/characters/small/civ_middle/dockers/docker_01/docker_small_01_body.md6"
        lod 0 {
            mesh "models/characters/small/civ_middle/dockers/docker_01/docker_small_01_head.md6mesh"
        }
    }
}
```

### Sidecar — `.decl.xml` (12 files)

Single self-closing XML element `<ResourceInfo name="…" classname="…" … />`. Void Explorer export/import metadata, not a game artifact. Lint contract is "validate the XML well-formedness and emit nothing else." Representative: [golden/generated.decls.animbasic.…additives_body..animbasic.decl.xml](../../testdata/golden/generated.decls.animbasic.models.characters.dlc01.player.billie.additives_body..animbasic.decl.xml).

## Dispatch decision: hybrid (extension + content-sniff)

**Rule.** Extension picks XML vs. text; content-sniff picks among shapes 1–5.

- `.decl.xml` → XML handler (extension-based; no ambiguity).
- `.decl` → tokenize the first non-whitespace token after the leading `{`:
  - `previewmd6` → Shape 2
  - `m_PhysicsMaterial` → Shape 3
  - `newstyle` → Shape 4 (route to no-op handler)
  - `init` → Shape 5 (route to Shape 2 grammar)
  - otherwise → Shape 1

**Why hybrid, not filename-infix.** The existing `golden/doto/game1/generated.decls.physicsmaterial.contactsystem.weapons.decl` carries `physicsmaterial`/`material` in its filename but is structurally Shape 1 (`{ inherit = "…"; edit = { … } }`), not Shape 3. Filename-infix dispatch would mis-route it. Content-sniff is robust against this kind of naming drift.

**Why not pure content-sniff.** XML and text grammars are reliably split by extension. Doing extension first keeps the sniff small (5 brace-leading tokens) and gives a clear error path when an XML file lands in a `.decl` slot or vice versa.

**Where dispatch lives.** `internal/parse/` (new file, e.g. `dispatch.go`); `WalkEntities` becomes the Shape-1 (curly inherit/edit) grammar after the dispatch layer is added, with new walkers for shapes 2/3/5 and a no-op walker for shape 4.

## Allowlist inventory for M4.2

Fresh as of 2026-05-02. Total: **754,265 diagnostics across 70 files.** VOID_SCAN row revised post-M4.1.1 (see below).

| Code | Count | Sev | Concentration |
|------|-------|-----|---------------|
| PARSE_UNEXPECTED_TOKEN | 751,470 | E | Shape 2: 628,623 · Shape 1: 120,990 · Shape 4: 1,213 · Shape 3: 502 · Sidecar: 120 · Shape 5: 22 |
| VOID_SCAN | 828 | E | Shape 4: 548 · Shape 3: 244 · Sidecar: 36 (post-M4.1.1; was 2,786) |
| PARSE_EXPECTED_SYMBOL | 329 | E | Shape 2: 306 · Shape 4: 23 |
| PARSE_EXPECTED_SEMICOLON | 1 | E | Shape 1 |
| VALIDATE_ARRAY_COUNT_MISMATCH | 7 | W | existing `.entities` |
| VALIDATE_ARRAY_MISSING_NUM | 1 | W | existing `.entities` |
| LINT_VE_INCONSISTENCY | 1 | W | existing `.entities` (correct behavior, *not* in allowlist) |

**VOID_SCAN residual after M4.1.1.** All 828 hits come from the scanner's catch-all "unknown byte" branch on bytes the lex layer doesn't recognize. By shape:

- **Shape 4 renderprog (548)** — HLSL bytes (`# $ < > | @ ? ! & ~ * + . /`) inside `hlsl_prefix { … }`. Permanently allowlisted per G-C carve-out; the no-op walker doesn't lex shader source.
- **Shape 3 material (244)** — bare-path values and namespaced option keys: `/` (211, e.g. `models/.../arm.tga`, `wrinkles/enable`) and `.` (33, file extensions). Drains when the Shape 3 grammar ticket teaches the scanner or grammar to recognize bare paths.
- **Sidecar `.decl.xml` (36)** — `< / >` from `<ResourceInfo … />`. Drains when the XML handler displaces the text scanner for `.decl.xml`.

Shape 1 (5), Shape 2 (16), and Shape 1/3/4 punctuation (`,` `(` `)` `:`, 1,937 hits across all shapes) drained in M4.1.1 by promoting those four bytes to `SYMBOL` tokens — they are real grammar in shape 1 tuples (`color = ( 1, 1, 1 );`), shape 2 scoped names (`Foo::Bar`), and shape 3 function-call values.

**M4.2's starting allowlist:**

- Allow `PARSE_UNEXPECTED_TOKEN`, `VOID_SCAN`, `PARSE_EXPECTED_SYMBOL`, `PARSE_EXPECTED_SEMICOLON` across the entire `golden/` tree (flat root + `d2/game1/` + `doto/game1/`). These drain as Shapes 1–3 and 5 grammars land. Shape 4 (renderprog) is permanently allowlisted per G-C carve-out.
- Allow `VALIDATE_ARRAY_COUNT_MISMATCH` (×7) and `VALIDATE_ARRAY_MISSING_NUM` (×1) on the existing `.entities` (separate Phase C work).
- `LINT_VE_INCONSISTENCY` (×1) is the Void-Explorer warning behaving correctly — it must **not** be in the allowlist.

**M4 closes when** the allowlist drains to (Shape-4-only PARSE/VOID + 8 entities-validate-warns), i.e. once the in-scope grammars stop emitting false positives.

## Game-subdir decision

`testdata/golden/`'s flat root holds the broad corpus alongside the spot-check subpaths `d2/game1/` and `doto/game1/`. M4.2's harness walks the whole tree; un-promoted files stay flat at the root.

A file only graduates from the flat root into `d2/game1/` or `doto/game1/` when it becomes a spot-checked golden in `goldenFileNames` ([internal/scan/scan_test.go:23](../../internal/scan/scan_test.go#L23)) — i.e. when we want byte-level scanner output pinned. Until then, the flat root is the right home: large, content-only.

## What M4.2 inherits from this doc

- The shape taxonomy + counts above seed the per-shape grammar tickets (Shapes 1, 2+5, 3 in scope; Shape 4 a no-op walker).
- The dispatch decision is the contract for the new `internal/parse/` routing layer.
- The allowlist inventory above is M4.2's verbatim starting allowlist; the harness fails when an unallowlisted code fires or when an allowlisted code's count grows.
