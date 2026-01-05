### What's the point?
Void Slice is a modding tool for Dishonored 2 and Dishonored DOTO that **reads game files for the purpose of linting, verifying code changes, or comparing D2 code to DOTO code**.

The objective of this repo is to **fill a gap in tooling for modders** of Dishonored 2 and Dishonored DOTO. While Void Explorer enables patch-based modding of code and game assets, no tooling exists yet for parsing, linting, verifying, or otherwise evaluating compiled id Tech code.

In other words, Void Explorer provides the incredibly useful Export/Import/Generate Mod functions, and Void Slice operates within Void Explorer's Import/Export folders to provide additional support during development.

### Features under consideration for this project
- Basic linting of a file, so modders don't have to rely on game crashes to infer the correct syntax
	- Check for parity in brackets `[] {} ()` and quotes `"" ''`
	- Check indexes and counts to prevent off-by-one issues in statements like `count = X;` or `num = 0;`
	- Check file references for validity, including the use of `NULL;`
- Void Explorer mod verification
	- Basic linting of all modified files
	- Flagging changes that are likely to cause crashes (.entities files, for example)
- D2 and DOTO
	- Dishonored DOTO is built 'on top' of Dishonored 2.. so what can we learn by comparing their files? (hint: a lot..)
	- Comparing D2 and DOTO code will reveal a lot about how DOTO was built, which is information that can be used both to enhance this repo and develop more sophisticated mods.

### Want to help?
If you made it this far, you might be interested in contributing. Please reach out on [Nexus Mods](https://www.nexusmods.com/profile/kleptobismal) where we can discuss how to make that happen!
