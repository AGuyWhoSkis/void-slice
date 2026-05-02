// Shared fixtures for the harnesses in this directory.
//
// Each fixture's `path` is the string passed to BOTH the WASM export (as the
// `filename` argument) and, for `reference: "cli"` cases, to the native CLI
// (as argv). Same string → same `file` field in JSON, so cross-layer
// comparisons stay meaningful.

export const FIXTURES = [
  {
    name: "clean",
    path: "testdata/corpus-mini/d2/game1/generated.decls.gamelogicmanager.ui.gamelogic.manager..gamelogicmanager.decl",
    reference: "cli",
  },
  {
    name: "validate-warning",
    path: "testdata/broken/count-mismatch.decl",
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
