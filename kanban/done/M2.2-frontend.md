# T9 · Frontend Playground (`web/`)

**Status:** done  
**Version:** v2  
**Size:** large  
**Blocks:** T12 (deploy)

## What

A single-page React app that lets a visitor paste or drag-drop a game file, calls `POST /lint` on the backend, and renders diagnostics as gutter markers and a sidebar list. Lives in `web/`.

## Scope

**Stack:** React + Vite + CodeMirror 6

**Core interaction:**
- Landing section: headline, one-paragraph explanation, "Try it" CTA that scrolls to playground
- Playground: drag-drop zone or paste area accepts a file; contents loaded into CodeMirror editor
- On file load → `POST /lint` to the backend → render diagnostics:
  - Red gutter markers on `Error`-severity lines
  - Amber gutter markers on `Warning`-severity lines
  - Sidebar list: line/col, code, message; clicking a diagnostic scrolls the editor to that line
- Three preloaded broken sample files (from T6 testdata) so a first-time visitor can click and immediately see diagnostics without uploading anything

**Aesthetic constraint:**
- Stripe-docs clean: whitespace-heavy, monospaced where it counts (editor + diagnostic codes), two neutrals (near-white bg, near-black text), error red, warning amber
- **No Awwwards drift** — resist adding animations, gradients, or decorative elements beyond what serves clarity

**API wiring:**
- Backend URL from a Vite env var (`VITE_API_URL`), defaulting to `http://localhost:8080` for local dev
- Handle: loading state during lint call, error state if backend is unreachable, empty state before first file

**Build output:**
- `web/dist/` — static assets deployable to Cloudflare Pages

## Dependencies

T8 (HTTP server must be running to test the full flow locally)

## Verification

```bash
cd web
npm install
npm run dev   # localhost:5173, backend on localhost:8080

# manual test checklist:
# - drag in testdata/broken/count-mismatch.decl → diagnostics appear
# - click preloaded sample → editor loads + diagnostics
# - click diagnostic in sidebar → editor scrolls to that line
# - drag in a clean .decl → no markers, sidebar empty

npm run build  # dist/ produced, no build errors
```

## Completion

**What was done**

- New `web/` directory: Vite + React 18 + TypeScript scaffold (`package.json`, `vite.config.ts`, `tsconfig.json`, `index.html`, `src/main.tsx`, `src/App.tsx`).
- CodeMirror 6 wired in `src/components/Editor.tsx` — `lineNumbers`, `history`, default keymap, and a custom diagnostic gutter that renders 4-px red bars on Error lines and amber bars on Warning lines. Diagnostics are collapsed per line (Error wins on ties). The gutter is swapped via a `Compartment`, so re-linting doesn't tear down the editor state.
- `Playground.tsx` owns the file → state → API loop:
  - Drag-drop zone with hover state, `<input type="file">` fallback, and three sample-chip buttons that load `count-mismatch`, `dup-index`, `missing-semicolon` fixtures (raw-imported from `web/src/samples/` via Vite's `?raw` loader).
  - Calls `POST /lint?filename=<name>` on file/sample load; nonce guards out-of-order responses.
  - Status row reports loading/clean/n-diagnostics/error inline above the editor.
- `DiagnosticsList.tsx` renders the sidebar; clicking an entry scrolls the editor to that line via a nonce-keyed `scrollToLine` prop.
- `api.ts` reads `VITE_API_URL` (default `http://localhost:8080`) and POSTs as `application/octet-stream`.
- `Landing.tsx` + `App.tsx` deliver the headline / tagline / "Try it →" CTA that anchors to `#playground`.
- `styles.css`: Stripe-docs-clean two-neutral palette (`#fafafa` bg, `#1a1a1a` fg), error red, warning amber. Plain CSS, no Tailwind. No animations or gradients.
- Three real lint fixtures copied into `web/src/samples/` and bundled via `?raw`. Type declarations in `src/vite-env.d.ts`.

**Key decisions**

- Hand-rolled CodeMirror React wrapper (`useEffect` + refs + a `Compartment` for the gutter) instead of `@uiw/react-codemirror` — keeps the dependency surface small and gives full control over the gutter swap.
- Sample files live in `web/src/samples/` as copies of `testdata/broken/*.decl` rather than imported from outside the Vite root. Files are 156–200 bytes; sync drift is unlikely and easy to spot. Vite's `?raw` loader bundles them at build time so production has no extra fetch.
- `VITE_API_URL` defaults to `http://localhost:8080` for dev. Production override happens at Pages build time per T11.
- Plain CSS (no Tailwind, no CSS-in-JS) — the design uses ~5 colors and three layouts; a framework would be over-engineering and would also add to the bundle.

**Deviations**

- None from the ticket scope.

**Verification**

- `npm install` clean (126 packages).
- `npm run build` (= `tsc -b && vite build`) green: `dist/index.html` 0.62 kB, `dist/assets/index-*.css` 4.49 kB, `dist/assets/index-*.js` 421.9 kB (137 kB gzipped — CodeMirror 6 is the floor here).
- `npm run dev` serves on 127.0.0.1:5173; the `voidslice serve` backend on :8080 responds to the same `POST /lint?filename=…` payload the frontend issues, with the expected `Access-Control-Allow-Origin: *` header on both preflight and POST.
- **UI not exercised in a browser** by the agent — the manual checklist from the ticket (drag-drop a file, click a sample, click a diagnostic to scroll, clean file shows empty sidebar) requires user verification at `http://127.0.0.1:5173` with the backend running. Build, types, and the HTTP layer are all green.
