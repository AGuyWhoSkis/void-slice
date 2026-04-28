# T10 · Containerization

**Status:** todo  
**Version:** v2  
**Size:** small  
**Blocks:** T11 (CI/CD), T12 (deploy)

## What

Multi-stage Dockerfile for the Go binary. `docker-compose.yml` for running the full local stack (backend + frontend dev server).

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
