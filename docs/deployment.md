# Deployment

Decided 2026-09-03: SoW deploys on **Railway** (backend), **Supabase**
(Postgres), and **Cloudflare** (web app, landing page, R2 document
storage, DNS). There is no infrastructure repo and no IaC; platform
configuration lives in the Railway, Supabase, and Cloudflare dashboards,
and this document is the record of it.

> Values marked `TODO(fill-in)` are configured by hand in the dashboards
> and should be filled in here so the setup stays reproducible.

## Topology

| Piece | Where | Hostname |
|---|---|---|
| Go backend | Railway service, built from `backend/` | `api.<stage>.<domain>` |
| Web SPA | Cloudflare Workers static assets, built from `web/` | `app.<stage>.<domain>` |
| Landing site | Cloudflare Workers static assets, built from `marketing/` (Astro) | `<domain>` (prod only) |
| Postgres | Supabase project, one per stage | via `DATABASE_URL` |
| Document storage | Cloudflare R2, one bucket per stage | via `R2_DOCUMENTS_BUCKET` |
| Transactional email | Resend | via `RESEND_API_KEY`, `EMAIL_FROM` |
| DNS | Cloudflare zone `<domain>` | TODO(fill-in): domain |

Stage names the backend accepts: `dev`, `staging`, `prod` (`STAGE` env,
validated in `backend/internal/config/config.go`).

## Backend (Railway)

- **Root directory**: `backend` (monorepo scoping).
- **Build**: Railpack auto-detects Go; the explicit equivalent is
  `go build -o app ./cmd/app`, start command `./app`.
- **Port**: Railway injects `PORT`; the app reads it (default 8080).
- **Watch paths**: `backend/**` and `contracts/**`, so web-only merges
  don't redeploy the backend.
- **Health checks**: `GET /livez` (process up) and `GET /readyz`
  (dependencies, including the database). Point Railway's healthcheck at
  `/readyz` so a lost database fails the deploy instead of serving
  errors.
- **Deploys**: automatic on push to `main` via Railway's GitHub
  connection. Manual fallback: redeploy from the Railway dashboard, or
  `railway up` from `backend/` with the Railway CLI.

### Environment variables (Railway service)

The full schema lives in `backend/internal/config/config.go`; that file
is the source of truth. Required in a deployed environment:

| Variable | Notes |
|---|---|
| `STAGE` | `dev` / `staging` / `prod` |
| `DATABASE_URL` | Supabase Postgres DSN. Secret. See the pooler note below |
| `WORKOS_API_KEY` | Secret |
| `WORKOS_CLIENT_ID` | |
| `WORKOS_WEBHOOK_SECRET` | Secret |
| `WORKOS_REDIRECT_URI` | `https://api.<stage>.<domain>/v1/auth/callback`; registered in the WorkOS dashboard |
| `WEB_APP_URL` | `https://app.<stage>.<domain>`; the callback redirects here |
| `SESSION_COOKIE_KEY` | 32-byte key for the encrypted refresh cookie. Secret |
| `CORS_ALLOWED_ORIGINS` | Comma-separated; must include `https://app.<stage>.<domain>` and the marketing origin that requests the public blank template |
| `R2_ACCOUNT_ID` | Cloudflare account holding the documents bucket. TODO(fill-in) |
| `R2_DOCUMENTS_BUCKET` | R2 bucket name. TODO(fill-in) |
| `R2_ACCESS_KEY_ID` / `R2_SECRET_ACCESS_KEY` | R2 S3-API keys scoped to the one bucket. The API signs presigned URLs with them. Secret |
| `RESEND_API_KEY` | Resend API key. Required in a deployed environment. Secret |
| `EMAIL_FROM` | Verified Resend sender for authentication emails and Word-template attachments. Required in a deployed environment |
| `MAGIC_LINK_BASE_URL` | `https://app.<stage>.<domain>`; signer links are built on it |
| `SENTRY_DSN` | Required only when `STAGE=prod` |

Optional tuning: `DB_POOL_MAX_CONNS`, `DB_POOL_MIN_CONNS`,
`RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`, `SHUTDOWN_GRACE_SECONDS`,
`WORKOS_JWKS_URL`, `OTEL_EXPORTER_OTLP_ENDPOINT`.

Secrets exist only in Railway's environment configuration — never in the
repo, per the constitution. Rotation = update the variable in Railway,
which restarts the service.

### Supabase Postgres and the pooler

Supabase exposes three connection strings per project. Use them as
follows:

- **Session pooler** (port 5432 on the pooler host) for the API's
  `DATABASE_URL`. pgx v5 caches prepared statements by default, and the
  transaction-mode pooler rejects them. Session mode keeps pgx's
  defaults working. If connection counts ever force transaction mode
  (port 6543), set `default_query_exec_mode=simple_protocol` on the DSN
  and re-run the integration suite before deploying.
- **Direct connection** for migrations and one-off admin work from a
  laptop.
- Keep `DB_POOL_MAX_CONNS` below the Supabase plan's pooler client limit
  across all replicas.

Supabase Auth, Storage, and Realtime are not used. Postgres is the only
Supabase product in this stack; the identity provider is WorkOS and
documents live in R2.

### Database migrations

Migrations are goose files in `backend/internal/migrations/` and do not
run automatically on deploy. Either:

- run them from a laptop against the real database:
  `DATABASE_URL=<direct dsn> make migrate-up` (from `backend/`), or
- set Railway's **pre-deploy command** to
  `go tool goose -dir internal/migrations postgres "$DATABASE_URL" up`
  so every deploy migrates first. TODO(fill-in): which of the two is
  currently in effect.

`make migrate-down` is guarded to `STAGE=dev` — destructive rollbacks
never run against deployed stages by accident.

## Web (Cloudflare Workers static assets)

- **Root directory**: `web`.
- **Build command**: `npm run build` (`tsc -b && vite build`).
- **Config**: `web/wrangler.jsonc` declares
  `assets: { directory: "dist", not_found_handling: "single-page-application" }`
  and the custom domain. Security headers (CSP, HSTS, nosniff) live in
  `web/public/_headers` and are versioned with the app. The CSP allows
  `frame-src blob:` so the app can show a PDF it rendered in the browser
  inside an iframe. The CSP
  `connect-src` names the API origin, which differs per stage, so the
  checked-in file holds the token `__API_ORIGIN__` and `vite build`
  replaces it in `dist/_headers` with the origin of `VITE_API_URL`. The
  build fails if `VITE_API_URL` is unset or not an absolute URL. Configured
  Word-template delivery uses that API origin and keeps acknowledgment,
  contract values, and recipient addresses in JSON request bodies.
- **Deploys**: `wrangler deploy` from the `web.yml` workflow on push to
  `main`, using a Cloudflare API token scoped to Workers. TODO(fill-in):
  token name and the account id it is stored under in GitHub secrets.

### Build-time variables (web workflow)

Vite inlines `VITE_`-prefixed variables into the bundle, so none of them
may be a secret (documented in `web/.env.example`):

| Variable | Deployed value |
|---|---|
| `VITE_API_URL` | `https://api.<stage>.<domain>` (origin only, no trailing slash) |

The web app has no WorkOS configuration. Login, callback, refresh, and
logout all go through the API (constitution, Authentication).

## Landing site (Cloudflare Workers static assets)

- **Root directory**: `marketing`. Build: `npm ci && npm run build`
  (Astro, static output to `dist/`).
- **Build-time env vars**, both required and validated (a missing or
  relative value fails the build):
  - `PUBLIC_SITE_URL`: the site's own origin (`https://<domain>`), used
    for canonical URLs, Open Graph URLs, the sitemap, and `robots.txt`.
  - `PUBLIC_APP_ORIGIN`: the web app's origin
    (`https://app.<stage>.<domain>`), stamped into the sign-in and
    sign-up links. TODO(fill-in): the production values.
  - `PUBLIC_TEMPLATE_DELIVERY_URL`: the API collection base, normally
    `https://api.<stage>.<domain>/v1/public/sow-configurator`. The public
    configurator origin is stamped into CSP `connect-src`.
  - `PUBLIC_CONFIGURATOR_EMAIL_ENABLED`: `false` until a real provider-backed
    Word attachment send has passed end-to-end verification. Set `true` only
    after that check and sender-domain verification.
- **Config**: `marketing/wrangler.jsonc` with
  `assets: { directory: "dist", not_found_handling: "404-page" }` and the
  apex custom domain. `marketing/public/_headers` carries the CSP:
  `script-src 'self'` with no inline scripts, and a `connect-src` token
  the build replaces.
- **Template delivery**: every public template trigger opens the legal gate.
  After acknowledgment the browser posts to `/download` for a generated DOCX
  response or `/email` for the same generated DOCX as an attachment. There is
  no public static contract file.
- **Deploys**: `wrangler deploy` from `marketing.yml` on push to `main`.
  Only `prod` exists for the landing page.

## R2 (documents)

- One bucket per stage, private. Objects are keyed by agreement id and
  document version. Nothing is served from a public bucket URL.
- The API mints short-lived presigned GET URLs for viewing and presigned
  PUT URLs for any direct upload, using the scoped S3 keys above.
- Object lifecycle rules are set per the retention decided in the
  governing feature spec. TODO(fill-in): the configured rules.

## DNS & TLS (Cloudflare)

Records on the `<domain>` zone, per stage:

| Record | Type | Target | Proxy |
|---|---|---|---|
| `<domain>` (apex) | Workers custom domain | marketing Worker | Managed by Cloudflare |
| `app.<stage>` | Workers custom domain | web Worker | Managed by Cloudflare |
| `api.<stage>` | CNAME | Railway service domain | DNS-only (grey cloud) so Railway issues and renews the certificate |

Hostnames deeper than one label below the apex (`app.<stage>.<domain>`)
sit outside Cloudflare's universal certificate. Workers custom domains
provision their own certificates, and the API record is DNS-only, so no
Advanced Certificate Manager purchase is needed today. TODO(fill-in):
confirm once the domain is chosen.

## WorkOS

The hosted AuthKit page is not used: sign-in and sign-up are first-party
forms in the web app that call WorkOS's headless User Management API
through the backend, and "Continue with Google" is the one direct
navigation to WorkOS. Deployed auth breaks unless the WorkOS dashboard is
set up per stage:

- Authentication → Email + Password: **on**. Require email verification:
  **on**.
- Authentication → Google OAuth: **on**.
- Redirects: `https://api.<stage>.<domain>/v1/auth/callback` (the API,
  not the web app).
- Password reset URL: `https://app.<stage>.<domain>/reset-password`.
- App homepage / logout return URLs: `https://app.<stage>.<domain>`.
- Backend `CORS_ALLOWED_ORIGINS` includes the web origin (backend side,
  set in Railway).

No WorkOS custom domain is needed. The API owns the refresh cookie, so
silent refresh is first-party on every stage, including the free WorkOS
staging environment used for `dev` and `staging`.

## Cloudflare rate limiting

Two rate-limiting rules on the API zone protect the `/v1/auth` credential
endpoints in front of the API's own per-IP limiter:

| Rule | Match | Characteristic | Threshold | Action |
|---|---|---|---|---|
| General auth throttle | `http.request.method eq "POST" and starts_with(http.request.uri.path, "/v1/auth/")` | `ip.src` | 10 requests / 1 minute | Block for 10 minutes |
| Credential-endpoint challenge | same, restricted to `sign-in`, `sign-up`, `forgot-password`, `resend-verification` | `ip.src` | 5 requests / 1 minute | Managed Challenge |

These thresholds are a starting point, recorded here so they can be
tuned without spelunking the dashboard. The API's `AUTH_RATE_LIMIT_*`
per-IP limiter is the second line of defense; the web app's resend
cooldown is a UX guard only, not a security control.

## Standing up a new stage

1. Supabase: a new project for the stage. Record the session-pooler DSN
   and the direct DSN.
2. Railway: new environment (or service) for `backend/`, same watch
   paths; set every variable from the table above with stage values.
3. Run migrations against the stage database with the direct DSN.
4. Cloudflare R2: a documents bucket for the stage, plus S3-API keys
   scoped to it.
5. Resend: verify a sending domain (or use Resend's restricted test sender for
   an owner-only non-production check), create an API key with sending access,
   and set `RESEND_API_KEY` plus a verified `EMAIL_FROM` in Railway. Production
   template-email success is unavailable until Resend accepts that sender.
6. Cloudflare Workers: deploy `web/` with `VITE_API_URL` pointed at the
   stage. Deploy `marketing/` with `PUBLIC_TEMPLATE_DELIVERY_URL` pointed at
   the API delivery base. Attach the intended custom domains.
7. Cloudflare DNS: the `api.<stage>` CNAME.
8. WorkOS: register the stage's redirect URI and complete the dashboard
   checklist above (password auth, email verification, Google OAuth,
   password-reset URL).
9. Cloudflare: add the two rate-limiting rules above, scoped to the
   stage's API hostname.
10. Verify: `https://api.<stage>.<domain>/readyz` returns 200, then sign
    up, create an agreement, send it to a test inbox, and sign it
    end-to-end on `https://app.<stage>.<domain>`.
