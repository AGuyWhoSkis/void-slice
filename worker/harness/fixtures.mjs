// Shared fixtures for the harnesses in this directory.
//
// Each fixture's `path` is the string passed to BOTH the WASM export (as the
// `filename` argument) and, for `reference: "cli"` cases, to the native CLI
// (as argv). Same string → same `file` field in JSON, so cross-layer
// comparisons stay meaningful.
//
// Optional `layers` restricts which oracle layers a fixture runs through.
// Default is all four (`native`, `wasm`, `worker`, `frontend`). The corpus
// `.entities` fixture exceeds the worker's 1 MiB body cap, so it sets
// `["native","wasm"]`; the cap is a real prod constraint, not test-tweakable.

export const FIXTURES = [
  {
    name: "clean",
    path: "testdata/golden/d2/game1/generated.decls.gamelogicmanager.ui.gamelogic.manager..gamelogicmanager.decl",
    reference: "cli",
  },
  {
    name: "validate-warning",
    path: "testdata/broken/index-oob.decl",
    reference: "cli",
  },
  {
    name: "scan-error",
    path: "testdata/broken/missing-semicolon.decl",
    reference: "cli",
  },
  {
    name: "empty-input",
    path: "empty.decl",
    inlineSrc: "",
    reference: { file: "empty.decl", diagnostics: [] },
  },
  // M4.12 corpus parity — one representative per in-scope `.decl` shape (1, 2,
  // 3, 5, sidecar) plus the `.entities` golden. Shape 1 is already covered by
  // the `clean` fixture above. Paths cite kanban/goals/M4-decl-taxonomy.md.
  {
    name: "corpus-shape2-animset",
    path: "testdata/golden/generated.decls.animset.models.weapons.timeshift_device.timeshift..animset.decl",
    reference: "cli",
  },
  {
    name: "corpus-shape3-material",
    path: "testdata/golden/generated.decls.material.models.environment.voidhouse.rock_set.void_rock_01..material.decl",
    reference: "cli",
  },
  {
    name: "corpus-shape5-md6def",
    path: "testdata/golden/doto/game1/generated.decls.md6def.models.characters.small.civ_middle.dockers.docker_01.docker_small_01_head..md6.decl",
    reference: "cli",
  },
  {
    name: "corpus-sidecar",
    path: "testdata/golden/generated.decls.animbasic.models.characters.dlc01.player.billie.additives_body..animbasic.decl.xml",
    reference: "cli",
  },
  {
    name: "corpus-entities",
    path: "testdata/golden/d2/game1/maps.campaign.dunwall.escape.tower.dunwall_escape_tower_p.entities",
    reference: "cli",
    layers: ["native", "wasm"],
  },
];

export const WRONG_ARG_EXPECTED = {
  file: "",
  diagnostics: [
    {
      line: 0,
      col: 0,
      severity: "error",
      code: "WORKER_HARNESS",
      message: "voidsliceLint requires (filename, src)",
    },
  ],
};
