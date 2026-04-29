# Cloudflare deploy — handover

Status as of post-T12 deploy (2026-04-28). The Cloudflare push has happened and
the playground is live; § 2 below is retained as a runbook for re-running or
reproducing the deploy. The five sanity checks at the bottom of § 2 still need
to be ticked off before T12 itself can close — see the [T12 ticket](kanban/todo/v2/T12-deploy.md).

## 1. Expectations

### Architecture (post-T25 decision)

```
                  ┌──────────────┐
  browser ──────► │ Pages (web/) │  static SPA, served from Cloudflare edge
                  └──────┬───────┘
                         │  fetch(VITE_API_URL + /lint)
                         ▼
                  ┌──────────────┐
                  │  Worker      │  voidslice-api
                  │  + WASM mod  │  worker/voidslice.wasm (Go js/wasm)
                  └──────────────┘
```

L4 collapsed into L3. No Cloudflare Containers, no Container Registry, no
fly.io fallback. Dockerfile / docker-compose stay in the repo for local dev.

### What's in the repo and ready to deploy

| Path | Role |
|------|------|
| `cmd/voidslice-wasm/main.go` | `syscall/js` entry — exports `voidsliceLint(filename, src)` |
| `worker/index.js` | Worker fetch handler, `/health` + `/lint`, CORS, multipart + octet-stream |
| `worker/build.sh` | Compiles WASM and copies `wasm_exec.js` from the active Go toolchain |
| `wrangler.toml` | Worker name `voidslice-api`, prod env stub (the `.wasm` is imported directly in `worker/index.js`, not bound via config) |
| `.github/workflows/ci.yml` | test / lint / build / deploy (deploy main-only) |

### Numbers you should know

- **Worker bundle**: 3.0 MB raw, **865 KB gzipped**. CI fails if gzip exceeds 3 MB.
- **Memory worst-case**: 137 MB peak RSS on a pathological 1 MB `.decl` (over the
  128 MB isolate ceiling). Mitigation today is the **1 MiB body cap** in
  `internal/server` and the Worker; the proper fix (diagnostic-count cap) is
  unfilled. See § 3.
- **CPU**: ~5 ms native on typical inputs, ~2 s native worst-case under the body
  cap. js/wasm is 2–5× slower → safe under Workers Paid 30 s budget.
- **Cost floor**: **Workers Paid plan ($5 / month)** is required.
  Free-tier 10 ms CPU is conclusively unviable (T26 confirmed).

### What CI does today

- Push to any branch → `test` (`go vet` + `go test`), `lint` (golangci-lint),
  `build` (CLI + WASM + frontend, with the gz-3 MB assertion).
- Push to **`main`** → all of the above, plus `deploy`:
  - `wrangler deploy --env production` (Worker + WASM)
  - `wrangler pages deploy web/dist --project-name=voidslice --branch=main`

Deploy job will fail until the secrets/vars in § 2 are set.

## 2. Next steps (T12 — your Cloudflare account)

### Pre-flight

1. **Upgrade to Workers Paid** in the Cloudflare dashboard. Without this, every
   `/lint` request after the first 10 ms gets killed.
2. **Create the Pages project** named `voidslice` (matches the `--project-name`
   flag in the workflow). Use "Direct Upload" or "Git" — the workflow uses
   direct upload via Wrangler so either works.

### Secrets and vars (GitHub → Settings → Secrets and variables → Actions)

| Kind | Name | Value | Scope |
|------|------|-------|-------|
| Secret | `CLOUDFLARE_API_TOKEN` | API token with **Workers Scripts:Edit** + **Pages:Edit** + **Account: Read** | required |
| Secret | `CLOUDFLARE_ACCOUNT_ID` | Cloudflare account ID (dashboard URL) | required |
| Variable | `VITE_API_URL` | The deployed Worker URL (see below) | required for prod build |

Token creation: Cloudflare dashboard → My Profile → API Tokens → "Create Token"
→ "Custom token". Don't reuse the global API key.

### Deploy sequence

1. Set the three secrets/vars above.
2. Push to `main`. The workflow runs end-to-end. First deploy will create the
   Worker at `https://voidslice-api-production.<account>.workers.dev` (the
   `--env production` flag picks the `[env.production]` block in
   `wrangler.toml`).
3. Take that URL, set it as the GitHub `VITE_API_URL` variable, push again.
   Now the Pages build embeds the right backend URL.
4. Visit the Pages URL (`https://voidslice.pages.dev` or the auto-assigned
   branch URL). Drag in `testdata/broken/count-mismatch.decl`. Expect the
   `VALIDATE_ARRAY_COUNT_MISMATCH` warning to render.

### Custom domain (optional but listed in T12)

1. Cloudflare Pages → project → Custom domains → add e.g. `void-slice.dev`.
2. Cloudflare DNS → add a `CNAME api → voidslice-api-production…workers.dev`
   (or use a route `api.void-slice.dev/*` mapped to the Worker — both work,
   the route approach is more idiomatic).
3. Update `VITE_API_URL` to `https://api.void-slice.dev` and rebuild Pages.
4. Update `wrangler.toml` `[env.production].vars.ALLOWED_ORIGIN` from `"*"` to
   `https://void-slice.dev` once the custom domain is live. **Do this
   intentionally** — too-early lockdown breaks the playground's preview
   deployments.

### Sanity checks (from T12 ticket)

- Fresh browser, no cache → playground loads.
- Drag in a broken `.decl` → diagnostics in <1 s.
- Incognito window → no auth dependency.
- Phone on mobile data → CORS works, layout reflows.
- Cloudflare dashboard → Workers Analytics shows zero errors over 15 min.

## 3. Missed gaps

These are deliberate or known-unfilled. Each one is a real risk if ignored.

### Hard

- **Diagnostic-count cap is not implemented.** A 1 MB pathological `.decl` will
  still produce ~115K diagnostics under the current code, blow past 128 MB
  Worker memory, and fail the request. The 1 MiB body cap is a partial
  mitigation but doesn't bound the amplification. **T26 recommended filing
  this as a follow-up ticket, but no ticket was actually opened — the T27
  number was reused for LSP integration tests.** This remains an unfilled gap
  and should be filed before opening the Pages URL to wider public traffic.
  Suggested implementation: add a counter inside `parse.WalkEntities`'s
  diag-emit path; at 1,000 diagnostics, append one final
  `LINT_DIAGNOSTIC_LIMIT` warning and stop.

- ~~**No rate limiting.**~~ **Closed.** A `ratelimit` binding (`RATE_LIMITER`,
  30 req / 60 s, keyed on `cf-connecting-ip`) is declared in
  [wrangler.toml](wrangler.toml) and consumed at the top of `handleLint` in
  [worker/index.js](worker/index.js); over-budget requests get a 429 before
  the body is read or the WASM is invoked. `/health` is intentionally
  uncapped so smoke checks aren't throttled. If abuse patterns shift to
  many-IP / low-rate, layer Cloudflare's dashboard Rate Limiting Rules or Bot
  Fight Mode on top — they run at the edge before the Worker.

- **Workers logs are ephemeral.** `console.log` in `worker/index.js` goes to
  Workers Logs (24h retention by default on Paid). No log forwarding to
  external sinks (Datadog, Logflare, R2). If you want post-mortem visibility
  past a day, configure a Workers Logs export.

### Soft

- **No post-deploy smoke from CI.** The workflow deploys then exits. Adding a
  step that `curl`s the Worker `/health` and a known-broken `/lint` would
  catch obviously broken deploys before users hit them.

- **No Pages `_headers` file.** No CSP, no `X-Content-Type-Options`, no
  `Referrer-Policy`. Defaults are fine for a playground but a 5-line
  `web/public/_headers` would tighten this for free.

- **No cache layer.** Every `/lint` runs the WASM end to end. The same input
  always produces the same output, so a Cache API key on `(filename, sha256(src))`
  could shave repeat-request latency. Probably not worth it until traffic
  warrants — but worth noting.

- **`compatibility_date = 2025-09-01`** in `wrangler.toml` is recent but not
  pinned to anything specific. Bump it intentionally on a known-good day; don't
  let it drift.

- **TinyGo bundle path not explored.** TinyGo would likely cut the WASM from
  865 KB gz to ~50–150 KB gz, but requires a code audit (TinyGo's `reflect`
  and `encoding/json` subsets are limited). T25 deliberately deferred this —
  reopen if bundle size becomes a constraint.

- **No staging environment.** `[env.production]` exists in `wrangler.toml` but
  nothing else. A `[env.staging]` block + a `staging` branch trigger in the
  workflow would let you dry-run deploys. T12's "dry-run requirement" bullet
  is currently satisfied only by the `if: github.ref == 'refs/heads/main'`
  guard — that's "feature branches don't deploy", not a true staging.

- **No automated rollback.** `wrangler rollback` exists but isn't wired into
  the workflow. If a deploy ships a regression, you're hand-rolling-back from
  the Cloudflare dashboard.

- **`web/public/samples/*.decl` are static copies of `testdata/broken/*.decl`.**
  Drift is unlikely (3 small fixtures) but possible. T9 noted this; not worth
  a sync mechanism yet.

### Documentation drift to fix in T13

- `README.md` still describes v1 (CLI only). T13 will update it.
- `architecture.md` predates the Worker decision. T13 will rewrite it with the
  three-transport-layers diagram and the LSP roadmap.

## TL;DR

Push to `main` after setting three secrets/vars, upgrade to Workers Paid, and
the playground goes live. Add a diagnostics cap (T27) before publicizing the
URL; everything else in § 3 is "should fix when traffic warrants" rather than
"will break on day one".
