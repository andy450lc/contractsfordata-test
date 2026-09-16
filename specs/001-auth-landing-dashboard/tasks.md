# Tasks: Sign-Up, Sign-In, Landing Page, and Dashboard

**Input**: Design documents from `/specs/001-auth-landing-dashboard/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included. The constitution mandates TDD: every test task is written and failing before the implementation task that follows it.

**Organization**: Phases 1–2 build the foundation (this is the repository's first feature, so User Story 4's operable foundation is the prerequisite for Stories 1–3). Phases 3–6 map to the spec's user stories in priority order.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)
- Paths are repository-relative. Marketplace paths refer to `../marketplace/`.

## Path Conventions

- Backend: `backend/` (Go, module `github.com/pixels-two/sow/backend`)
- Web: `web/` (Vite + React SPA)
- Marketing: `marketing/` (static)
- Contract: `contracts/openapi.yaml`
- CI: `.github/workflows/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Copy the marketplace toolchain and create the monorepo skeleton so every later task has a home.

- [X] T001 Create `contracts/openapi.yaml` from `specs/001-auth-landing-dashboard/contracts/openapi.yaml` (strip the two comment lines at the top)
- [X] T002 [P] Create `backend/` from the marketplace foundation: copy `Makefile`, `.air.toml`, `.golangci.yml`, `sqlc.yaml`, `oapi-codegen.yaml`, `docker-compose.yml`, `.gitignore`, `README.md`; run `go mod init github.com/pixels-two/sow/backend` with `go 1.25`; add the `tool` directives for air, goose, sqlc, oapi-codegen; replace `marketplace` names with `sow` in the compose file and Makefile default DSN
- [X] T003 [P] Create `web/` from the marketplace toolchain: copy `package.json` (drop `@workos-inc/authkit-react`, `motion`, `radix-ui`, `tag-input` deps; add `zustand`, `wrangler` dev dep), `.nvmrc`, `.npmrc`, `.prettierrc`, `.prettierignore`, `.gitignore`, `eslint.config.js`, `tsconfig*.json`, `vite.config.ts`, `components.json`, `index.html`, `public/favicon.svg`, `src/index.css`, `src/main.tsx`, `src/lib/utils.ts`, `src/lib/dates.ts`, `src/test/setup.ts`, `src/test/render.tsx`, `src/components/full-page-loader.tsx`, `src/components/ui/{button,card,alert,skeleton}.tsx`; rename product strings; run `npm install`
- [X] T004 [P] Create `marketing/` with `index.html`, `styles.css`, `favicon.svg`, `wrangler.jsonc` (`assets.directory: "."`, `not_found_handling: "404-page"`), `_headers` (CSP `default-src 'self'`, HSTS, nosniff, frame-deny, referrer-policy), and `404.html`
- [X] T005 [P] Create `.github/workflows/backend.yml`, `.github/workflows/web.yml`, and `.github/workflows/marketing.yml` from the marketplace workflows with the path filters from `docs/monorepo-and-ci.md` (marketing: `wrangler deploy --dry-run` as its only check)
- [X] T006 [P] Create `backend/.env.example` and `web/.env.example` documenting every variable in plan.md Technical Context (`STAGE`, `PORT`, `DATABASE_URL`, `DB_POOL_*`, `CORS_ALLOWED_ORIGINS`, `RATE_LIMIT_*`, `AUTH_RATE_LIMIT_*`, `SHUTDOWN_GRACE_SECONDS`, `WORKOS_API_KEY`, `WORKOS_CLIENT_ID`, `WORKOS_WEBHOOK_SECRET`, `WORKOS_REDIRECT_URI`, `WORKOS_JWKS_URL`, `WORKOS_BASE_URL`, `WEB_APP_URL`, `SESSION_COOKIE_KEY`, `SESSION_MAX_AGE_SECONDS`, `SENTRY_DSN`, `OTEL_EXPORTER_OTLP_ENDPOINT`; web: `VITE_API_URL`)
- [X] T007 Create root `.gitignore` (node_modules, dist, coverage, .env, .env.local, tmp/) and root `README.md` with the monorepo layout and the three `make`/`npm` entry points

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: The backend and web skeletons every story runs on. Copied from the marketplace's features 001 and 002, trimmed to this scope. Health is the proof path through every backend layer.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

### Backend skeleton

- [X] T008 Copy `backend/internal/config/config.go` from the marketplace and reduce `AppConfig` to the variables in T006 (drop S3, R2, Hatchet, ingest); add `WorkOSRedirectURI`, `WorkOSBaseURL`, `WebAppURL`, `SessionCookieKey`, `SessionMaxAgeSeconds`, `AuthRateLimitRPS`, `AuthRateLimitBurst` with `required,notEmpty` on the WorkOS and session values
- [X] T009 [P] Copy `backend/internal/logging/logging.go`, `backend/internal/common/constants.go`, `backend/internal/common/errors.go`, `backend/internal/common/status.go` (or wherever `HTTPStatus` lives in the marketplace), `backend/internal/interfaces/interfaces.go`; rename the module path; keep only `ErrUnauthorized`, `ErrUserNotFound`, `ErrValidation` sentinels and add `ErrInvalidState`, `ErrForbiddenOrigin`, `ErrProviderUnavailable` with status mappings 400/403/503
- [X] T010 [P] Copy `backend/internal/metrics/metrics.go` and `backend/internal/metrics/pool-collector.go`
- [X] T011 [P] Copy `backend/internal/middleware/request-logger.go`, `backend/internal/middleware/validate-request.go`, `backend/internal/middleware/otel.go`
- [X] T012 [P] Copy `backend/internal/models/health.go`, `backend/internal/stores/health-store.go`, `backend/internal/services/health-service.go`, `backend/internal/controllers/health-controller.go`, `backend/internal/routes/health-routes.go`
- [X] T013 [P] Copy `backend/internal/migrations/embed.go` and create `backend/internal/migrations/000001_users.sql` per data-model.md (table, `users_email_idx` on `lower(email)`, down drops both)
- [X] T014 [P] Create `backend/internal/stores/queries/users.sql` with sqlc queries `GetUser`, `UpsertUserIfNewer` (the exact statement in data-model.md), `DeleteUser`; run `make sqlc-generate` to produce `backend/internal/stores/sqlcgen/`
- [X] T015 Copy `backend/internal/boot/env.go`, `config.go`, `database.go`, `otel.go`, `sentry.go`, `http.go`, `app.go` from the marketplace; remove Hatchet, S3, R2, admin, onboarding, listing, dataset, order, suspension wiring from `app.go`; keep the hardening stack, central error handler, CORS from config, body limit, global limiter, health routes; create `backend/cmd/app/main.go` as `fx.New(boot.AppOptions()).Run()`
- [X] T016 Run `make oapi-generate` to produce `backend/internal/models/dto/api.gen.go` from `contracts/openapi.yaml`; confirm `go build ./...` passes
- [X] T017 Copy `backend/tests/integration/harness_test.go` from the marketplace; reduce `defaultEnv()` to T006's variables with fake WorkOS values; keep the testcontainers Postgres boot, the fx app boot on a random port, and the pause-container helper
- [X] T018 Copy `backend/tests/integration/health_test.go` (livez, readyz pass, readyz fails on paused container, hardening headers, 429 on global limiter, single error log carrying `request_id` and `trace_id`); run `make test` and confirm green

### Web skeleton

- [X] T019 [P] Create `web/src/lib/env.ts` with only `VITE_API_URL` (zod, fail fast) and `web/src/lib/env.test.ts`
- [X] T020 [P] Copy `web/src/app/root-error-boundary.tsx` and its test, `web/src/pages/not-found-page.tsx` and `web/src/app/not-found.integration.test.tsx`
- [X] T021 [P] Run `npm run generate:api` to produce `web/src/lib/api/schema.d.ts`; create `web/src/lib/api/client.ts` from the marketplace version minus the suspension hook, with `ApiError`, `fieldErrorsFrom`, `shouldRetry`, and a `credentials: 'omit'` default
- [X] T022 Create `web/src/lib/query-client.ts` (defaults: `staleTime` 30 s, `retry: shouldRetry`, no refetch on window focus) and `web/src/app/providers.tsx` with `QueryClientProvider` only
- [X] T023 Create `web/src/app/router.tsx` with the five routes from `contracts/ui-routes.md` using placeholder page components, and `web/src/test/msw.ts` with a `setupServer` and contract-conformant handlers for `/v1/auth/refresh`, `/v1/auth/logout`, `/v1/me`; wire MSW into `web/src/test/setup.ts`; run `npm run typecheck && npm run test` and confirm green

**Checkpoint**: `make test`, `make lint`, `npm run test`, `npm run lint`, `npm run typecheck` all pass with health and not-found covered. Foundation ready.

---

## Phase 3: User Story 1 - New visitor signs up and lands on the dashboard (Priority: P1) 🎯 MVP

**Goal**: Landing page → hosted registration → callback provisions the user and sets the cookie → SPA refreshes → dashboard greets by name and email.

**Independent Test**: Start on the landing page, register with a fresh email, arrive on `/dashboard` greeted by name, and see one `users` row.

### Tests for User Story 1

> Write these first. Each MUST fail before its implementation task.

- [X] T024 [P] [US1] Unit tests for the cookie seal/open and state token in `backend/internal/session/cookie_test.go` and `backend/internal/session/state_test.go`: round trip, unknown kid, bad tag, expired state, tampered state, `safeReturnTo` rules
- [X] T025 [P] [US1] Extend `backend/tests/integration/harness_test.go` with a fake WorkOS `httptest` server (per research R15): `POST /user_management/authenticate` handling `authorization_code` and `refresh_token` grants with rotation, `POST /user_management/sessions/revoke`, `GET /user_management/users/{id}`, and a JWKS endpoint; mint RS256 access tokens with `sub`, `sid`, `exp`; point `WORKOS_BASE_URL` and `WORKOS_JWKS_URL` at it
- [X] T026 [P] [US1] Integration tests in `backend/tests/integration/auth_test.go` for login and callback: `GET /v1/auth/login` answers 302 whose `Location` carries `client_id`, `redirect_uri`, `screen_hint`, and a verifiable `state`; `?screen=sign-up` sets `screen_hint=sign-up`; a bad `returnTo` becomes `/dashboard`; `GET /v1/auth/callback` with a valid code and state answers 302 to `WEB_APP_URL/callback?returnTo=…`, sets `sow_session` with `HttpOnly; SameSite=Strict; Path=/v1/auth`, and inserts the `users` row; missing code, tampered state, expired state, and provider rejection each redirect to `WEB_APP_URL/?error=<code>` with a clearing cookie and no row
- [X] T027 [P] [US1] Web integration test `web/src/features/auth/sign-up.integration.test.tsx`: `/sign-up` renders the product name and a "Create account" link whose `href` is `${VITE_API_URL}/v1/auth/login?screen=sign-up&returnTo=%2Fdashboard`, plus a link to `/`
- [X] T028 [P] [US1] Web integration test `web/src/features/auth/callback.integration.test.tsx`: `/callback?returnTo=/dashboard` calls refresh (MSW 200) and lands on the dashboard; MSW 401 lands on `/?error=flow_incomplete`
- [X] T029 [P] [US1] Web integration test `web/src/features/dashboard/dashboard.integration.test.tsx`: with an authenticated session store and MSW `/v1/me`, `/dashboard` shows the name, the email, "You're logged in", and a sign-out button; with an empty name it greets by email

### Implementation for User Story 1

- [X] T030 [P] [US1] Create `backend/internal/session/cookie.go` (AES-256-GCM seal/open, `kid` parsing of `SESSION_COOKIE_KEY`, `Seal`, `Open`, `Clear` producing `*http.Cookie` per `contracts/session-cookie.md`) and `backend/internal/session/state.go` (HMAC state issue/verify, `SafeReturnTo`)
- [X] T031 [P] [US1] Create `backend/internal/models/user.go`, `backend/internal/models/session.go` (cookie payload), and `backend/internal/models/dto/auth-dto.go` (login query DTO with `returnTo`, `screen`; callback query DTO with `code`, `state`) plus `UserFrom` converter in `backend/internal/models/dto/user-dto.go`
- [X] T032 [P] [US1] Create `backend/internal/services/workos-client.go`: a `WorkOSClient` interface (`AuthorizationURL`, `AuthenticateWithCode`, `AuthenticateWithRefreshToken`, `RevokeSession`, `GetUser`) and its SDK-backed implementation; `backend/internal/boot/workos.go` providing `*workos.Client` (`WithBaseURL` when `WORKOS_BASE_URL` set), the webhook verifier, and the auth middleware (from research R7)
- [X] T033 [US1] Create `backend/internal/stores/user-store.go` (`Get`, `UpsertIfNewer`, `Delete`, `WithTx`) over sqlcgen
- [X] T034 [US1] Create `backend/internal/services/user-service.go` with `UpsertFromProvider(ctx, workosUser)` and `Get(ctx, id)`
- [X] T035 [US1] Create `backend/internal/services/auth-service.go` with `LoginURL(returnTo, screen)`, `CompleteLogin(ctx, code, state, ip, ua) (cookie, returnTo, err)` that exchanges the code, upserts the user, and seals the cookie
- [X] T036 [US1] Create `backend/internal/controllers/auth-controller.go` with `Login` and `Callback` handlers (three-stanza shape; callback maps every error to the redirect table in `contracts/session-cookie.md`) and `backend/internal/routes/auth-routes.go` registering `GET /v1/auth/login` and `GET /v1/auth/callback` under a `/v1/auth` group with the stricter limiter from research R6 (`backend/internal/middleware/auth-rate-limit.go`); wire stores, services, controllers, and routes in `backend/internal/boot/app.go`
- [X] T037 [US1] Run `make test` for `auth_test.go` login and callback cases and `session` unit tests; confirm green
- [X] T038 [P] [US1] Create `web/src/stores/session.ts` (Zustand: `status`, `accessToken`, `expiresAt`, `setAuthenticated`, `setAnonymous`) and `web/src/lib/auth/session.ts` (`bootstrap`, `refresh` single-flight, `getAccessToken`, `signInUrl(screen, returnTo)`, `signOut`, refresh scheduling with the 60 s buffer) with unit tests in `web/src/lib/auth/session.test.ts`
- [X] T039 [P] [US1] Create `web/src/features/auth/return-target.ts` (copy `safeReturnTo`) with its test, and `web/src/features/auth/api.ts` (`useMe` via the generated client and `getAccessToken`)
- [X] T040 [US1] Create `web/src/features/auth/pages/sign-up-page.tsx`, `web/src/features/auth/pages/callback-page.tsx`, `web/src/features/auth/index.ts`; replace the router placeholders for `/sign-up` and `/callback`
- [X] T041 [US1] Create `web/src/features/dashboard/pages/dashboard-page.tsx` and `web/src/features/dashboard/index.ts`; replace the `/dashboard` placeholder (guard arrives in US3; for now the page renders when the store is authenticated)
- [X] T042 [US1] Write `marketing/index.html` and `marketing/styles.css` per `contracts/ui-routes.md` with the app origin in one `<base href>` tag; verify no horizontal scroll at 375 px in the preview browser
- [X] T043 [US1] Run the web suites for T027–T029 and confirm green

**Checkpoint**: A fresh registration through the fake WorkOS in tests, and through real WorkOS once credentials land, ends on a greeting dashboard with a `users` row.

---

## Phase 4: User Story 2 - Returning user signs in and stays signed in (Priority: P2)

**Goal**: Sign-in works, and the session survives reload, tab close, and access-token expiry through the API-owned refresh cookie, in every browser.

**Independent Test**: Sign in, reload, close and reopen, wait past token expiry; no credential prompt in Chrome incognito, Safari, Firefox.

### Tests for User Story 2

- [X] T044 [P] [US2] Integration tests in `backend/tests/integration/auth_test.go` for refresh: with a valid cookie and allowed `Origin`, `POST /v1/auth/refresh` answers 200 `{access_token, expires_at}`, the token validates against the JWKS, and the response re-seals a cookie holding the rotated refresh token; a second refresh with the new cookie succeeds; no cookie → 401 with a clearing cookie; provider rejection (fake returns 400) → 401 and clearing cookie; provider unreachable → 503 and the cookie is kept; missing `Origin` → 403 before the cookie is opened
- [X] T045 [P] [US2] Web integration test `web/src/features/auth/sign-in.integration.test.tsx`: `/` renders the "Sign in" link with `screen=sign-in` and the `redirect` param folded into `returnTo`; each `?error=` value shows its notice from `contracts/ui-routes.md`; an authenticated visitor on `/` is sent to the return target
- [X] T046 [P] [US2] Web integration test `web/src/features/auth/session-persistence.integration.test.tsx`: app load with MSW refresh 200 → dashboard renders without a sign-in page; a bearer call answered 401 triggers exactly one refresh and one retry; refresh 401 after that clears the session and shows `/?error=session_expired`
- [X] T047 [P] [US2] Unit tests in `web/src/lib/auth/session.test.ts` for scheduling: refresh fires 60 s before `expires_at`, never sooner than 5 s, concurrent `refresh()` calls share one request

### Implementation for User Story 2

- [X] T048 [US2] Add `Refresh(ctx, cookieValue) (accessToken, expiresAt, newCookie, err)` to `backend/internal/services/auth-service.go`, reading `exp` from the returned JWT; add `backend/internal/middleware/origin.go` (`RequireOrigin` against `CORS_ALLOWED_ORIGINS`); add the `Refresh` handler to `backend/internal/controllers/auth-controller.go` and register `POST /v1/auth/refresh` with `RequireOrigin` in `backend/internal/routes/auth-routes.go`; enable credentialed CORS on the `/v1/auth` group only
- [X] T049 [US2] Run `make test` for the refresh cases; confirm green
- [X] T050 [US2] Create `web/src/features/auth/pages/sign-in-page.tsx` with the error-notice table; replace the `/` placeholder
- [X] T051 [US2] Add the 401-refresh-retry middleware to `web/src/lib/api/client.ts` and the `session_expired` redirect to `web/src/lib/auth/session.ts`; call `bootstrap()` from `web/src/app/providers.tsx` on mount
- [X] T052 [US2] Run the web suites for T045–T047; confirm green

**Checkpoint**: Reload, tab reopen, and token expiry all keep the user signed in against the fake provider; the manual browser walk in quickstart.md steps 6 and 8 is ready to run once credentials land.

---

## Phase 5: User Story 3 - Signed-in state is enforced everywhere (Priority: P3)

**Goal**: Guarded routes, refused anonymous API calls, working sign-out, and foreign-origin refusal.

**Independent Test**: Anonymous `/dashboard` redirects with the destination preserved; `/v1/me` without a bearer answers 401; sign-out ends the session everywhere.

### Tests for User Story 3

- [X] T053 [P] [US3] Integration tests in `backend/tests/integration/me_test.go`: no header, malformed header, expired token, wrong-key token → 401 with no body data; valid token → 200 with the record; valid token and no row → the fake's `GET /user_management/users/{id}` is called and the row is provisioned
- [X] T054 [P] [US3] Integration tests in `backend/tests/integration/auth_test.go` for logout: with cookie and allowed `Origin` → 204, the fake records a revoke for the cookie's `sid`, and a clearing cookie is sent; without a cookie → 204; foreign `Origin` → 403; a refresh with the old cookie after logout → 401
- [X] T055 [P] [US3] Integration tests in `backend/tests/integration/auth_test.go` for the stricter limiter: bursts beyond `AUTH_RATE_LIMIT_BURST` on `/v1/auth/refresh` answer 429 while `/v1/me` under the global limit still answers
- [X] T056 [P] [US3] Web integration test `web/src/features/auth/guard.integration.test.tsx`: anonymous `/dashboard` → `/?redirect=%2Fdashboard`; the sign-in link then carries `returnTo=%2Fdashboard`; `unknown` status shows the loader; sign-out button posts logout with credentials, clears the store, writes the cross-tab sentinel, and navigates to `/`; a `storage` event on the sentinel from another tab clears the session

### Implementation for User Story 3

- [X] T057 [P] [US3] Create `backend/internal/middleware/auth.go` from the marketplace (bearer JWKS validation, `sub` → context and logger) without the superadmin and access-state claims; `UserID(ctx)` helper
- [X] T058 [US3] Add `Me(ctx, userID)` with catch-up to `backend/internal/services/user-service.go`; create `backend/internal/controllers/user-controller.go` and `backend/internal/routes/user-routes.go` for `GET /v1/me` behind `RequireSession`
- [X] T059 [US3] Add `Logout(ctx, cookieValue)` to `backend/internal/services/auth-service.go` (revoke, swallow-and-log provider failure) and the `Logout` handler and route with `RequireOrigin`
- [X] T060 [US3] Run `make test` for T053–T055; confirm green
- [X] T061 [US3] Create `web/src/features/auth/components/protected-layout.tsx` and `web/src/features/auth/components/sign-out-button.tsx`; nest `/dashboard` under `ProtectedLayout` in `web/src/app/router.tsx`; add the `storage` sentinel listener to `web/src/lib/auth/session.ts`
- [X] T062 [US3] Run the web suite for T056; confirm green

**Checkpoint**: Every protected surface refuses anonymous access, and sign-out is complete on both sides.

---

## Phase 6: User Story 4 - Operate a runnable, probeable foundation (Priority: P4)

**Goal**: Fail-fast config, webhook sync, migration smoke, CI gates, and deploy configuration on top of the Phase 2 skeleton.

**Independent Test**: Probes behave per `contracts/health-and-ops.md`, a blank required setting refuses startup naming it, `goose up/down/up` is clean, all three workflows pass.

### Tests for User Story 4

- [X] T063 [P] [US4] Integration test in `backend/tests/integration/config_test.go`: booting with `WORKOS_API_KEY` blank fails with an error naming `WORKOS_API_KEY`; same for `SESSION_COOKIE_KEY` and a malformed key entry
- [X] T064 [P] [US4] Integration tests in `backend/tests/integration/webhook_test.go` (copy from the marketplace, add `email`): signed `user.created` inserts; duplicate delivery is a no-op; stale `user.updated` changes nothing; newer `user.updated` changes name and email; tampered signature → 401 and no change; `user.deleted` removes the row; unknown event → 200

### Implementation for User Story 4

- [X] T065 [P] [US4] Create `backend/internal/models/dto/webhook-dto.go`, `backend/internal/controllers/webhook-controller.go` (raw-body signature verification), `backend/internal/routes/webhook-routes.go` for `POST /v1/webhooks/workos`; add `ApplyWebhookEvent` to `backend/internal/services/user-service.go`
- [X] T066 [P] [US4] Add `SESSION_COOKIE_KEY` format validation to `backend/internal/boot/config.go` (parse every `kid:base64` entry at boot; 32-byte keys only)
- [X] T067 [US4] Run `make test` for T063–T064; confirm green
- [X] T068 [P] [US4] Create `web/wrangler.jsonc` (assets from `dist`, SPA fallback) and `web/public/_headers` (CSP allowing `connect-src` to the API origin, HSTS, nosniff, frame-deny, referrer-policy)
- [X] T069 [P] [US4] Finish `.github/workflows/backend.yml` (build, golangci-lint, sqlc diff, oapi-codegen diff, tests with 85% gated coverage, govulncheck, migration up/down/up smoke against a Postgres service) and `.github/workflows/web.yml` (npm ci, generate:api diff, typecheck, lint, format:check, test:coverage, build, `wrangler deploy --dry-run`)
- [X] T070 [US4] Update `docs/deployment.md` with the final env variable tables and the `<base href>` note for `marketing/index.html`; update `backend/README.md` and `web/README.md` architecture sections

**Checkpoint**: All CI-equivalent commands in quickstart.md pass locally.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T071 Run the full gate set once: `cd backend && make lint && go build ./... && make test`; `cd web && npm run generate:api && git diff --exit-code src/lib/api/schema.d.ts && npm run typecheck && npm run lint && npm run format:check && npm run test:coverage && npm run build`; `cd backend && go tool oapi-codegen -config oapi-codegen.yaml ../contracts/openapi.yaml && git diff --exit-code internal/models/dto/api.gen.go`
- [X] T072 Check gocyclo (≤ 10) and comment voice across `backend/internal/` and `web/src/`; fix any function over the limit
- [X] T073 Look at `/`, `/sign-up`, `/dashboard` (with MSW-free real backend against the fake-free local stack once credentials exist) and `marketing/index.html` in the preview browser at desktop and 375 px; record screenshots on the feature's issue
- [X] T074 Pause for the environment hand-off: request `backend/.env` and `web/.env.local` values from the user, then run quickstart.md steps 1–9 and record outcomes on the issue
- [X] T075 Mark completed tasks `[X]`, commit, push `feature/001-auth-landing-dashboard`, open a PR to `main` naming the issues it closes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies. T002–T006 run in parallel after T001.
- **Foundational (Phase 2)**: depends on Phase 1. Backend tasks T009–T014 run in parallel after T008; T015 needs T008–T014; T016 needs T001 and T015; T017–T018 need T016. Web tasks T019–T021 run in parallel after T003; T022 needs T021; T023 needs T019–T022.
- **US1 (Phase 3)**: depends on Phase 2. Tests T024–T029 first, in parallel. T030–T032 in parallel; T033 → T034 → T035 → T036 → T037. T038–T039 in parallel; T040 → T041; T042 independent; T043 last.
- **US2 (Phase 4)**: depends on US1 (cookie, session client, callback). Tests T044–T047 in parallel. T048 → T049. T050–T051 → T052.
- **US3 (Phase 5)**: depends on US2 (refresh path for the 401 retry). Tests T053–T056 in parallel. T057 → T058 → T059 → T060. T061 → T062.
- **US4 (Phase 6)**: depends on Phase 2 for the skeleton and on US1 for the user store. Tests T063–T064 in parallel. T065–T066 in parallel → T067. T068–T069 in parallel. T070 last.
- **Polish (Phase 7)**: depends on all stories.

### User Story Dependencies

- **US1 (P1)**: Foundation only. Delivers the MVP loop.
- **US2 (P2)**: Builds on US1's cookie and session client.
- **US3 (P3)**: Builds on US2's refresh path.
- **US4 (P4)**: Mostly parallel with US2–US3 after US1 (webhook needs the user store).

### Parallel Opportunities

- Phase 1: T002, T003, T004, T005, T006 together.
- Phase 2: the whole backend group (T008–T018) alongside the whole web group (T019–T023).
- Phase 3: all six tests together; then T030, T031, T032 together; T038, T039, T042 together with the backend chain.
- Phase 4: T044–T047 together; T048–T049 alongside T050–T051.
- Phase 5: T053–T056 together; T057 alongside T061.
- Phase 6: T063–T064 together; T065, T066, T068, T069 together.

### Parallel Example: Phase 3 tests

```text
T024 backend/internal/session/{cookie,state}_test.go
T025 backend/tests/integration/harness_test.go (fake WorkOS)
T026 backend/tests/integration/auth_test.go (login, callback)
T027 web/src/features/auth/sign-up.integration.test.tsx
T028 web/src/features/auth/callback.integration.test.tsx
T029 web/src/features/dashboard/dashboard.integration.test.tsx
```

## Implementation Strategy

### MVP First (Phases 1–3)

1. Phases 1–2: the copied foundation, green on health and not-found.
2. Phase 3: registration through callback to a greeting dashboard.
3. **STOP and VALIDATE**: `make test` and `npm run test` green; the login redirect and callback verified against the fake provider; landing page looked at.

### Incremental Delivery

- US2 adds refresh and the sign-in page → the session survives everything.
- US3 adds the guard, `/me`, logout → enforcement complete.
- US4 adds webhook sync, config validation, deploy config, CI → operable.
- Phase 7 runs the gates once, looks at every page, and pauses for real credentials before the manual walk.

### Agent economy (constitution, Development Workflow)

- Copy and rename work (T002–T006, T009–T014, T019–T021, T068–T069): Sonnet or Haiku, low or medium effort.
- Session primitives, auth service, harness with fake WorkOS, session client (T025, T030, T032, T035, T038, T048): Opus at the effort each needs.
- Review agents read only; the orchestrating session owns the single full gate run in T071.
