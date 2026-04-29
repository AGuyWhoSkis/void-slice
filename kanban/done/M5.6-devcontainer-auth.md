# T21 · Fix Dev Container Authentication

**Status:** done  
**Version:** meta  
**Size:** small

## What

Claude Code OAuth inside the dev container fails: the browser callback URL is unreachable from within the container, and pasting the auth code also does not work. Investigate and implement a workaround so authentication persists across container restarts without manual intervention.

## Scope

Options to evaluate in order of preference:

1. **Host-credential bind-mount** — authenticate on the host, then mount `~/.claude/` (or the relevant credential file) into the container via `devcontainer.json` `mounts`
2. **API key auth** — use `ANTHROPIC_API_KEY` env var instead of OAuth; set it in `devcontainer.json` `remoteEnv` or a local `.env`
3. **Port-forward the callback** — configure VS Code port forwarding so the OAuth redirect reaches the container's loopback address

Document the chosen approach in `devcontainer.json` or a dev-setup note so the fix survives container rebuilds. Also add a **Dev setup** entry to T20 (CLAUDE.md) — the auth solution and re-authentication steps — so every Claude Code session inside the container has that context without reading devcontainer.json.

## Dependencies

None

## Verification

```bash
# inside container
claude auth status   # should return authenticated
# restart container, re-open, verify auth persists without re-running the OAuth flow
```

## Completion

Chose option 1 (host bind-mount). Added to `.devcontainer/devcontainer.json` mounts:

```
"source=${localEnv:HOME}/.claude,target=/home/node/.claude,type=bind,consistency=cached"
```

`${localEnv:HOME}` resolves to the WSL2 host home at container creation time. `CLAUDE_CONFIG_DIR=/home/node/.claude` (already set in `containerEnv`) means Claude Code reads credentials from the bind-mounted path automatically.

Re-auth procedure: run `claude auth login` in a WSL2 terminal on the host. No container rebuild needed. Documented in `CLAUDE.md § Dev setup`.
