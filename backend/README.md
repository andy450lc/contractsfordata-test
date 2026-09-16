# SoW Backend

Go service for SoW. Governed by the constitution
(`.specify/memory/constitution.md`) and `docs/backend-conventions.md` —
read those before changing anything structural.

## Architecture

```
cmd/app/main.go          pure fx wiring: fx.New(boot.AppOptions()).Run()
internal/
  boot/                  one fx constructor per dependency (config, db,
                         http server, otel, metrics, sentry) + AppOptions,
                         the full graph (tests boot the same graph)
  config/                env-tagged AppConfig — no logic
  common/                context keys, constants, sentinel errors (ErrXxx)
  logging/               THE logging package: ctx-aware slog; every line
                         carries request_id/trace_id/span_id/user_id by
                         construction. Bare slog/log/fmt.Print is banned.
  middleware/            structural middlewares (ValidateRequest, ...)
  routes/ -> controllers/ -> services/ -> stores/ -> models/
                         request flow, strictly downward; one file per
                         entity per layer (agreement-routes.go, ...)
  stores/queries/*.sql   sqlc sources -> stores/sqlcgen (generated; never
                         hand-edit; `make sqlc-generate`)
  migrations/            goose NNNNNN_description up/down SQL pairs
tests/integration/       primary suite: real fx app + testcontainers PG
```

Request flow: route (+ middleware chain, declared vertically per route) →
thin controller (extract, one service call, respond) → service (business
logic, owns transactions) → store (sqlc queries, own timeouts, wraps
external errors with stack) → Postgres. Errors bubble to the centralized
error handler in `boot/http.go` — the only place errors are logged.

The business API lives under `/v1` and is contract-first: change
`contracts/openapi.yaml` before implementing. `/livez`, `/readyz`,
`/metrics` are operational endpoints outside the contract.

Current endpoints:

- `GET /livez` — liveness. Static ok, no dependency consulted.
- `GET /readyz` — readiness. 503 when the database check fails.
- `GET /metrics` — Prometheus scrape endpoint.

The auth, user, and webhook verticals land with the WorkOS session work.

## Auth

WorkOS is the identity provider and this API is the OAuth client,
following the token-mediating backend pattern. Sign-in, sign-up, email
verification, password reset, and resend all go through WorkOS's
headless User Management API — the browser reaches WorkOS itself only
for "Continue with Google". Configuration comes from `WORKOS_API_KEY`,
`WORKOS_CLIENT_ID`, `WORKOS_WEBHOOK_SECRET`, `WORKOS_REDIRECT_URI`,
`WEB_APP_URL`, and `SESSION_COOKIE_KEY` — all required. See
`.env.example`.

`internal/mail` sends the two platform emails WorkOS itself does not
send (password reset, "account exists") and requested Word-template
attachments through a small `Mailer` interface. Resend is used when
`RESEND_API_KEY` and a verified `EMAIL_FROM` are configured. An
unconfigured local transport reports email delivery as unavailable instead
of claiming success; direct Word download remains available.

## Template delivery

`POST /v1/template-deliveries/download` and
`POST /v1/template-deliveries/email` require an explicit
`"acknowledged": true` body value. The email route additionally requires a
valid work email and an `Idempotency-Key` header. Both routes validate the
same optional configurator payload and use the same deterministic OOXML
generator; the email service attaches the exact `.docx` bytes returned by
the direct-download path. Public calls are origin-checked, body-limited, and
covered by both global and template-specific rate limits.

For production, set `RESEND_API_KEY`, verify the sender domain in Resend,
set `EMAIL_FROM` to an address on that domain, and include every web and
marketing origin in `CORS_ALLOWED_ORIGINS`. Do not expose either mail value
to browser code.

## Quickstart

Prerequisites: Go 1.25+, Docker, make.

```bash
cp .env.example .env      # local defaults; never commit .env
make db-up                # local Postgres (docker compose)
make migrate-up           # goose migrations
make run                  # start the server
curl localhost:8080/readyz
```

## Development loop

```bash
make dev                  # air live reload: save a .go file, server
                          # rebuilds + restarts automatically (<10s)
make test                 # integration suite (needs Docker running)
make lint                 # golangci-lint, the CI rule set
make sqlc-generate        # after editing internal/stores/queries/*.sql
make oapi-generate        # after editing ../contracts/openapi.yaml
```

`make migrate-down` / `migrate-force` are guarded to `STAGE=dev`.

CI (`.github/workflows/backend.yml`, path-filtered) gates every PR on:
build, lint, race tests + 85% coverage on non-excluded packages, sqlc
diff, oapi-codegen diff, govulncheck, and a migration up/down smoke test.
