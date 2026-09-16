# Implementation Plan: Sign-Up, Sign-In, Landing Page, and Dashboard

**Branch**: `feature/001-auth-landing-dashboard` | **Date**: 2026-09-03 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-auth-landing-dashboard/spec.md`

## Summary

Stand up the whole repository foundation and the first user-facing loop in
one feature: a Go API in `backend/` (echo v5 + uber fx, Postgres via pgx +
sqlc + goose, slog with correlation ids, OTel, Prometheus RED metrics,
Sentry, hardened middleware, `/livez` + `/readyz`), a React SPA in `web/`
(Vite + TypeScript strict, shadcn/ui + Tailwind, react-router, TanStack
Query, generated OpenAPI client), and a static landing page in
`marketing/`. Authentication follows constitution 1.1.0: the API is the
OAuth client for WorkOS AuthKit. `GET /v1/auth/login` redirects to hosted
AuthKit, `GET /v1/auth/callback` exchanges the code server-side and sets
an encrypted HttpOnly refresh cookie on the API host, `POST /v1/auth/refresh`
turns that cookie into a short-lived access token the SPA keeps in memory,
and `POST /v1/auth/logout` revokes the session. Bearer JWTs are validated
against the WorkOS JWKS with no database hit. A `users` table is
provisioned at callback time and kept in sync by signature-verified,
order-safe WorkOS webhooks with a `/v1/me` catch-up. The dashboard greets
the user by name and email. Structure, tooling, and most foundation code
are copied from the marketplace repository and trimmed to this scope.

## Technical Context

**Language/Version**: Go 1.25 (`GOTOOLCHAIN=auto`; go.mod pins the exact
patch at init) for `backend/`; TypeScript ~6.0, React 19, Vite 8, Node 22
(pinned in `.nvmrc`) for `web/`; plain HTML/CSS for `marketing/`.

**Primary Dependencies**:
- Backend: echo v5, uber-go/fx, pgx v5 + sqlc, goose, caarlos0/env +
  godotenv, go-playground/validator, cockroachdb/errors, OTel SDK +
  otelecho, prometheus/client_golang, getsentry/sentry-go,
  workos-go v10 (server-side code exchange, refresh, revoke, user fetch,
  webhook verification), lestrrat-go/jwx v3 (JWKS cache + JWT validation),
  oapi-codegen (types only), air (dev reload), testcontainers-go.
- Web: react-router-dom 7, @tanstack/react-query 5, zustand, openapi-fetch
  + openapi-typescript, zod 4, shadcn/ui + Tailwind 4, vitest + Testing
  Library + MSW, ESLint flat config + Prettier, wrangler (deploy only).
- Marketing: none.

**Storage**: Postgres (Supabase in deployed stages, Docker Postgres
locally, testcontainers in tests). One table, `users`, via goose migration
`000001_users.sql`; queries via sqlc.

**Testing**: Backend — integration-first with testcontainers Postgres,
booting the real fx graph, a local JWKS `httptest` server, and a fake
WorkOS HTTP server reached through `workos.WithBaseURL`. Web — Vitest +
Testing Library + MSW at the HTTP boundary; no vendor SDK exists to mock.
Playwright deferred (no deployed stage yet).

**Target Platform**: Go API on Railway (Linux container); SPA and landing
page as Cloudflare Workers static assets; local dev on macOS at
`localhost:8080` / `localhost:5173`.

**Project Type**: Web application monorepo (`contracts/`, `backend/`,
`web/`, `marketing/`).

**Performance Goals**: Auth middleware adds no per-request DB hit (JWKS
cached in-process). Probes respond under 1 s. Landing page is static and
readable under 2 s (SC-003). Silent refresh completes within one round
trip to the API plus one to WorkOS.

**Constraints**: Access tokens in JS memory only. Refresh token only ever
inside an AES-256-GCM cookie on the API host (`HttpOnly; Secure;
SameSite=Strict; Path=/v1/auth`). No PII in logs (ids only). Session
endpoints verify `Origin` and carry a stricter rate limit. No WorkOS
custom domain. Secrets backend-only. Cyclomatic complexity ≤ 10.

**Scale/Scope**: 6 API operations, 1 table, 5 SPA routes, 1 static page,
2 CI workflows plus 1 for marketing, single replica.

## Constitution Check

*GATE: evaluated pre-Phase-0 and re-checked post-design — PASS. One
recorded deferral, no violations.*

| Principle | Status | Notes |
|---|---|---|
| I. Spec-Driven Development | PASS | spec.md governs; `contracts/openapi.yaml` is designed here (`contracts/openapi.yaml` in this folder is the target content) before any implementation. |
| II. Test-First & Meaningful Coverage | PASS | Failing integration tests precede each vertical (auth endpoints, middleware, `/me`, webhook, health, web flows). Coverage gate 85% on gated packages, excluding `cmd/`, `boot/`, `sqlcgen/`, `api.gen.go`, `components/ui/`, `main.tsx`. |
| III. Security is Paramount | PASS | Every `/v1` endpoint is behind middleware: bearer JWT or signed cookie plus `Origin` check plus stricter rate limit; webhook HMAC-verified on raw body; no documents in this feature; secrets never reach the bundle; refresh token encrypted at rest in the cookie and never logged; email stored with stated retention (spec FR-007). |
| IV. Simplicity & Patterns | PASS | Entity-per-layer files (`auth-*`, `user-*`, `webhook-*`, `health-*`); fx DI; no organisation, role, or team modeling; the marketplace's Hatchet worker, S3, R2, suspension, and admin surfaces are not copied. |
| V. Contract Integrity & RESTful API | PASS | Contract first; oapi-codegen types-only gate and openapi-typescript diff gate both wired from day one; timestamps Unix ms; `/v1` prefix. |
| VI. Observability & Debuggability | PASS | slog with `request_id`/`trace_id`/`span_id`/`user_id` by construction, OTel, RED metrics + pgxpool collector, Sentry no-op outside deployed stages, central error handler logs once. Web: route error boundaries and root boundary; Sentry browser SDK deferred until the first deployed stage (frontend conventions §8 scopes it to that moment). |

**Post-Phase-1 re-check**: PASS. Design added the `SESSION_COOKIE_KEY`
rotation scheme and the `Origin` check, both strengthening III. No
Complexity Tracking entries.

## Project Structure

### Documentation (this feature)

```text
specs/001-auth-landing-dashboard/
├── plan.md                    # This file
├── research.md                # Phase 0 — decisions R1–R16
├── data-model.md              # Phase 1 — users table, session cookie, sync rules
├── quickstart.md              # Phase 1 — run & validate end to end
├── contracts/
│   ├── openapi.yaml           # Target content for contracts/openapi.yaml
│   ├── session-cookie.md      # Refresh cookie format, state token, Origin rule
│   ├── workos-webhook.md      # Inbound webhook contract
│   ├── health-and-ops.md      # /livez, /readyz, /metrics
│   └── ui-routes.md           # SPA routes and landing page behavior
└── tasks.md                   # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
contracts/
└── openapi.yaml                          # designed here, source of truth

backend/
├── go.mod / go.sum                       # module github.com/pixels-two/sow/backend
├── Makefile                              # run, dev, build, test, lint, db-up, migrate-*, sqlc-generate, oapi-generate
├── .air.toml  .golangci.yml  .env.example  sqlc.yaml  oapi-codegen.yaml  docker-compose.yml
├── cmd/app/main.go                       # fx.New(boot.AppOptions()).Run()
├── internal/
│   ├── boot/                             # app.go (fx graph), env.go, config.go, database.go, http.go, otel.go, sentry.go, workos.go
│   ├── config/config.go                  # AppConfig incl. WORKOS_*, WEB_APP_URL, SESSION_COOKIE_KEY, SESSION_MAX_AGE_SECONDS, AUTH_RATE_LIMIT_*
│   ├── common/                           # constants.go (context keys, cookie name, header names), errors.go (ErrUnauthorized, ErrUserNotFound, ErrInvalidState, …), status.go (HTTPStatus)
│   ├── logging/logging.go                # ctx-aware slog wrapper
│   ├── metrics/                          # metrics.go, pool-collector.go
│   ├── interfaces/interfaces.go
│   ├── middleware/                       # request-logger.go, validate-request.go, auth.go (bearer JWKS), origin.go (Origin allowlist), auth-rate-limit.go
│   ├── session/                          # cookie.go (AES-GCM seal/open with key ids), state.go (HMAC state token)
│   ├── routes/                           # health-routes.go, auth-routes.go, user-routes.go, webhook-routes.go
│   ├── controllers/                      # health-, auth-, user-, webhook-controller.go
│   ├── services/                         # health-, auth-, user-service.go; workos-client.go (interface over the SDK)
│   ├── stores/                           # health-store.go, user-store.go, queries/users.sql, sqlcgen/
│   ├── models/                           # health.go, user.go, session.go; dto/ (api.gen.go, auth-dto.go, webhook-dto.go, user-dto.go)
│   └── migrations/                       # embed.go, 000001_users.sql
└── tests/integration/                    # harness_test.go (fx app + testcontainers + fake WorkOS + JWKS), health_test.go, auth_test.go, me_test.go, webhook_test.go

web/
├── package.json  .nvmrc  vite.config.ts  tsconfig*.json  eslint.config.js  .prettierrc  components.json  wrangler.jsonc  .env.example
├── index.html
├── public/_headers                       # CSP and security headers for Workers static assets
└── src/
    ├── main.tsx  index.css
    ├── app/                              # router.tsx, providers.tsx, root-error-boundary.tsx
    ├── lib/                              # env.ts, api/client.ts, api/schema.d.ts (generated), auth/session.ts (refresh, scheduling, single-flight), dates.ts, utils.ts
    ├── stores/session.ts                 # Zustand: access token, expiry, status
    ├── features/auth/                    # pages/sign-in-page.tsx, sign-up-page.tsx, callback-page.tsx; components/protected-layout.tsx, sign-out-button.tsx; api.ts (useMe); return-target.ts; index.ts
    ├── features/dashboard/               # pages/dashboard-page.tsx; index.ts
    ├── pages/not-found-page.tsx
    ├── components/ui/                    # shadcn: button, card, alert, skeleton
    ├── components/full-page-loader.tsx
    └── test/                             # setup.ts, render.tsx, msw.ts

marketing/
├── index.html  styles.css  favicon.svg
├── wrangler.jsonc
└── _headers

.github/workflows/
├── backend.yml                           # paths: backend/**, contracts/**
├── web.yml                               # paths: web/**, contracts/**
└── marketing.yml                         # paths: marketing/**
```

**Structure Decision**: Monorepo per `docs/monorepo-and-ci.md`. Backend
follows conventions §1 with a new `session` package for the cookie and
state primitives, kept separate from `middleware` because both the
controller (set/clear) and the middleware (read) use it. Web follows the
feature-module layout; `lib/auth/session.ts` replaces the vendor SDK the
marketplace used. The health vertical doubles as the proof path through
every backend layer, as in the marketplace's first feature.

## Complexity Tracking

No constitution violations. Deferrals and their grounds:

| Deferred item | Mandate source | Why deferral is compliant |
|---|---|---|
| Web Sentry browser SDK and browser→backend `traceparent` | conventions §8 | Scoped by the same section to "the first environment real users touch"; no deployed stage exists. The API accepts `traceparent` already, so wiring the browser side is a one-file change in the deployment feature. |
| Playwright E2E | conventions §11 | Mandated against a real backend in a preview environment; none exists. The manual walk in quickstart.md covers the hosted flow until then. |
| Redis-backed rate limiter and idempotency store | conventions §3, §10 | Single replica; in-memory limiter is the documented default. No endpoint in this feature sends an agreement, records a signature, or sends email. |

## Implementation Notes (post-build reconciliation, 2026-09-03)

Deviations from the planned design, all recorded as they happened:

- **Landing page links, not `<base href>`** (research R12): a `<base>` tag
  rewrote the stylesheet and favicon URLs too. The app origin now appears
  directly in the four links, one search-and-replace per stage.
- **Go 1.25.7, not 1.25** (research R16): goose v3.27.3 requires it. sqlc
  and air are pinned to the last releases that build on the 1.25 line.
- **Credentialed CORS is global**: echo's CORS middleware answers
  preflights before a group-level middleware could run, so
  `AllowCredentials: true` sits on the global CORS config with the explicit
  origin allowlist. The `Origin` check on the cookie endpoints is the
  per-route guard.
- **`session` package** holds the AES-GCM cookie seal, the keyring, and
  the HMAC state, imported by both the service and the controller.
- **Callback logs its own failure line**: the callback answers with a
  redirect rather than the central error handler's JSON, so it maps the
  error to a code and writes the single log line itself.
- **Webhook envelope is a hand-written DTO** with a raw `data` field so
  the user object is parsed only for `user.*` events. The generated
  `WorkOSEvent` type stays the contract's shape.
- **Provider user type**: the SDK's `User` is an alias of
  `EmailChangeConfirmationUser` with pointer `first_name`/`last_name`.
  Display name joins the two.
- **Web MSW default is anonymous**: the default refresh handler answers
  401. Tests that need a signed-in visitor install `refreshHandler()`.
- **Dashboard browser look is deferred to the credential hand-off**: the
  sign-in, sign-up, error-notice, and guard-redirect states were looked at
  against the dev server. The dashboard needs a real WorkOS sign-in.
