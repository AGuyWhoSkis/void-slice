# Linter-hardening — context for `/goal-define M12`

Pre-goal context. Self-contained: feed to `/goal-define M12` on a fresh branch without needing the M11 branch in scope. The taxonomy and recommendation below were derived in M11's idTech-reach memo §5 while answering an adjacent question (idParser as reference oracle); this doc lifts the M12-relevant outputs into a durable handoff.

## Bug class and axiom

**Axiom:** *a diagnostic should never be wrong.*

**Motivating bug:** observed cascade — an unterminated quote produces diagnostics on lines that are valid once the quote is closed. The same downstream lines are clean in the unmutated file and flagged once a quote breaks upstream. Repro sample on hand; commit under [`testdata/`](../../testdata/) in M12's first ticket.

## Methodology gap

Golden-TDD as practiced gives only a *partial* soundness check — zero diagnostics on known-valid files. It is silent on:

1. **Soundness under mutation** — the cascade. A diagnostic that only appears after upstream corruption is exactly what golden-TDD doesn't test.
2. **Completeness, either direction** — missing real errors, or surplus rules that should exist.

M12 is a methodology fix, not just a bug fix.

## Oracle taxonomy

- **Soundness oracles** answer *does the linter contradict itself on overlapping inputs?* The cascade is canonical: same line, two verdicts depending on upstream state. No external authority needed — fault injection on a known-valid file plus a locality property ("diagnostics confined to the mutated span") detects it directly.
- **Completeness oracles** answer *does the linter miss real Void Engine errors?* Require external authority — Void Engine itself, an independent parser, community reverse-engineering. Slower and more expensive per query.

**M12 is soundness-only.** The axiom is a soundness axiom; the cascade is a soundness violation. Completeness belongs in a separate future goal.

## Resource map (soundness-relevant)

| Resource | Cost | Notes |
|----------|------|-------|
| Golden corpora ([`testdata/golden/`](../../testdata/golden/)) + fault injection + locality property | Cheap | Mutate a known-good file in a known span; assert diagnostics confined to that span. Directly catches the cascade. No language boundary. |
| Go's `testing.F` fuzzer | Free | Native, no deps. Layered under property tests for crash/hang coverage. |
| Diagnostic codes as locality hypotheses | Cheap | Each code claims "violate me, see this and only this." Directly mutation-testable per code. |
| Past-bug log | Free | Every reported false positive seeds a regression case. |
| Cross-corpus differential (D2 / DOTO / Deathloop) | Cheap | Pattern present in one corpus but flagged in another = lint or valid-set is wrong. Borderline for M12 — defer unless investigation surfaces cross-corpus cascades. |

Out of scope as oracles: idLexer/idParser, Void Engine via Void Explorer, reverse-engineered tooling and wikis. These are completeness oracles. M11 §3 documented the C++/Go wrapper cost and parse-layer drift that limit idParser to a token-shape oracle — not enough return on integration cost to address the bug class on the table.

## Candidates and recommendation

Three candidates were on the table:

1. **Property-based fuzzing on mutated golden files with a locality property.**
2. **Differential against idParser or a second implementation.**
3. **Mutation testing of the validator itself.**

**Recommendation: (1).** It operationalizes the axiom directly, needs no external authority, stays inside Go, and matches the cascade's failure mode. (2) is a completeness oracle in disguise and belongs in a separate future goal hunting missing diagnostics, where its C++/Go wrapper cost would have a return. (3) answers an adjacent question — "does the validator code actually exercise its checks?" — distinct from "are the checks themselves sound."

M12's investigation ticket may revisit this pick if corpus characterization surfaces something this doc didn't see — but the burden of proof is on diverging, not on adopting.

## Suggested first ticket (investigation-only)

- **Commit the cascade repro fixture** under [`testdata/`](../../testdata/). Cheap, durable, seeds every subsequent ticket.
- **Characterize the cascade's scope** across the corpus. How many failure shapes look like this — unterminated quote, brace, block comment, other? Per-shape mutation experiments on golden files.
- **Stress-test the technique pick** against the evidence. Carry (1) forward by default; diverge only if findings demand it.

Subsequent tickets implement the harness and fixes for whatever cascades the investigation surfaces.

## Out of scope for M12

- Inferring intent.
- Completeness oracles (idParser differential, Void Engine round-trips, missing-diagnostic hunting) — future goal.
- Cross-corpus differential (D2 / DOTO / Deathloop) unless investigation surfaces cross-corpus cascades.
