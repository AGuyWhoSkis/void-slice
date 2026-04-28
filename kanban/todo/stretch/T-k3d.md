# T-k3d · k3d Kubernetes Lab (Stretch)

**Status:** todo  
**Version:** stretch (evaluate at end-of-week-1)  
**Size:** medium

## What

Run the containerized Go service in a local k3d (k3s-in-Docker) cluster to validate the Kubernetes deployment story. README-v1.md defers this explicitly to an end-of-week-1 velocity check.

## Scope

- Spin up a k3d cluster locally
- Write a minimal `k8s/deployment.yaml` for the Go service (Deployment + Service)
- Import the Docker image into the k3d registry
- Verify the service is reachable inside the cluster via `kubectl port-forward`
- Smoke-test: `curl` the `/health` endpoint through the forwarded port

**Not in scope:** Helm chart, Ingress, persistent storage, production k8s deployment.

## Gate

Only start this ticket if week-1 velocity was on track and there is buffer time. If week 1 ran long, drop without guilt — Cloudflare Containers is the production target anyway.

## Dependencies

T10 (Docker image must build cleanly)

## Verification

```bash
k3d cluster create voidslice-dev
k3d image import voidslice:latest -c voidslice-dev
kubectl apply -f k8s/deployment.yaml
kubectl port-forward svc/voidslice 8080:8080 &
curl http://localhost:8080/health   # {"status":"ok"}
```
