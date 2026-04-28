# T11 · CI/CD

**Status:** todo  
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
    - build Docker image
    - push to Cloudflare Container Registry
    - deploy Worker + Container via Wrangler
    - build frontend (npm run build in web/)
    - deploy Pages via Wrangler
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

T10 (Dockerfile must exist), T9 (web/dist must be buildable)

## Verification

- Push to a feature branch → CI runs lint + test + build (no deploy)
- Push to main → full pipeline including deploy completes green
- Cloudflare dashboard shows updated Worker and Pages deployment
