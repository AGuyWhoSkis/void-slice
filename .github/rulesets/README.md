# Rulesets

GitHub branch rulesets, committed as JSON. Source of truth for what the GitHub UI should show — survives manual UI edits and makes the rules reviewable in PRs.

## Files

- `main.json` — protects the default branch. Requires 1 non-author approval, 5 green checks (`lint`, `build`, `test`, `harnesses`, `preview-deploy`), linear history. Blocks force-push and deletion. Repo Admin role (`actor_id: 5`) bypasses with `bypass_mode: always` so the solo maintainer can merge without a non-author approval; agents are blocked from merging via Claude Code hooks in `.claude/settings.json` (defense-in-depth, not GitHub-side).

## Apply

Rulesets don't auto-apply — push the JSON to GitHub manually after edits:

```sh
gh api -X POST repos/AGuyWhoSkis/void-slice/rulesets --input .github/rulesets/main.json
```

To update an existing ruleset, find its ID and `PUT` instead:

```sh
id=$(gh api repos/AGuyWhoSkis/void-slice/rulesets --jq '.[] | select(.name=="main") | .id')
gh api -X PUT repos/AGuyWhoSkis/void-slice/rulesets/"$id" --input .github/rulesets/main.json
```

## Verify

```sh
gh api repos/AGuyWhoSkis/void-slice/rulesets --jq '.[] | select(.name=="main")'
```

The active config should match `main.json`. If it drifts, re-apply.
