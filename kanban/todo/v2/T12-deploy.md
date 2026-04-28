# T12 · Production Deploy

**Status:** todo  
**Version:** v2  
**Size:** small  
**Blocks:** nothing (final v2 milestone)

## What

First production push to Cloudflare. Custom domain wired via Cloudflare DNS. Sanity-check from multiple surfaces.

## Scope

**Steps:**
1. Trigger the main branch CI/CD pipeline (T11) and confirm it deploys cleanly
2. In Cloudflare DNS: add a CNAME or A record pointing your custom domain to the Pages deployment
3. In Pages settings: add the custom domain; wait for TLS provisioning (usually <5 min)
4. Set `VITE_API_URL` in the Pages build config to the production Worker URL (e.g., `https://api.yourdomain.com`)

**Sanity checks (do all of these):**
- Open the URL in a fresh browser (no cache) — playground loads
- Drag in a broken `.decl` file — diagnostics appear within 1 second
- Test from an incognito window — confirm no auth or cookie dependency
- Test from your phone on mobile data — confirm CORS headers and mobile layout
- Check Cloudflare's built-in uptime monitoring is enabled for the domain

**L4 architecture (resolved by T25):**
- Primary: Workers/WASM — linter compiled to `wasip1/wasm`, runs inside the Worker (no container)
- First fallback: Cloudflare Containers — if WASM is fundamentally blocked; target archetype D (low wake fraction, high CPU-active-within-wake)
- Second fallback: `fly.io` — `fly launch` on the same Dockerfile, update the Worker to proxy there; keeps L1–L3 on Cloudflare

## Dependencies

T11 (CI/CD pipeline must run end-to-end successfully at least once), T25 (WASM spike — determines L4 architecture), T26 (resource profile — confirms linter fits within Workers limits and informs upload size cap)

## Verification

- URL is live and accessible from a fresh browser
- Linting a broken file produces visible diagnostics
- No console errors in browser devtools
- Cloudflare dashboard shows zero errors in the last 15 minutes
