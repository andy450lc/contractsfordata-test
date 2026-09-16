# Quickstart: First-Party Sign-In and Sign-Up Forms

**Feature**: 004-first-party-auth-forms · Validation guide (run after
implementation; see [plan.md](plan.md) for design, [spec.md](spec.md) for
acceptance criteria).

## Prerequisites

- Everything from `specs/001-auth-landing-dashboard/quickstart.md`
  (Docker, Node 22, `backend/.env`, `web/.env.local`, migrations run).
- WorkOS dashboard, SoW staging environment (step 1 below).
- For the real-email checks: a Resend API key with a verified sending
  domain, or the account owner's address as the only recipient. Local
  runs without `RESEND_API_KEY` write the emails to the API log instead.

## 1. WorkOS dashboard checklist (once per environment)

1. Authentication → Email + Password: **on**. Require email
   verification: **on**.
2. Authentication → Google OAuth: **on** (WorkOS test credentials are
   fine for staging).
3. Redirects: keep `http://localhost:8080/v1/auth/callback` (and the
   deployed API callback). Set the **password reset** URL to
   `http://localhost:5173/reset-password` (deployed:
   `https://app.<stage>.<domain>/reset-password`). Set the app homepage
   to the web origin.
4. Emails → Email verification: sent by WorkOS (default).

## 2. Automated validation (CI-equivalent)

```bash
cd backend && make lint && go build ./... && make test
```

```bash
cd web && npm run generate:api && git diff --exit-code src/lib/api/schema.d.ts && npm run typecheck && npm run lint && npm run format:check && npm run test:coverage && npm run build
```

```bash
cd backend && make oapi-generate && git diff --exit-code internal/models/dto/api.gen.go
```

| Suite | Proves |
|---|---|
| `backend/tests/integration/auth_headless_test.go` | US1, US2, US4, FR-006–FR-012, FR-015–FR-018: sign-in sets the cookie and returns a token; wrong password and unknown email answer the identical 401 body; unverified email answers 202 with the provider's pending token; verify-email with the right code sets the cookie; wrong code answers 400 `invalid_code`; sign-up creates the user with names and password and answers 202; duplicate sign-up answers the same 202, creates nothing, and the fake mailer holds the "account exists" email; weak password answers 422 with the provider message; resend-verification and forgot-password answer 202 for known and unknown addresses; forgot-password for a known address puts a reset email with the web reset link in the fake mailer; reset-password with a valid token sets the password, signs in, sets the cookie; used token answers 400; every endpoint refuses a foreign Origin with 403 and is under the auth limiter |
| `backend/tests/integration/auth_test.go` | FR-013: `/auth/login?provider=google` redirects to the provider's authorize endpoint with `provider=GoogleOAuth` and no `screen_hint`; missing or unknown provider answers 400; callback, refresh, logout unchanged |
| `web` auth integration tests | US1–US4 pages per `contracts/ui-routes.md`: form validation, request bodies observed by MSW, 200 adopts the session and lands on the destination, 202 shows the code step, resend cooldown, invalid code notice, Google button href, forgot-password neutral confirmation, reset page strips the token from the URL and handles each outcome, signed-in redirect on every page |

## 3. Manual walk (Stories 1–4, local)

Start the API and the web app (`make dev` in `backend/`, `npm run dev`
in `web/`). No `RESEND_API_KEY` locally, so watch the API log for the
two platform emails.

1. **Sign up**: open `http://localhost:5173/sign-up`, fill first name,
   last name, a fresh email you can read, and a password. Expect the
   "Check your inbox" step. Read the 6-digit code from the WorkOS email,
   enter it. Expect `/dashboard` greeting by first name. The address bar
   never left `localhost:5173`. Confirm a `users` row exists.
2. **Sign out, sign in**: sign out, sign in with the same email and
   password. Expect the dashboard in one submission.
3. **Wrong password, unknown email**: try both. Expect the same alert
   text and no difference visible.
4. **Unverified sign-in**: sign up a second fresh email, close the tab
   at the code step, then sign in with that email and password. Expect
   the code step, and the code to complete sign-in.
5. **Duplicate sign-up**: sign up again with the first email. Expect
   the same "Check your inbox" step. In the API log, expect the
   "account exists" email for that address. Entering any code fails with
   the invalid-code notice.
6. **Google**: on `/`, click "Continue with Google". Expect the next
   page to be Google's (no AuthKit chooser). Complete it with a Google
   account whose email matches the first user. Expect `/dashboard` as
   that same user (same `users.id`). Then repeat with a Google account
   that has no SoW user. Expect a new user with Google's name and no
   code step.
7. **Forgot password**: sign out, open `/forgot-password`, submit the
   first email. Expect the neutral confirmation. Copy the reset link
   from the API log, open it. Expect the address bar to lose `?token=`
   right away. Set a new password. Expect `/dashboard`. Sign out and
   confirm the old password fails and the new one works. Open the same
   link again. Expect "This link is no longer valid".
8. **Old accounts**: sign in with an account created under Feature 001
   (hosted page). Expect it to work with no extra step.

## 4. Deployed checks (once a stage exists)

- Repeat steps 1, 6, and 7 against the stage with `RESEND_API_KEY` set.
  Emails arrive from `EMAIL_FROM`.
- Cloudflare: from one machine, send 15 sign-in attempts in a minute.
  Expect the 11th onward to be refused by Cloudflare (a Cloudflare error
  page or 429 with `cf-mitigated` header), never reaching the API log.
- Browser history across the whole walk shows only the web origin and
  Google hosts (SC-001).
