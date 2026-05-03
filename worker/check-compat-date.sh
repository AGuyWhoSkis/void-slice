#!/usr/bin/env bash
# Fails when wrangler.toml's compatibility_date drifts past THRESHOLD_DAYS.
# A stale compat_date silently opts the Worker out of newer Workers-runtime
# behavior — the goal is to make the bump a deliberate decision (skim the
# changelog, tick the date forward) rather than letting it rot.
#
# Threshold is hard-coded by design: no env var, no input. Bumping requires
# editing wrangler.toml and re-running this check.
set -euo pipefail

THRESHOLD_DAYS=180

WRANGLER_TOML="${WRANGLER_TOML:-wrangler.toml}"

compat_date=$(grep -E '^[[:space:]]*compatibility_date[[:space:]]*=' "$WRANGLER_TOML" \
  | sed -E 's/.*"([0-9]{4}-[0-9]{2}-[0-9]{2})".*/\1/' \
  | head -1)

if [ -z "$compat_date" ]; then
  echo "::error::compatibility_date not found in $WRANGLER_TOML"
  exit 1
fi

compat_epoch=$(date -u -d "$compat_date" +%s)
now_epoch=$(date -u +%s)
age_days=$(( (now_epoch - compat_epoch) / 86400 ))

if [ "$age_days" -gt "$THRESHOLD_DAYS" ]; then
  cat >&2 <<EOF
::error file=${WRANGLER_TOML}::compatibility_date is stale
  current: $compat_date  (age: ${age_days} days)
  threshold: ${THRESHOLD_DAYS} days

Bump the date in ${WRANGLER_TOML} after reviewing the Workers compat-date changelog:
  https://developers.cloudflare.com/workers/configuration/compatibility-dates/
EOF
  exit 1
fi

echo "compat_date=${compat_date} age=${age_days}d (threshold ${THRESHOLD_DAYS}d)"
