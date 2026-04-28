# T9 · Frontend Playground (`web/`)

**Status:** todo  
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
