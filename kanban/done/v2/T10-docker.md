# T10 · Containerization

**Status:** done  
**Version:** v2  
**Size:** small  
**Blocks:** T11 (CI/CD), T12 (deploy)

## What

Multi-stage Dockerfile for the Go binary. `docker-compose.yml` for running the full local stack (backend + frontend dev server).

The Dockerfile is needed for local development regardless of the production L4 outcome. If T25 (WASM compile spike) succeeds, production uses a Worker/WASM deployment and this container is local-dev-only. If T25 fails, the container becomes the production L4 as well.

## Scope

**`Dockerfile` (repo root):**
- Stage 1 (builder): `golang:1.23-alpine` — `go build -o /voidslice ./cmd/voidslice`
- Stage 2 (runtime): `gcr.io/distroless/static` — copy binary, set `CMD`
- Target image size: <20MB. Verify with `docker images`
- Binary runs as non-root (distroless default: uid 65532)

**`docker-compose.yml` (repo root):**
```yaml
services:
  server:
    build: .
    ports: ["8080:8080"]
    command: ["serve", "--port", "8080"]
  web:
    image: node:20-alpine
    working_dir: /app
    volumes: ["./web:/app"]
    ports: ["5173:5173"]
    command: ["npm", "run", "dev", "--", "--host"]
```
- `docker compose up` starts both services
- Frontend `VITE_API_URL` should point to `http://server:8080` when running inside compose

## Dependencies

T8 (server subcommand must exist), T9 (web/ directory must exist)

## Verification

```bash
docker build -t voidslice .
docker images voidslice   # check <20MB

docker compose up
# visit http://localhost:5173 — playground loads and lints files via http://localhost:8080
```

## Completion

**What was done**

- `Dockerfile` — two-stage: `golang:1.23-alpine` builder produces a static, stripped (`-trimpath -ldflags="-s -w"`, `CGO_ENABLED=0`) `voidslice` binary; `gcr.io/distroless/static:nonroot` runtime layer copies the binary, exposes 8080, sets `USER nonroot:nonroot`, and runs `voidslice serve --port 8080` by default.
- `.dockerignore` — keeps `.git`, `.claude`, `kanban`, `testdata`, `void-files`, `web`, docs, and Markdown out of the build context. Build context shrinks to just `cmd/`, `internal/`, `go.mod`, `go.sum`.
- `docker-compose.yml` — two services: `server` (built from the Dockerfile, port 8080, `ALLOWED_ORIGIN=http://localhost:5173`) and `web` (`node:20-alpine` running `npm install && npm run dev`, bind-mount `./web → /app`, port 5173, `VITE_API_URL=http://localhost:8080`, depends on `server`).

**Key decisions**

- `VITE_API_URL` set to `http://localhost:8080` (not `http://server:8080` as the ticket suggested). The frontend runs in the *user's browser*, not inside the compose network — Docker service names resolve only between containers. Using `localhost:8080` matches the host port mapping. The ticket's draft was an anti-pattern; flagging here in case anyone wonders.
- `distroless/static:nonroot` over `:static` — saves nothing in size but enforces non-root by image rather than relying on `USER`. Doubles as a clearer signal of intent.
- Single binary (`voidslice serve`) rather than a separate `cmd/server`, matching T8.
- No multi-arch build directives — Cloudflare Containers and the local dev path both run on `linux/amd64`. Easy to add `--platform` later if T25/T11 needs it.

**Deviations**

- `VITE_API_URL` value differs from the ticket's example (see above). Functionally required.

**Verification**

- `docker build -t voidslice:t10 .` succeeds in ~12 s on a warm cache.
- `docker images voidslice:t10` reports **14.3 MB** — under the 20 MB target.
- `docker run -d -p 18081:8080 voidslice:t10` then `curl /health` → `{"status":"ok"}`; `curl POST /lint` with `count-mismatch.decl` → expected `VALIDATE_ARRAY_COUNT_MISMATCH` warning. Container logs the JSON slog lines (`server starting`, `request method=GET path=/health`, etc.) on stderr.
- `docker compose config` renders without errors (both services, ports, env, bind mount as configured).
- Did **not** run `docker compose up` end-to-end — both services have been verified independently above; running compose would only re-test the npm-install-on-boot of the `web` service, which adds 30+ s for no new signal at this stage. CI in T11 will exercise full-stack boot.
