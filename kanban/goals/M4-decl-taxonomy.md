# M4 — `.decl` format taxonomy & dispatch decision

The `.decl` extension covers five distinct top-level grammars, plus an XML sidecar variant. This doc is the inventory + dispatch decision that future M4 grammar tickets cite. It is the deliverable of M4.1.

All counts were derived by content-sniff (not filename) over `testdata/corpus-mini/{intake,d2/game1,doto/game1}/`. Universe: 57 `.decl` + 12 `.decl.xml` + 1 `.entities`.

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

Curly-braced root, `=` and `;` separated, identifier keys, scalar/quoted/object values. Sub-types include `arktree`, `cpntplayer*`, `entitydef`, `fx`, `kiscule`, `mapinfo`, `midnightscene`, `particledt`, `physicsmaterial`, `rulehandler`, `localized.*.speechbarks`, `localized.*.speechscene`. Representative: [d2/game1/generated.decls.cpntplayerfxmanager.…decl](../../testdata/corpus-mini/d2/game1/generated.decls.cpntplayerfxmanager.components.characters.player.base.fx_manager..cpntplayerfxmanager.decl).

```
{
    inherit = "components/characters/player/base/powers/playerpowersystem.cpntplayerpowersystem";
    edit = {
        m_someField = "value";
    }
}
```

### Shape 2 — animset (7 files)

Curly-braced root, **no** `=`, **no** `;`, whitespace-separated. Top-level `previewmd6 "…"` and `skeleton "…"`, nested `groups { "…" }`, `aliases { group N alias { name "…" … } }`. Representative: [intake/generated.decls.animset.…timeshift..animset.decl](../../testdata/corpus-mini/intake/generated.decls.animset.models.weapons.timeshift_device.timeshift..animset.decl).

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

Curly-braced root, no `=`, no `;`, tab-aligned key-value pairs. `m_PhysicsMaterial "…"` at top, then `version N`, `state { }`, `options { … }`. Representative: [intake/generated.decls.material.…void_rock_01..material.decl](../../testdata/corpus-mini/intake/generated.decls.material.models.environment.voidhouse.rock_set.void_rock_01..material.decl).

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

Curly-braced root, `newstyle` token, `state { … }`, `hlsl_prefix { … }` containing HLSL preprocessor directives + shader source. Representative: [intake/generated.decls.renderprog.arksssblur.decl](../../testdata/corpus-mini/intake/generated.decls.renderprog.arksssblur.decl). Counted here so dispatch routes them to a no-op handler; grammar work belongs to a future G-C goal.

### Shape 5 — md6def (1 file)

**Not in M4.1's pre-classification — discovered during content-sniff.** Curly-braced root containing a top-level `init { … }` block. Whitespace-separated, quoted-string values, integer literals (e.g. `lod 0 { mesh "…" }`). Lexically near-identical to Shape 2; the Shape 2 grammar ticket should fold this in rather than carve a separate ticket. Representative: [doto/game1/generated.decls.md6def.…docker_small_01_head..md6.decl](../../testdata/corpus-mini/doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl).

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

Single self-closing XML element `<ResourceInfo name="…" classname="…" … />`. Void Explorer export/import metadata, not a game artifact. Lint contract is "validate the XML well-formedness and emit nothing else." Representative: [intake/generated.decls.animbasic.…additives_body..animbasic.decl.xml](../../testdata/corpus-mini/intake/generated.decls.animbasic.models.characters.dlc01.player.billie.additives_body..animbasic.decl.xml).

## Dispatch decision: hybrid (extension + content-sniff)

**Rule.** Extension picks XML vs. text; content-sniff picks among shapes 1–5.

- `.decl.xml` → XML handler (extension-based; no ambiguity).
- `.decl` → tokenize the first non-whitespace token after the leading `{`:
  - `previewmd6` → Shape 2
  - `m_PhysicsMaterial` → Shape 3
  - `newstyle` → Shape 4 (route to no-op handler)
  - `init` → Shape 5 (route to Shape 2 grammar)
  - otherwise → Shape 1

**Why hybrid, not filename-infix.** The existing `corpus-mini/doto/game1/generated.decls.physicsmaterial.contactsystem.weapons.decl` carries `physicsmaterial`/`material` in its filename but is structurally Shape 1 (`{ inherit = "…"; edit = { … } }`), not Shape 3. Filename-infix dispatch would mis-route it. Content-sniff is robust against this kind of naming drift.

**Why not pure content-sniff.** XML and text grammars are reliably split by extension. Doing extension first keeps the sniff small (5 brace-leading tokens) and gives a clear error path when an XML file lands in a `.decl` slot or vice versa.

**Where dispatch lives.** `internal/parse/` (new file, e.g. `dispatch.go`); `WalkEntities` becomes the Shape-1 (curly inherit/edit) grammar after the dispatch layer is added, with new walkers for shapes 2/3/5 and a no-op walker for shape 4.

## Allowlist inventory for M4.2

Fresh as of 2026-05-02. Total: **754,265 diagnostics across 70 files.**

| Code | Count | Sev | Concentration |
|------|-------|-----|---------------|
| PARSE_UNEXPECTED_TOKEN | 751,470 | E | Shape 2: 628,623 · Shape 1: 120,990 · Shape 4: 1,213 · Shape 3: 502 · Sidecar: 120 · Shape 5: 22 |
| VOID_SCAN | 2,786 | E | Shape 4: 2,388 · Shape 3: 341 · Sidecar: 36 · Shape 2: 16 · Shape 1: 5 |
| PARSE_EXPECTED_SYMBOL | 329 | E | Shape 2: 306 · Shape 4: 23 |
| PARSE_EXPECTED_SEMICOLON | 1 | E | Shape 1 |
| VALIDATE_ARRAY_COUNT_MISMATCH | 7 | W | existing `.entities` |
| VALIDATE_ARRAY_MISSING_NUM | 1 | W | existing `.entities` |
| LINT_VE_INCONSISTENCY | 1 | W | existing `.entities` (correct behavior, *not* in allowlist) |

**M4.2's starting allowlist:**

- Allow `PARSE_UNEXPECTED_TOKEN`, `VOID_SCAN`, `PARSE_EXPECTED_SYMBOL`, `PARSE_EXPECTED_SEMICOLON` across `intake/` and `corpus-mini/{d2,doto}/game1/`. These drain as Shapes 1–3 and 5 grammars land. Shape 4 (renderprog) is permanently allowlisted per G-C carve-out.
- Allow `VALIDATE_ARRAY_COUNT_MISMATCH` (×7) and `VALIDATE_ARRAY_MISSING_NUM` (×1) on the existing `.entities` (separate Phase C work).
- `LINT_VE_INCONSISTENCY` (×1) is the Void-Explorer warning behaving correctly — it must **not** be in the allowlist.

**M4 closes when** the allowlist drains to (Shape-4-only PARSE/VOID + 8 entities-validate-warns), i.e. once the in-scope grammars stop emitting false positives.

## Game-subdir decision

`testdata/corpus-mini/intake/` is a **permanent third subpath** alongside `d2/game1/` and `doto/game1/`. M4.2's harness walks all three roots; files stay flat under `intake/`.

A file only graduates from `intake/` into `d2/game1/` or `doto/game1/` when it becomes a spot-checked golden in `goldenFileNames` ([internal/scan/scan_test.go:23](../../internal/scan/scan_test.go#L23)) — i.e. when we want byte-level scanner output pinned. Until then, `intake/` is the right home: large, flat, content-only.

The `intake/` name is misleading once it stops being a holding pen. Rename deferred to M4 close to avoid churn during active grammar work.

## What M4.2 inherits from this doc

- The shape taxonomy + counts above seed the per-shape grammar tickets (Shapes 1, 2+5, 3 in scope; Shape 4 a no-op walker).
- The dispatch decision is the contract for the new `internal/parse/` routing layer.
- The allowlist inventory above is M4.2's verbatim starting allowlist; the harness fails when an unallowlisted code fires or when an allowlisted code's count grows.
