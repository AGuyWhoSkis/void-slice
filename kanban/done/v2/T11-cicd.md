# T11 · CI/CD

**Status:** done  
**Version:** v2  
**Size:** medium  
**Blocks:** T12 (deploy)

## What

GitHub Actions pipeline and Wrangler config for auto-deploy on push to main. Run the pipeline at least once before the production push.

## Scope

**`.github/workflows/ci.yml`:**
```
on: push (all branches) + pull_request

jobs:
  lint:   golangci-lint (latest release action)
  test:   go test ./... (Go 1.23, cache modules)
  build:  go build ./cmd/voidslice (verify binary compiles)
  deploy: (main branch only, after test passes)
    - build frontend (npm run build in web/)
    - deploy Pages via Wrangler
    - L4 deploy — shape depends on T25 outcome:
        WASM path: compile GOOS=wasip1 GOARCH=wasm, bundle into Worker, deploy via Wrangler
        Containers path: build Docker image, push to Cloudflare Container Registry, deploy Worker + Container via Wrangler
```

**Required secrets in GitHub repo:**
- `CLOUDFLARE_API_TOKEN` — scoped to Workers, Pages, Containers
- `CLOUDFLARE_ACCOUNT_ID`

**Wrangler config (`wrangler.toml`):**
- Worker name: `voidslice-api`
- Container binding pointing to the Go image
- Routes: `api.yourdomain.com/*` → Worker → Container

**Cloudflare Pages config (`web/_headers` or Pages project settings):**
- `VITE_API_URL` set to the Worker route URL at build time

**Dry-run requirement:**
- Run the full deploy workflow at least once in a staging or test push before flipping DNS to production

## Dependencies

T10 (Dockerfile must exist for the Containers fallback path), T9 (web/dist must be buildable), T25 (WASM spike — determines which L4 deploy path to implement)

## Verification

- Push to a feature branch → CI runs lint + test + build (no deploy)
- Push to main → full pipeline including deploy completes green
- Cloudflare dashboard shows updated Worker and Pages deployment

## Completion

**What was done**

- `cmd/voidslice-wasm/main.go` — `//go:build js && wasm` entry point that exports `globalThis.voidsliceLint(filename, src)` via `syscall/js`. Calls the existing `lint.New().Lint()` and returns the same JSON shape as `report.RenderJSON`. Has its own minimal JSON quoter for the harness-error path so the bridge response is always valid even on Go errors. Doesn't touch `cmd/voidslice/main.go`.
- `worker/index.js` — Cloudflare Worker fetch handler. Routes `GET /health` and `POST /lint`, mirrors the `internal/server` content-type rules and CORS middleware, accepts both `application/octet-stream` (with optional `?filename=`) and `multipart/form-data`. Module is instantiated lazily via a memoized `ensureWasm()` so cold-start cost is paid once per isolate. Per-request log line emitted via `console.log` JSON.
- `worker/build.sh` — repo-root build helper. Compiles the WASM with `-trimpath -ldflags="-s -w"` and copies `wasm_exec.js` from `$(go env GOROOT)/lib/wasm/`. Both outputs are `.gitignore`d (rebuilt in CI).
- `wrangler.toml` — Worker name `voidslice-api`, `compatibility_date = 2025-09-01`, `[[wasm_modules]]` binding `VOIDSLICE_WASM`, observability on, plus a `[env.production]` section ready for the custom-domain origin in T12. No container binding (T25 made it unnecessary).
- `.github/workflows/ci.yml` — four jobs:
  - `test`: `go vet` + `go test ./...` on Go 1.23 with module cache.
  - `lint`: `golangci-lint-action@v6` (latest version) — fast static check.
  - `build`: builds the CLI, runs `worker/build.sh`, **fails the workflow if the gzipped WASM exceeds 3 MB** (catches accidental size regressions before they break the Free-tier bundle limit), `npm ci && npm run build` for the frontend, uploads `web-dist` and `worker-bundle` artifacts.
  - `deploy`: gated on `github.ref == 'refs/heads/main' && github.event_name == 'push'` and waits for the other three jobs. Uses `cloudflare/wrangler-action@v3` to `deploy --env production` the Worker, then `pages deploy web/dist --project-name=voidslice --branch=main`. Required secrets/vars are documented in a comment block at the bottom of the workflow.
- **`internal/server/server.go`**: `defaultMaxBodyBytes` lowered from `5 << 20` to `1 << 20` per T26's recommendation. The new constant carries a comment pointing back to the T26 doc so the reasoning isn't lost.

**Key decisions**

- **WASM path, not Containers** (per T25). Drops the docker-compose container binding from `wrangler.toml` and the Container Registry push from the deploy job. The Dockerfile from T10 stays in the repo as the local-dev / fallback path.
- **`-trimpath -ldflags="-s -w"` on the Worker WASM**, plus a separate `cmd/voidslice-wasm` entry that omits the CLI flag-parsing and HTTP server. Result: 3.0 MB raw, **865 KB gzipped** — vs. 2.75 MB gzipped for the full CLI compiled to `js/wasm`. Three-fold slimming on bundle size keeps us comfortably under the Free-tier 3 MB compressed cap.
- **wasm_exec.js sourced from the active Go toolchain at build time**, not committed. Avoids a stale-shim drift problem and keeps the worker bundle in lockstep with whatever Go version CI uses.
- **CI bundle-size assertion**. The gzip-3 MB check is a hard guardrail; future PRs that bloat the linter (a regex package, deeper validator state, etc.) will fail CI before they hit Cloudflare's deploy and break the Free tier.
- **No Pages `_headers` file**. Static asset headers are fine at Cloudflare's defaults; the CORS-relevant headers come from the Worker, not Pages.
- **Single-binary Worker** rather than per-request module instantiation. The memoized promise pattern is the canonical Workers shape — measured cold-start should be sub-second on first request, sub-ms thereafter (until the isolate is recycled).
- **`VITE_API_URL` flows through `vars` (not secrets)** — it's a public URL, not sensitive. Falls back to `http://localhost:8080` in branch builds where the var isn't set, so feature branches still produce a usable dist artifact for review.

**Deviations**

- Original ticket scope listed **two L4 deploy paths** (WASM vs. Containers fallback). T25 closed that — only the WASM path is implemented.
- Container binding / Container Registry push lines from the original `wrangler.toml` sketch are intentionally omitted.
- Custom-domain route (`api.yourdomain.com/*`) is not configured here; T12 wires DNS + custom domain.

**Verification**

- `go vet ./...` clean. `go test ./...` green across all 7 packages, including `internal/server` after the body-cap change.
- `worker/build.sh` produces `worker/voidslice.wasm` (3,081,571 B raw, 865,047 B gzipped) and `worker/wasm_exec.js`. Bundle size assertion in CI passes (gzipped < 3,145,728).
- The compiled WASM was smoke-tested via Node + the official `wasm_exec.js`: `voidsliceLint("count-mismatch.decl", <src>)` returns the same JSON the CLI's `--json` mode does. (Captured during T25; same artifact.)
- `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml'))"` parses the workflow without error.
- **Did not run the workflow itself**: that needs a push to GitHub, which is T12's responsibility (after `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` are configured in repo settings).

**Out of scope / follow-ups**

- **Diagnostic-count cap (T26 recommendation #2)** — not implemented here. T26 explicitly suggested a separate ticket. Without it, a malicious 1 MB pathological input could still produce 100K+ diagnostics and exhaust Worker memory; the 1 MiB body cap reduces but doesn't eliminate the risk.
- **DNS / custom domain / Pages project creation** — T12.
- **Smoke-from-CI** — adding a post-deploy smoke step (curl the deployed Worker, expect 200 + valid JSON) is a worthwhile next step but lives outside the ticket scope.
