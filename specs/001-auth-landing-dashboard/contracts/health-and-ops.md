# Operational Endpoint Contracts: Backend Skeleton

These endpoints are infrastructure probes, deliberately **outside** the
`/v1` business API and therefore not part of `contracts/openapi.yaml`
(constitution V; conventions §3; research R9). They are contractual for
the deployment platform and the metrics scraper, so they are fixed here.

All three are exempt from access logging, request metrics, and rate
limiting.

## GET /livez — liveness

Answers "is the process up". Consults no external dependency.

- **200** `application/json` — always, while the process can serve:

  ```json
  { "status": "ok" }
  ```

- No other statuses. A non-response is the failure signal (platform
  restarts the instance).

## GET /readyz — readiness

Answers "can this instance do its job". Runs all registered dependency
checks (initially: `database`), each bounded to 1s.

- **200** `application/json` — every check healthy:

  ```json
  { "status": "ok", "checks": [ { "name": "database", "healthy": true } ] }
  ```

- **503** `application/json` — any check failed (platform withholds
  traffic; instance is NOT restarted):

  ```json
  {
    "status": "unavailable",
    "checks": [
      { "name": "database", "healthy": false, "error": "context deadline exceeded" }
    ]
  }
  ```

- `error` is a short operational message — never connection strings,
  credentials, or PII.

## GET /metrics — Prometheus scrape

- **200** `text/plain; version=0.0.4` — Prometheus exposition format via
  `promhttp.Handler()`.
- Includes at minimum: `http_request_duration_seconds{method,path,status}`
  (histogram; `path` is the route template), pgxpool stats collector
  gauges, and default Go runtime metrics.
- Not exposed on the public ingress in deployed environments (scraper
  network only) — platform configuration, out of this repo's scope, but
  noted as the operating assumption.

## /v1 business API

`contracts/openapi.yaml` (repo root) is created by this feature as a valid
OpenAPI 3.1 shell — `info`, `servers`, `paths: {}` — and gains its first
operations in the first business feature. Error envelope and validation
response shapes for `/v1` (`{"error": "...", "details": [...]}` per
conventions §3) become contractual when the first operation lands.

## Cross-cutting response contract (all routes)

Every response, including the three above, carries the hardening headers
(conventions §3): `X-Content-Type-Options: nosniff`,
`X-Frame-Options: DENY`, CSP `default-src 'none'`, no-referrer — plus
HSTS on TLS-terminated traffic (direct TLS or `X-Forwarded-Proto: https`
from the deployed proxy; browsers ignore HSTS on plain HTTP, so local
non-TLS responses correctly omit it). Every
response carries an `X-Request-ID` (incoming value honored, else
generated). Unknown `/v1` routes return `404` in the standard error shape;
rate-limited requests receive `429`.
