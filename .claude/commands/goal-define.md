Define a new project goal. Goals are durable artifacts that capture *why a body of work exists* — distinct from tickets, which are disposable execution units. A goal is rare (a handful per project), articulates intent and boundaries, and lives at `kanban/goals/M{N}.md`.

Do **not** treat this as a template fill-in. The value of this command is the conversation that pressure-tests the goal *before* it gets written. A goal file is only as useful as the thinking behind it.

---

## Conversation flow

Walk through the elements below with the user. Don't ask all at once — go one at a time, react to each answer, and push back when an answer is thin. Goals that get written without this pushback tend to be vague and unactionable later.

### 1. Why does this goal exist?

Ask: what problem or opportunity drives this? What does "this goal succeeded" look like, *concretely*?

Listen for:
- A clear underlying need (a user pain, an internal-bloat problem, a missing capability, a discovery that changed the landscape).
- A success condition that's recognizable from the outside, not just a vague aspiration.

If the answer is generic ("clean up the codebase," "improve performance") push back: against what? measured how? for whom?

### 2. What's the scope — and just as important, what's *out*?

Ask: which systems, packages, or surfaces does this touch? What adjacent work is explicitly *not* in this goal?

Listen for:
- Concrete paths or subsystems (e.g. `internal/scan/`, the LSP server, the kanban tree).
- Explicit exclusions. Look at how M1–M5 handle this — each goal calls out adjacent goals it deliberately doesn't overlap with.

If the user just lists features they want to build, that's not scope, that's a wishlist. Push for the boundary: "what's a reasonable-sounding piece of work that you'd reject as 'no, that belongs to a different goal'?"

### 3. Initial status?

Goals don't sit in `todo`. Valid initial values:
- `active` — work is starting now or imminent.
- `parked` — captured as a real goal, but waiting on something (another goal, real-world feedback, a decision).
- `ongoing` — perpetual / no defined finish line (e.g. M5 dev tooling).

Ask which fits. If `parked`, ask what the gating condition is so the status line can name it.

### 4. Seed tickets?

Ask: are there tickets already in mind? Or does the goal exist primarily as a container that'll grow?

The answer doesn't change the goal file — goal files do not list tickets (the kanban folders are the source of truth). It only shapes whether the user's next step is `/goal-slice` (break the goal into a minimal ticket set) or just sitting on the goal until it's ready.

Do **not** create tickets as part of this command. Direct the user to `/goal-slice <goal-id>` after the goal is written, when they're ready to break it down.

---

## Drafting the goal file

Once the conversation has yielded enough material, draft the file *in chat* for the user to review and iterate on. Do not write to disk until the user confirms.

Goal file structure (assemble from the conversation, don't fill in a template):

```markdown
# M{N} — <short name>

<Paragraph from §1: the why and the success condition. Should read like a brief from someone who deeply understands the project, not like a feature list.>

**Scope:** <Paragraph from §2: in-scope surfaces and explicit exclusions, with references to adjacent goals where relevant.>

**Status:** <value from §3, with gating condition if parked>
```

Match the tone of M1–M5 — terse, factual, confident about boundaries. Avoid hedging language ("might," "perhaps") in the why and scope; if the goal is uncertain enough that hedging feels right, the answer is to keep working through §1 and §2, not to encode the uncertainty into the file.

---

## Mechanical steps (after user confirms the draft)

1. Pick the next `M{N}` by listing `kanban/goals/`. The highest existing N plus 1.
2. Write the file at `kanban/goals/M{N}.md`.
3. Add a row to the Goals table in `kanban/README.md`:
   ```
   | [M{N} — <short name>](goals/M{N}.md) | `goals/M{N}.md` |
   ```
   Insert in numerical order.
4. Confirm to the user where the file landed and remind them: seed tickets, if any, get created via `/crud-ticket`.

---

## Out of scope for this command

- **Goal closure.** Handled by the lifecycle in [kanban/README.md § Lifecycle](../../kanban/README.md): flip the goal file's `**Status:**` to `done.` and optionally append a retro (see [goals/M2.md](../../kanban/goals/M2.md), [goals/M6.md](../../kanban/goals/M6.md) as exemplars). Do not bundle that flow in here.
- **Status updates after creation.** The user edits `**Status:**` directly; the kanban-move hook excludes `goals/`, so there's nothing to automate.
- **Goal deletion.** Rare and case-by-case; not worth a command.
- **Creating the seed tickets themselves.** That's `/crud-ticket`'s job. Keep this command focused on the goal artifact.
