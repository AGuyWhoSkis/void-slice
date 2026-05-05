File a nit: a tiny single-edit chore captured during another goal's work, parked on the long-lived `nits` branch so it doesn't pollute the active goal's PR. `$ARGUMENTS` is the nit description.

Capture-only. Listing pending nits and adopting them are `/goal-define`'s and `/goal-slice`'s jobs.

---

## Conversation flow

1. **Description.** Use `$ARGUMENTS` verbatim as the nit description. If empty, ask the user once for a one-line description and stop if they don't provide one.

2. **Paths (required).** Ask: *"Best guess at the file or glob this nit touches?"* Required — refuse to file the nit if the user can't name one. The point is the Paths-subset check at adoption time; without a Paths field, the nit can't be adopted safely.

3. **Context (optional).** Ask: *"One paragraph of context — what's the rationale or pointer to the underlying issue? (Press enter to skip.)"* Skip silently if the user doesn't provide one.

4. **File it.** Invoke:

   ```
   tools/nit.sh --description "<description>" --paths "<glob>" [--context "<paragraph>"]
   ```

   The script handles syncing the nits worktree, writing the file, committing, and pushing. On success it prints the committed file path; relay it to the user. On failure (worktree missing, push failed after retries) relay the error verbatim — don't try to recover.

---

## What this command does NOT do

- **Auto-bootstrap.** If `tools/nit.sh` reports the worktree is missing, tell the user to run `tools/nits-bootstrap.sh` and stop. Don't run it for them — substrate setup is a deliberate user action.
- **Edit or delete existing nits.** Capture-only. To delete a nit, `git rm` it manually in `../void-slice-nits/` and push.
- **List pending nits.** Listing happens in `/goal-define` (read-only) and `/goal-slice` (with adoption).
- **Auto-infer Paths from `git diff` or open files.** Fragile and surfaces wrong defaults more often than not. Always ask explicitly.
