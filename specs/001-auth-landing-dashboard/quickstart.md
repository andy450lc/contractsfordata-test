# Quickstart: Sign-Up, Sign-In, Landing Page, and Dashboard

**Feature**: 001-auth-landing-dashboard · Validation guide (run after
implementation; see [plan.md](plan.md) for design, [spec.md](spec.md) for
acceptance criteria).

## Prerequisites

- Docker Desktop running (local Postgres via `backend/docker-compose.yml`;
  testcontainers for the Go suite).
- Node 22 and npm 10 (`web/.nvmrc`, `packageManager`).
- A WorkOS staging environment dedicated to SoW, with:
  - Email/password as the only enabled authentication method.
  - Redirect URI `http://localhost:8080/v1/auth/callback` allow-listed.
  - A webhook endpoint `https://<ngrok-host>/v1/webhooks/workos` with
    `user.created`, `user.updated`, `user.deleted`, and its signing
    secret.
- Env files filled from the templates:
  - `backend/.env`: `DATABASE_URL`, `WORKOS_API_KEY`, `WORKOS_CLIENT_ID`,
    `WORKOS_WEBHOOK_SECRET`, `WORKOS_REDIRECT_URI`, `WEB_APP_URL`,
    `SESSION_COOKIE_KEY` (generate with
    `echo "k1:$(openssl rand -base64 32)"`), `CORS_ALLOWED_ORIGINS`.
  - `web/.env.local`: `VITE_API_URL=http://localhost:8080`.

Work pauses for this hand-off after the environment templates land (spec
Assumptions). Everything below the automated section needs the real
values.

## Automated validation (CI-equivalent)

```bash
cd backend && make lint && go build ./... && make test
```

```bash
cd web && npm run generate:api && git diff --exit-code src/lib/api/schema.d.ts && npm run typecheck && npm run lint && npm run format:check && npm run test:coverage && npm run build
```

```bash
cd backend && go tool oapi-codegen -config oapi-codegen.yaml ../contracts/openapi.yaml && git diff --exit-code internal/models/dto/api.gen.go
```

Key suites and what they prove:

| Suite | Proves |
|---|---|
| `backend/tests/integration/health_test.go` | US4: `/livez` static, `/readyz` DB-checked and failing when Postgres pauses, hardening headers, 429 on the global limiter, single error log with ids |
| `backend/tests/integration/auth_test.go` | US1–US3, FR-009–FR-013: login redirect carries a signed state; callback with a valid code sets the cookie and provisions the user; tampered or expired state redirects with `invalid_state` and no cookie; refresh with the cookie returns a token and rotates the cookie; refresh without cookie, with a foreign `Origin`, or with a provider rejection answers 401/403 and clears; logout revokes and clears; stricter limiter on `/v1/auth` |
| `backend/tests/integration/me_test.go` | FR-006, SC-005: no or invalid bearer → 401; valid bearer → record; missing row → catch-up provisioning |
| `backend/tests/integration/webhook_test.go` | FR-008, SC-006: signed create/update applied; duplicates and stale events are no-ops; tampered signature → 401; delete removes the row |
| `web` auth integration tests | US1–US3: sign-in and sign-up links point at the API login endpoint with the right screen and return path; `/dashboard` guarded with redirect preservation; callback refreshes and lands; dashboard greets by name and email; sign-out returns to `/`; error notices per `?error=`; bearer 401 triggers one refresh and one retry |
| `web` session unit tests | R8, R11: refresh scheduling buffer, single-flight refresh, expiry handling |

## Manual end-to-end walk (Stories 1–3)

1. Start the tunnel and set the WorkOS webhook URL if it changed:

   ```bash
   ngrok http 8080
   ```

2. Start Postgres and the backend:

   ```bash
   cd backend && make db-up && make migrate-up && make dev
   ```

3. Start the web app:

   ```bash
   cd web && npm run dev
   ```

4. Open the landing page locally (any static server, for example
   `npx serve marketing`), confirm the two links point at the app.
5. **Story 1 — sign-up**: `http://localhost:5173/sign-up` → "Create
   account" → hosted AuthKit registration → register with a fresh email
   and verify it → the browser returns through `localhost:8080/v1/auth/
   callback` to `localhost:5173/callback` and lands on `/dashboard`
   greeting you by name and email. `psql $DATABASE_URL -c "select id,
   email, name from users"` shows the row.
6. **Story 2 — persistence**: reload → still signed in. Close the tab,
   reopen `http://localhost:5173/dashboard` → still signed in. Leave the
   tab open past the access-token lifetime (WorkOS dashboard session
   settings), then click anything → no redirect. Repeat in a Chrome
   incognito window, Safari, and Firefox.
7. **Story 3 — enforcement**: private window → `/dashboard` → sign-in
   page with `?redirect=/dashboard`; sign in → back on `/dashboard`.
   `curl -i localhost:8080/v1/me` → 401. `curl -i -X POST
   localhost:8080/v1/auth/refresh` → 403 (no Origin). Sign out → `/`;
   `/dashboard` redirects again.
8. **Two tabs**: with two dashboard tabs open, wait past the token
   lifetime and click in both. Both stay signed in. Sign out in one; the
   other redirects to `/` on its next action.
9. **Webhook replay** (optional): re-send a delivered `user.created` from
   the WorkOS dashboard → 200 and the row count is unchanged.

## Browser verification rule

Per the constitution's Development Workflow, the dashboard, sign-in,
sign-up, and landing pages are looked at in the shared preview browser
before the feature is reported done. Steps 5–7 are that look.
