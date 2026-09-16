# Research: Sign-Up, Sign-In, Landing Page, and Dashboard

**Date**: 2026-09-03 · **Feature**: 001-auth-landing-dashboard

No NEEDS CLARIFICATION markers remained after Technical Context. The
constitution (1.1.0) and the binding convention docs fix the stack. The
decisions below cover what required judgment inside those constraints,
plus the mechanics of the token-mediating backend, verified against the
installed `workos-go/v10` v10.3.0 sources and the WorkOS documentation on
2026-09-03. Where the marketplace repository already resolved a question
(its features 001–003), the decision is carried over and the source is
named.

## R1. Foundation is copied from the marketplace, then trimmed

- **Decision**: Copy the marketplace's `backend/` foundation (boot,
  config, common, logging, metrics, middleware request-logger and
  validate-request, health vertical, migrations embed, integration
  harness, Makefile, `.golangci.yml`, `.air.toml`, `sqlc.yaml`,
  `oapi-codegen.yaml`, CI workflow) and its `web/` toolchain (Vite,
  TypeScript, ESLint, Prettier, Tailwind, shadcn config, Vitest setup,
  `render.tsx`, `root-error-boundary.tsx`, `not-found-page.tsx`,
  `full-page-loader.tsx`, `lib/env.ts`, `lib/api/client.ts` minus the
  suspension hook, CI workflow). Rename the module path to
  `github.com/pixels-two/sow/backend` and the product strings. Leave out
  everything tied to marketplace features: Hatchet, S3, R2, admin,
  onboarding, suspension, listings, datasets, orders.
- **Rationale**: The user asked for it, the constitution and conventions
  are shared, and the marketplace code already passes the same lint,
  coverage, and structure gates. Copying beats re-deriving.
- **Alternatives considered**: Scaffold fresh (rejected: slower and
  reintroduces the mistakes the marketplace already fixed, such as the
  metrics status-on-error bug and the fx/middleware import cycle).

## R2. Token-mediating backend: endpoint set and flow

- **Decision**: Four operations under `/v1/auth`:
  - `GET /v1/auth/login?returnTo=&screen=sign-in|sign-up` builds the
    AuthKit authorization URL with `GetAuthorizationURL` (`Provider:
    authkit`, `ScreenHint`, `RedirectURI` = `WORKOS_REDIRECT_URI`, `State`
    = signed state token, see R4) and answers `302`.
  - `GET /v1/auth/callback?code=&state=` verifies the state, calls
    `AuthenticateWithCode` with the API key, upserts the user from the
    response, seals the refresh token and session id into the cookie (R3),
    and answers `302` to `WEB_APP_URL/callback?returnTo=<path>`. Any
    failure answers `302` to `WEB_APP_URL/?error=<code>` with the cookie
    cleared and no session.
  - `POST /v1/auth/refresh` (cookie, `Origin` checked) opens the cookie,
    calls `AuthenticateWithRefreshToken`, re-seals the rotated refresh
    token, and returns `200 {access_token, expires_at}`. Any WorkOS
    rejection answers `401` and clears the cookie.
  - `POST /v1/auth/logout` (cookie, `Origin` checked) calls
    `RevokeSession` with the stored session id, clears the cookie, and
    answers `204`. Revocation failure still clears the cookie and
    answers `204`; the failure is logged.
- **Rationale**: Constitution Authentication section. `screen` on login
  lets the sign-up page open AuthKit on its registration screen, matching
  the marketplace's `signUp()` behavior. Redirecting to the SPA's
  `/callback` rather than straight to the destination lets the SPA run its
  first refresh and land with an access token already in memory.
- **Alternatives considered**: The SDK's sealed-session helpers
  (`SealSessionFromAuthResponse`, `RefreshSession`). They seal access
  token, refresh token, and the whole user object under one password with
  no key id, so rotation is a hard cutover and the cookie carries PII the
  cookie does not need. Rejected in favor of a small AES-GCM seal owned
  by the project (R3). PKCE on top of the confidential client: harmless
  but redundant when the client secret is already required at exchange;
  omitted (Principle IV).

## R3. Refresh cookie format

- **Decision**: Cookie `sow_session` (name constant in `common`). Value
  `v1.<kid>.<base64url(nonce || ciphertext)>` where the plaintext is JSON
  `{"rt": refresh token, "sid": session id, "uid": user id, "iat": unix
  ms}` sealed with AES-256-GCM. `SESSION_COOKIE_KEY` holds one or more
  `kid:base64key` pairs separated by commas; the first seals, all open.
  Attributes: `HttpOnly; SameSite=Strict; Path=/v1/auth; Max-Age=
  SESSION_MAX_AGE_SECONDS` (default 604800, seven days); `Secure` in
  staging and prod, omitted in `dev` so plain-http localhost works in
  every browser. No `Domain` attribute: host-only on the API host.
- **Rationale**: Constitution III and backend conventions §10. The key id
  gives zero-downtime rotation. `Path=/v1/auth` keeps the cookie off every
  other request. `SameSite=Strict` is safe because the SPA and the API
  are same-site and the only cross-site arrival is the WorkOS callback,
  which sets the cookie rather than reading it. `uid` and `sid` in the
  cookie let logout and logging work without decoding the access token.
- **Alternatives considered**: Storing sessions server-side in Postgres
  (rejected: a table, a cleanup job, and a DB hit per refresh for no
  gain at this scale); `__Host-` prefix (rejected: it requires `Path=/`,
  which conflicts with the narrower path).

## R4. State token and returnTo

- **Decision**: `state` is `base64url(json{nonce, returnTo, exp}) + "." +
  base64url(HMAC-SHA256(payload, key))`, keyed with the active
  `SESSION_COOKIE_KEY`, with `exp` ten minutes out. `returnTo` is
  validated with the marketplace's `safeReturnTo` rule (same-app absolute
  path, no `//`, no backslash) before it is signed and again after it is
  verified. The API forwards the validated path to the SPA on the
  callback redirect as a query parameter.
- **Rationale**: Spec edge case "return leg arrives with a missing,
  expired, or tampered state". A signed stateless token needs no storage
  and cannot be replayed usefully after `exp`. Reusing the cookie key
  keeps the config surface to one secret.
- **Alternatives considered**: A `state` nonce stored in a short-lived
  cookie (rejected: a second cookie and a `SameSite` subtlety on the
  cross-site return; the HMAC gives the same guarantee).

## R5. Origin enforcement on cookie endpoints

- **Decision**: `middleware.RequireOrigin` rejects `POST /v1/auth/refresh`
  and `POST /v1/auth/logout` with `403 {"error":"forbidden_origin"}`
  unless the `Origin` header exactly matches an entry in
  `CORS_ALLOWED_ORIGINS`. CORS on the `/v1/auth` group allows credentials
  for those origins only. `GET /v1/auth/login` and `/callback` are
  browser navigations and carry no `Origin`; they are protected by
  `returnTo` validation and the state signature instead.
- **Rationale**: Constitution Authentication item 6; spec US3 scenario 5.
  Browsers always send `Origin` on cross-origin `POST` and on same-origin
  `fetch` with a method other than GET, so a missing header is itself a
  rejection.
- **Alternatives considered**: A CSRF token in a header (rejected: the
  SPA never has a secret to put there; `Origin` plus `SameSite=Strict`
  is the standard defense for this shape).

## R6. Stricter rate limit on `/v1/auth`

- **Decision**: A second echo `RateLimiter` instance on the `/v1/auth`
  group, per IP, `AUTH_RATE_LIMIT_RPS` (default 2) and
  `AUTH_RATE_LIMIT_BURST` (default 10), on top of the global limiter.
- **Rationale**: Constitution III ("rate limited more strictly than
  account routes"); the endpoints call WorkOS on every hit.
- **Alternatives considered**: Per-user limiter (rejected: refresh and
  login have no user yet).

## R7. Access token validation

- **Decision**: Copy the marketplace `middleware/auth.go` (jwx v3
  `jwk.Cache` over `https://api.workos.com/sso/jwks/<client id>`, or
  `WORKOS_JWKS_URL` when set) minus its superadmin and access-state
  claims. `sub` becomes the user id on the context and in the logger.
- **Rationale**: Unchanged from the marketplace and the constitution
  (item 4: no database lookup on the hot path).
- **Alternatives considered**: None; this is settled.

## R8. Access token lifetime and client refresh policy

- **Decision**: WorkOS access tokens expire per the environment's session
  settings (default five minutes). The refresh response carries
  `expires_at` (Unix ms) read from the token's `exp` claim. The SPA
  schedules the next refresh 60 seconds before `expires_at` (never sooner
  than 5 seconds out), runs at most one refresh at a time, and on any 401
  from a bearer call performs one refresh and retries once. Refresh
  failure with 401 clears the in-memory session and routes to sign-in.
- **Rationale**: Spec US2 scenario 5 and 6; frontend conventions §9.
  Reading `exp` from the token avoids a second config value that could
  drift from the WorkOS dashboard.
- **Alternatives considered**: Refresh on every navigation (rejected: a
  WorkOS call per page).

## R9. Concurrent tabs

- **Decision**: Each tab refreshes independently. WorkOS refresh tokens
  rotate, and a second tab presenting the previous token inside the
  provider's grace window is accepted; the API re-seals whatever token
  WorkOS returns. The web session client also listens for a `storage`
  event on a sentinel key to trigger a refresh after another tab signs
  out, so a signed-out tab does not keep a live token past its expiry.
- **Rationale**: Spec edge case "two tabs renew at the same moment".
  WorkOS documents refresh-token reuse tolerance; the design leans on it
  rather than adding a cross-tab lock. The `storage` sentinel carries no
  token, only a timestamp, so it does not violate the in-memory rule.
- **Alternatives considered**: Web Locks API to serialize refreshes
  across tabs (rejected: the cookie is shared anyway, and the lock only
  matters if WorkOS rejected the second refresh, which it does not
  within its grace window). Verified during manual walk in quickstart.

## R10. User provisioning and sync

- **Decision**: The callback upserts the user from
  `AuthenticateResponse.User` (id, email, name from `first_name` +
  `last_name`, provider `created_at`/`updated_at`) through the same
  order-safe upsert the webhook uses. `POST /v1/webhooks/workos` (copied
  from the marketplace, with `email` added) handles `user.created`,
  `user.updated`, `user.deleted`. `GET /v1/me` falls back to a WorkOS
  `GetUser` call when the row is missing.
- **Rationale**: Provisioning at callback closes the sign-up lag window
  entirely for the normal path; the webhook keeps later profile changes
  and deletions in sync; the catch-up covers a row deleted by a stale
  webhook race. Spec FR-007, FR-008, edge cases one to three.
- **Alternatives considered**: Webhook-only provisioning (the
  marketplace's initial approach; rejected because the callback already
  has the user object in hand).

## R11. SPA session client without a vendor SDK

- **Decision**: `stores/session.ts` (Zustand) holds `status:
  'unknown' | 'authenticated' | 'anonymous'`, `accessToken`, `expiresAt`.
  `lib/auth/session.ts` exposes `bootstrap()` (one refresh on app load),
  `refresh()` (single-flight), `getAccessToken()` (refreshes when within
  the buffer), `signOut()`, and `signInUrl(screen, returnTo)`. The API
  client injects the bearer header and retries once after a refresh on
  401. `ProtectedLayout` shows the loader while `unknown`, redirects with
  `?redirect=` while `anonymous`, renders children while
  `authenticated`. Sign-in and sign-up pages are plain links to the
  API's login endpoint. `/callback` calls `refresh()` then navigates to
  `safeReturnTo(returnTo)` or to `/?error=flow_incomplete`.
- **Rationale**: Constitution Authentication item 7 bans the browser SDK.
  The state lives in the first tier that fits (frontend conventions §4):
  the token is genuinely global client state.
- **Alternatives considered**: TanStack Query for the token (rejected:
  the token is not server data to cache; it is credential state with a
  scheduler).

## R12. Landing page

- **Decision**: `marketing/index.html` with `styles.css`: product name,
  one paragraph, "Sign in" linking to the app's `/` and "Get started"
  linking to `/sign-up`. The app origin appears once, in a `<base
  href>` tag at the top of the file, so a stage change is one edit.
  Deployed as Workers static assets with `not_found_handling:
  "404-page"`. No script, no analytics, no forms.
- **Rationale**: Spec FR-001, SC-003, the user's "static page" choice.
- **Alternatives considered**: Astro (rejected: one page needs no build).

## R13. Web hosting config

- **Decision**: `web/wrangler.jsonc` with `assets: { directory: "dist",
  not_found_handling: "single-page-application" }` and `web/public/_headers`
  carrying CSP (`default-src 'self'; connect-src 'self' <API origin>`),
  HSTS, nosniff, frame-deny, referrer policy. `wrangler` is a dev
  dependency; deploys are a CI step, not part of this feature's tests.
- **Rationale**: Frontend conventions §10 and §12; deployment doc.
- **Alternatives considered**: Cloudflare Pages (rejected: Workers
  static assets is the current recommended path and the deployment doc
  already names it).

## R14. Contract-first with both codegen gates

- **Decision**: `contracts/openapi.yaml` is written now with six
  operations. Backend: `oapi-codegen` types-only into
  `internal/models/dto/api.gen.go` with a CI diff gate. Web:
  `openapi-typescript` into `src/lib/api/schema.d.ts` with a CI diff
  gate. Hand-written DTOs remain only for multi-source binding wrappers
  (the callback query params).
- **Rationale**: Constitution V. The marketplace deferred the backend
  gate and wired it later; starting with it avoids the migration.
- **Alternatives considered**: Server-interface generation (rejected:
  no echo v5 server generator; routes stay hand-written per conventions).

## R15. Backend test seams

- **Decision**: The integration harness boots the real fx graph with
  `WORKOS_BASE_URL` pointed at a fake WorkOS `httptest` server (served
  through `workos.WithBaseURL`) that implements the code exchange, the
  refresh grant with rotation, session revoke, `GET /user_management/
  users/{id}`, and a JWKS endpoint serving a test RSA key. Access tokens
  the fake mints are real RS256 JWTs with `sub`, `sid`, `exp`. The
  harness also computes webhook signatures with the test secret.
- **Rationale**: Constitution II (integration over unit; vendors mocked
  only at the client boundary). One fake server proves the whole cookie
  flow: login redirect → callback → cookie → refresh → bearer → `/me` →
  logout, over HTTP against real Postgres.
- **Alternatives considered**: Mocking the `WorkOSClient` interface in
  Go (kept for the unit tests of `auth-service` error branches only).

## R16. Toolchain versions

- **Decision**: Go `1.25` in go.mod (local toolchain 1.25.3,
  `GOTOOLCHAIN=auto` upgrades if a dependency demands it). Node 22 in
  `.nvmrc`, npm 10 in `packageManager`. Docker Desktop for testcontainers
  and the local Postgres compose file. `wrangler` installed locally in
  `web/` and `marketing/`, not globally.
- **Rationale**: Matches what is installed on the development machine
  today; pinned files make it reproducible.
- **Alternatives considered**: Go 1.26 like the marketplace (rejected:
  not installed locally; auto-toolchain would download it on every
  fresh machine for no feature benefit).
