# T26 · Linter Resource Profile

**Status:** todo
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
