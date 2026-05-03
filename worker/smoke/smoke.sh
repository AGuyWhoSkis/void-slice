#!/usr/bin/env bash
# Post-deploy smoke test for the prod Worker. Hits /health and one /lint
# request whose body is a fixture with a known VALIDATE diagnostic — a
# Worker that deployed but fails to boot (wasm import error, missing
# binding, masked syntax error) trips one of these assertions.
#
# Sends an allowed Origin because the M7.2 lockdown rejects /lint requests
# from foreign origins before any work runs.
set -euo pipefail

BASE_URL="${1:?usage: smoke.sh <worker-base-url>}"
ORIGIN="${SMOKE_ORIGIN:-https://voidslice.pages.dev}"
FIXTURE="${SMOKE_FIXTURE:-testdata/broken/count-mismatch.decl}"

echo "smoke: BASE_URL=$BASE_URL ORIGIN=$ORIGIN FIXTURE=$FIXTURE"

echo "smoke: GET /health"
health=$(curl --fail-with-body -sS -m 15 -H "Origin: $ORIGIN" "$BASE_URL/health")
echo "$health"
if ! echo "$health" | grep -q '"status":"ok"'; then
  echo "smoke: /health body did not contain {\"status\":\"ok\"}"
  exit 1
fi

echo "smoke: POST /lint $FIXTURE"
lint=$(curl --fail-with-body -sS -m 30 \
  -H "Origin: $ORIGIN" \
  -H "Content-Type: application/octet-stream" \
  --data-binary "@$FIXTURE" \
  "$BASE_URL/lint?filename=$(basename "$FIXTURE")")
echo "$lint"
if ! echo "$lint" | grep -q '"diagnostics":\['; then
  echo "smoke: /lint response missing diagnostics array"
  exit 1
fi
if echo "$lint" | grep -q '"diagnostics":\[\]'; then
  echo "smoke: /lint diagnostics empty (fixture should produce a known non-empty set)"
  exit 1
fi

echo "smoke: ok"
