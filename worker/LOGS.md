# Worker log export

The Worker writes one structured JSON line per request — `method`, `path`,
`status`, `duration_ms` — at [worker/index.js:304-311](index.js#L304-L311).
Cloudflare's per-request observability keeps those queryable in the dashboard
for a few days at the platform's default retention. For post-mortems past
that window, `logpush = true` in [wrangler.toml](../wrangler.toml) opts the
script into the `workers_trace_events` Logpush dataset, and a one-time
dashboard-configured Logpush job ships those events to R2.

## Destination

| | |
|---|---|
| Provider | Cloudflare R2 (same account, no extra vendor) |
| Bucket | `voidslice-worker-logs` |
| Dataset | `workers_trace_events` |
| Sample rate | 100% (no sampling — current traffic is low) |
| Batch frequency | hourly |
| Retention | 30 days (R2 lifecycle rule on the bucket) |

The per-request JSON we emit lands inside each event's `Logs[].Message[0]`
as a string; the wrapping event also carries Cloudflare-side fields
(`EventTimestampMs`, `ScriptName`, `Outcome`, `RequestHeaders.cf-ray`, etc.).

## One-time operator setup

Done once, in the production Cloudflare account. Re-run only if the bucket
is destroyed or the job is deleted.

1. **Create the R2 bucket.** Dashboard → R2 → *Create bucket* →
   name `voidslice-worker-logs`, location *Automatic*. Or:
   ```sh
   wrangler r2 bucket create voidslice-worker-logs
   ```

2. **Add a 30-day lifecycle rule.** Dashboard → R2 → `voidslice-worker-logs`
   → *Settings* → *Object lifecycle rules* → *Add rule* → prefix empty,
   *Delete objects* after 30 days.

3. **Create the Logpush job.** Dashboard → Analytics & Logs → Logpush →
   *Add Logpush job* →
   - Dataset: **Workers trace events**
   - Filter: `ScriptName = voidslice-api` (production script only;
     preview *versions* of the same script are not separable here, see
     wrangler.toml comment)
   - Destination: **R2**, bucket `voidslice-worker-logs`, path `{DATE}/`
   - Fields: defaults are fine; ensure `Logs`, `EventTimestampMs`,
     `Outcome`, `ScriptName`, `RequestHeaders` are checked
   - Frequency: **High** (hourly batches)
   - Sample rate: **100%**

4. **Verify.** Hit the prod Worker `/health` and `/lint` a few times,
   wait ~10 minutes for the first batch, then list the bucket:
   ```sh
   wrangler r2 object list voidslice-worker-logs --prefix "$(date -u +%Y-%m-%d)/"
   ```

## Cost

| Component | Price | Expected at current traffic |
|---|---|---|
| Logpush | included with Workers Paid | $0 |
| R2 storage | $0.015 / GB-month | pennies / month |
| R2 Class A ops (writes) | $4.50 / million | one batch per hour ≈ 720/month → free tier |
| R2 egress to Workers | $0 (same account) | $0 |

Real cost dominator is storage volume × 30-day retention. At ~10k requests
/day and ~200 bytes per event after gzip, monthly storage is well under
100 MB — under one cent at $0.015/GB-month.

## Retrieval recipe

Logs land under date-prefixed gzipped JSONL files. To grep yesterday's
`/lint` requests:

```sh
DATE=$(date -u -d 'yesterday' +%Y-%m-%d)
mkdir -p /tmp/voidslice-logs && cd /tmp/voidslice-logs

# List the day's batches.
wrangler r2 object list voidslice-worker-logs --prefix "$DATE/"

# Download and grep. Each batch is one .json.gz file; lines are JSON events.
for key in $(wrangler r2 object list voidslice-worker-logs --prefix "$DATE/" \
             | awk '/Key:/ {print $2}'); do
  wrangler r2 object get "voidslice-worker-logs/$key" --file "$(basename "$key")"
done

zcat *.json.gz \
  | jq -c 'select(.Logs[0].Message[0] | fromjson | .path == "/lint")
           | .Logs[0].Message[0] | fromjson'
```

R2 is S3-compatible if you'd rather use the AWS CLI — credentials and
endpoint are in the dashboard under *R2 → Manage R2 API tokens*.

## `wrangler dev`

Unaffected. `logpush` only takes effect on `wrangler deploy`; local
development still emits the per-request JSON to stdout as before.
