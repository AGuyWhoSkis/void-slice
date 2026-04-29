# T26 · Linter Resource Profile

**Status:** done
**Version:** v2
**Size:** small

## What

Benchmark the linter's peak memory and wall-clock time on representative files from the `void-files/` corpus. These numbers determine whether the linter fits within Cloudflare Workers isolate limits (CPU cap, memory ceiling) or requires a heavier runtime, and inform cache-key design and upload size caps.

## Scope

- Run the linter against a representative sample of files from `void-files/` (small, medium, and largest available files).
- Record: peak RSS, wall-clock time, and input file size for each sample.
- Identify the largest file in the corpus and its lint time — this is the worst-case sizing input.
- Compare peak memory against the Cloudflare Workers memory ceiling (verify current value in docs; do not rely on training data).
- Compare wall-clock against the Workers per-request CPU cap.
- Recommend a maximum upload size cap based on the worst-case profile.
- Record results in `## Completion` for the v2 deployment ticket (T12).

## Dependencies

T7 (integration tests — confirms corpus is accessible and lint runs correctly end-to-end)

## Verification

A table of `{file, size_bytes, wall_clock_ms, peak_rss_mb}` for at least 5 representative corpus files, plus a written recommendation on whether the linter fits within Workers limits.

## Completion

**Top line:** the linter fits inside Workers' 128 MB / 30 s budget *only* if (a) the upload cap is tightened to **1 MiB** and (b) the diagnostic emit list is **capped at ~1,000 entries**. Without those, a pathological 1 MB `.decl` peaks at 137 MB RSS (over the isolate ceiling) and produces 115K diagnostics that bloat both the JSON response and the heap.

**Methodology**

- Native binary (`/tmp/voidslice lint --json`), best-of-5 wall via `time.perf_counter_ns`, peak RSS via `/usr/bin/time -f '%M'` (per-child watermark, not cumulative).
- Selected samples span 6 orders of magnitude on size: 2 B → 41 MB. Where possible, picked files with both clean and pathological structure.
- Native numbers map roughly 2–5× slower under `js/wasm` based on common Go-WASM ratios; budgets below assume the high end.

**Results**

| size (B) | wall (ms, native) | peak RSS (MB) | diagnostics | file |
|---:|---:|---:|---:|---|
| 2 | 4.4 | 6.8 | 2 | `…cpntwhaleoildevice…default.cpntwhaleoildevice.decl` (effectively empty) |
| 1,024 | 4.4 | 7.0 | 160 | `…ramseynote.decl` (parser falls through, lots of token errors) |
| 10,241 | 7.1 | 8.9 | 1,325 | `…midnightscene…fa.checkpoint02…midnightscene.decl` |
| 102,402 | 58.7 | 21.8 | 12,818 | `…dial_readytoleave…midnightscene.decl` |
| **1,066,470** | **2,064** | **136.9** | **115,534** | **`…heart_default.rulehandler.decl` (worst-case)** |
| 5,337,932 | 100.7 | 89.8 | 7 | `maps…boat.edge_of_the_world.boat_eotw_p.entities` (clean) |
| 14,995,138 | 222.8 | 165.1 | 11 | `maps…palace.palace_p_lowchaos.entities` (clean, but RSS over ceiling) |
| 41,803,509 | 735.8 | 597.0 | 261 | `maps…madinventor_p.entities` (largest in corpus) |

**Two distinct failure shapes**

1. **Diagnostic blowup on pathological `.decl`s.** The 1 MB `rulehandler/heart_default.decl` doesn't have the expected `Version N \n component { … }` outer shape — it starts with a bare `{ edit = { … } }`. The parser doesn't recover, so every token in the body becomes one `PARSE_UNEXPECTED_TOKEN` (115,534 of them). RSS scales with `len(diagnostics) × ~1 KB per diagnostic` (struct + message + JSON-encoded line+col strings). At 1 MB input this peaks at 137 MB — past the Worker ceiling. **This is the binding constraint, and it's input-content-driven, not input-size-driven.** A 100 KB pathological file already produces 12.8K diagnostics and 22 MB RSS — a ~210× amplification.
2. **Linear scaling on clean `.entities` maps.** The 41 MB map needs 597 MB RSS but only because the linter materializes the full token slice in memory (`scan.Scan` returns `[]Token`). For these files diagnostic count stays low (single digits), so the RSS is dominated by the token slice, not the diagnostics list. These files are far over any realistic upload cap and aren't worth optimizing for in v2.

**Cloudflare Workers fit**

Verified against current docs (already captured in T25): 128 MB memory, 10 ms CPU on Free / 30 s default on Paid, 3 MB compressed bundle on Free / 10 MB on Paid.

| Constraint | Linter under proposed caps |
|---|---|
| Memory ≤ 128 MB | OK if (a) body ≤ 1 MiB and (b) diagnostics ≤ 1,000 — both required |
| CPU ≤ 30 s (Paid) | OK; worst case under 1 MiB cap is ~2 s native → ~10 s wasm. Comfortable |
| CPU ≤ 10 ms (Free) | **Not viable.** Even a 10 KB clean file is 7 ms native → ~30 ms wasm. Free tier is conclusively off the table |
| Bundle ≤ 3 MB compressed | OK; 2.75 MB gzip (from T25) |

**Recommended changes for T11 deploy**

1. **Lower `internal/server.defaultMaxBodyBytes` from `5 << 20` to `1 << 20`.** One-line change. The original 5 MiB was provisional and explicitly flagged "T26 may adjust" in T8. Browser-side, the playground only paste/drops single `.decl`/`.entities` files; the largest realistic `.decl` (component or rulehandler kind) is ~1 MB. The huge `.entities` maps aren't a sensible playground input — they're 5–40 MB and will produce a flat "your file looks fine, here are 3 warnings" response that nobody pasted them in to see.
2. **Cap diagnostics emit at 1,000 entries.** Add a counter inside the `parse.Handler` adapter the validator uses; once it hits the cap, emit a final `LINT_DIAGNOSTIC_LIMIT` warning ("further diagnostics suppressed; fix earlier errors first") and stop appending. This is the actual defense — body-size alone doesn't bound it. **Implement in `internal/lint`**, so the CLI gets the same protection. Suggest a separate small ticket (`T27` cap-diagnostics?) so this doesn't bloat T11.
3. **Stay on Workers Paid plan.** Free 10 ms CPU is unviable; Paid 30 s default is comfortable. The $5/mo is the v2 cost floor.

**Out of scope (for follow-ups)**

- Streaming the token slice through the validator instead of materializing it. Would cut RSS for huge `.entities` files. Not needed for v2 since those files are over the upload cap.
- Pretty-printing or paginating the response when the diagnostic cap fires. T13 may want to add a "showing N of M" sidebar note in the playground.
- Fixing the parser to recover at the top-level on bad-shape `.decl`s. That's a v1-engine quality issue and a separate ticket.

**What this unblocks**

- **T11:** body cap + diagnostics cap noted as deploy prerequisites. Bundle and budget numbers are concrete.
- **T12:** "is the linter Workers-shaped?" answered: yes, with the two caps above. Paid plan required.
