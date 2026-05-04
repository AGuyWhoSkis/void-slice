# Void Slice

A linter for Dishonored 2 and Death of the Outsider game files. It spots structural problems in `.decl`, `.entities`, and `.entitydef` files before they break a level.

## Why this exists

[Void Explorer](https://www.nexusmods.com/dishonored2/mods/9) lets you Export, Import, and Generate Mods against the game's compiled id Tech files — but if your edit has a stray bracket, a missing semicolon, or a typo'd array index, you usually only find out after loading the level and watching it misbehave. Void Slice catches that class of mistake by reading the file and pointing at the line where it went wrong, so you can fix it before the game ever sees it.

## What it catches

A short tour, with the actual broken-file fixtures the test suite runs on:

- **Missing semicolon** — [missing-semicolon.decl](testdata/broken/missing-semicolon.decl)
- **Duplicate array index** — two `item[0]` entries in the same array — [dup-index.decl](testdata/broken/dup-index.decl)
- **Out-of-bounds array index** — `item[5]` in an array sized `num = 2` — [index-oob.decl](testdata/broken/index-oob.decl)
- **Identifier expected** — a number where a name should go — [expected-identifier.decl](testdata/broken/expected-identifier.decl)
- **Unterminated object** — a `{` block that never closes — [unterminated-object.decl](testdata/broken/unterminated-object.decl)
- **Mismatched brackets and quotes** — `[] {} ()` or `"" ''` that don't pair up
- **Binary file refusal** — `.bwm`, `.tome`, and other compiled formats are rejected with a clear message instead of garbled output
- **Round-trip warning for `.entities` / `.cfg`** — flags file types Void Explorer can't reliably re-import

Linting a whole mod folder at once and cross-checking D2 vs. DOTO are on the roadmap.

## How to use it

Three ways, pick whichever fits how you work:

- **Web playground** — paste, drop, or click a sample file in your browser. Nothing to install. Run from the [project source](https://github.com/AGuyWhoSkis/void-slice).
- **VS Code extension** — diagnostics appear as squiggles in the editor and in the Problems panel as you edit. See [voidslice-vscode/README.md](voidslice-vscode/README.md) for install steps.
- **Command line** — `voidslice lint path/to/file.decl` from a terminal, useful for scripting or batch checks.

## Contributing

You don't need to write code to help. Broken fixture files (real game files that surface a bug or false positive) and bug reports are just as valuable. See [CONTRIBUTING.md](CONTRIBUTING.md).

To talk shop or ask a question, find me on [Nexus Mods](https://www.nexusmods.com/profile/kleptobismal).
