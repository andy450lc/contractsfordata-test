# Implementation Plan: First-Party Sign-In and Sign-Up Forms

**Branch**: `feature/004-first-party-auth-forms` | **Date**: 2026-09-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/004-first-party-auth-forms/spec.md`

## Summary

Replace the hosted AuthKit page with the web app's own sign-in, sign-up,
verification-code, forgot-password, and reset-password screens. The Go
API gains six JSON endpoints under `/v1/auth` that wrap WorkOS User
Management through the already-vendored `workos-go/v10` SDK
(`AuthenticateWithPassword`, `Create`, `AuthenticateWithEmailVerification`,
`SendVerificationEmail`, `ResetPassword`, `ConfirmPasswordReset`) and end
by sealing the same session cookie the Feature 001 callback seals. Refresh,
logout, bearer validation, and webhook sync are untouched. "Continue with
Google" reuses `GET /v1/auth/login`, now with a `provider=google` query
that builds the authorization URL with the Google provider selected so the
browser goes straight to Google and returns through the existing callback.
WorkOS sends the verification-code email itself. Its headless API issues a
password-reset token but sends no email, so the API gains a small mailer
(Resend over plain HTTP, log-only in dev) for the reset and "account
exists" emails. The constitution's Authentication section is amended to
1.2.0 and the Cloudflare rate-limiting rule on `/v1/auth/*` is documented.

## Technical Context

**Language/Version**: Go 1.25.7 (`backend/`), TypeScript ~6.0 with React
19 and Vite 8 on Node 22 (`web/`).

**Primary Dependencies**:
- Backend: existing echo v5, uber fx, go-playground/validator,
  workos-go v10.3.0 (no new WorkOS methods need a version bump), jwx v3.
  No new module. The Resend mailer is a ~60-line `net/http` client.
- Web: existing react-hook-form 7, `@hookform/resolvers` 5, zod 4,
  TanStack Query 5, openapi-fetch, shadcn `Input`, `Label`, `Button`,
  `Card`, `Alert`. No new package.

**Storage**: none. The `users` table and its webhook sync are unchanged.
Pending verification tokens and reset tokens live at WorkOS and, for the
duration of one screen, in the web app's component state.

**Testing**: Backend integration suite (testcontainers Postgres, real fx
graph, the fake WorkOS HTTP server extended with the headless endpoints,
a capturing fake mailer). Web Vitest + Testing Library + MSW at the HTTP
boundary. Manual browser walk against the SoW WorkOS staging environment
for the Google leg and the real emails.

**Target Platform**: unchanged (Railway API, Cloudflare Workers static
assets, local `localhost:8080` / `localhost:5173`).

**Project Type**: Web application monorepo (`contracts/`, `backend/`,
`web/`).

**Performance Goals**: Sign-in is one API round trip plus one WorkOS call
(SC-002 within 3 s). No new hot-path work. No database access on any new
endpoint.

**Constraints**: Browser never navigates to `*.authkit.app` or
`api.workos.com` (SC-001). Passwords, codes, and reset tokens never appear
in logs, error messages, or metrics labels. Refusals are fixed codes with
no provider text except the `weak_password` message. Cyclomatic
complexity ≤ 10. Comment voice per constitution IV. Existing accounts keep
working with no migration.

**Scale/Scope**: 6 new API operations, 1 changed (`/auth/login` query), 0
tables, 4 web pages changed or added (`/`, `/sign-up`,
`/forgot-password`, `/reset-password`), 1 shared verification-code
component, 1 mailer package, constitution 1.2.0, 3 docs updated.

## Constitution Check

*GATE: evaluated pre-Phase-0 and re-checked post-design — PASS with one
governance change that this feature carries.*

| Principle | Status | Notes |
|---|---|---|
| I. Spec-Driven Development | PASS | spec.md governs. The contract change (`contracts/auth-endpoints.yaml` here, merged into `contracts/openapi.yaml` as the first implementation task) precedes code. The spec was amended during planning (Assumptions: platform-sent reset and "account exists" emails) before any task was derived from it. |
| II. Test-First & Meaningful Coverage | PASS | Failing integration tests precede each endpoint and each page. Backend tests assert cookies, status codes, error codes, provider calls received by the fake, and mail captured by the fake mailer. Web tests assert rendered state and MSW-observed request bodies. |
| III. Security is Paramount | PASS | New endpoints sit in the `/v1/auth` group: stricter limiter, `Origin` check, validated DTOs with length caps. Credentials travel only in POST bodies over TLS. The reset token appears once in the reset page URL and is removed from the address bar on read. Error codes never distinguish unknown email from wrong password. The mailer logs message bodies only in `dev`. Cloudflare rate-limit rule documented. |
| IV. Simplicity & Patterns | PASS | Endpoints extend the existing `auth-*` vertical. The mailer is one interface with two implementations behind fx. No user store changes. No new state machine beyond the two-step form. |
| V. Contract Integrity & RESTful API | PASS | Contract first. Generated types on both sides. `202` for accepted-without-session, `200` with `AccessToken` for signed-in, fixed error codes. The `screen` query parameter on `/auth/login` is replaced by `provider`. That is a breaking change on an operation only the SPA calls, flagged here with its migration: the SPA changes in the same PR. |
| VI. Observability & Debuggability | PASS | Every new endpoint goes through the existing request logger and central error handler. Failures are wrapped with operation context and classified through `providerError`. `user_id` is attached on success paths as the callback does today. |

**Governance change carried by this feature**: constitution Authentication
section 1.1.0 → 1.2.0 (MINOR): the hosted login page is no longer
mandated. Items 1 and 2 are rewritten (first-party forms, headless
exchanges, Google via direct provider selection), item 7 gains "the
identity provider's hosted pages are not used", and a new item states
where brute-force protection lives. The Development Workflow's
subagent rule gains one sentence: the orchestrating session delegates
implementation through the repository's implement workflow and does
not write feature code itself (copied from the marketplace practice,
whose workflow script now lives at
`.claude/workflows/speckit-parallel-implement.js`). Binding docs updated alongside:
`docs/backend-conventions.md` §10 Auth, `docs/frontend-conventions.md`
§9, `docs/deployment.md` (WorkOS settings, Resend variables, Cloudflare
rule).

**Post-Phase-1 re-check**: PASS. Design added the decoy pending token for
duplicate sign-ups (strengthens III, no enumeration) and the dev-only log
mailer (documented exception, `dev` stage only).

## Project Structure

### Documentation (this feature)

```text
specs/004-first-party-auth-forms/
├── plan.md                    # This file
├── research.md                # Phase 0 — decisions R1–R14
├── data-model.md              # Phase 1 — request/response shapes, client state, mail messages
├── quickstart.md              # Phase 1 — run & validate end to end
├── contracts/
│   ├── auth-endpoints.yaml    # Target additions to contracts/openapi.yaml
│   ├── emails.md              # The two platform-sent emails
│   └── ui-routes.md           # Pages, steps, and messages
└── tasks.md                   # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
contracts/
└── openapi.yaml                          # + 6 operations, /auth/login provider param, 7 schemas

backend/
├── .env.example                          # + RESEND_API_KEY, EMAIL_FROM
├── internal/
│   ├── config/config.go                  # + ResendAPIKey, EmailFrom (required when Deployed())
│   ├── common/errors.go                  # + ErrInvalidCredentials, ErrInvalidCode, ErrInvalidResetToken, ErrWeakPassword
│   ├── mail/                             # NEW: mailer.go (Mailer, Message), resend.go, log.go, templates.go
│   ├── boot/mail.go                      # NEW: NewMailer picks resend or log by config
│   ├── boot/app.go                       # + mail provider
│   ├── models/dto/auth-dto.go            # + SignIn, SignUp, VerifyEmail, EmailOnly, ResetPassword DTOs; LoginQueryDto.Provider
│   ├── models/dto/api.gen.go             # regenerated
│   ├── services/workos-client.go         # + AuthenticateWithPassword, CreateUser, AuthenticateWithEmailVerification, SendVerificationEmail, FindUserByEmail, CreatePasswordReset, ConfirmPasswordReset; AuthorizationURL takes a provider
│   ├── services/auth-service.go          # + SignIn, SignUp, VerifyEmail, ResendVerification, ForgotPassword, ResetPassword
│   ├── controllers/auth-controller.go    # + six handlers; Login reads provider
│   └── routes/auth-routes.go             # + six POST routes (Origin-checked, validated)
└── tests/integration/
    ├── fake_workos_test.go               # + password grant, email-verification grant, users create/list, email_verification/send, password_reset, password_reset/confirm
    ├── fake_mailer_test.go               # NEW: capturing Mailer via fx.Replace
    ├── auth_test.go                      # login provider tests updated
    └── auth_headless_test.go             # NEW: the six endpoints

web/
├── src/
│   ├── app/router.tsx                    # + /forgot-password, /reset-password
│   ├── lib/auth/session.ts               # + adoptSession(token); signInUrl → googleSignInUrl
│   ├── lib/api/schema.d.ts               # regenerated
│   ├── features/auth/
│   │   ├── api.ts                        # + useSignIn, useSignUp, useVerifyEmail, useResendVerification, useForgotPassword, useResetPassword
│   │   ├── schemas.ts                    # NEW: zod schemas shared by the forms
│   │   ├── messages.ts                   # NEW: error code → notice mapping
│   │   ├── components/auth-shell.tsx     # NEW: card layout shared by the pages
│   │   ├── components/google-button.tsx  # NEW
│   │   ├── components/verify-code-form.tsx  # NEW: code step with resend cooldown
│   │   ├── pages/sign-in-page.tsx        # form + code step + Google + links
│   │   ├── pages/sign-up-page.tsx        # form + code step + Google
│   │   ├── pages/forgot-password-page.tsx   # NEW
│   │   ├── pages/reset-password-page.tsx    # NEW
│   │   ├── sign-in.integration.test.tsx  # rewritten
│   │   ├── sign-up.integration.test.tsx  # rewritten
│   │   ├── password-reset.integration.test.tsx  # NEW
│   │   └── index.ts
│   └── test/msw.ts                       # + handlers for the six endpoints

.specify/memory/constitution.md           # 1.2.0
docs/backend-conventions.md               # §10 Auth
docs/frontend-conventions.md              # §9 Auth
docs/deployment.md                        # WorkOS settings, Resend, Cloudflare rule
```

**Structure Decision**: Extend the Feature 001 `auth-*` vertical rather
than add a second one. The only new package is `internal/mail`, kept
separate from `services` because it is a transport (like the WorkOS
client) and because the send-agreement feature will reuse it. On the web,
the four auth pages share one shell component and one code-step
component. Feature-local zod schemas and message tables follow the
frontend conventions §5 colocation rule.

## Complexity Tracking

No constitution violations. Deferrals and their grounds:

| Deferred item | Mandate source | Why deferral is compliant |
|---|---|---|
| Playwright E2E for the Google leg | conventions §11 | Needs a real Google account in a deployed stage. Covered by the manual walk in quickstart.md. |
| Idempotency keys on the mail-sending endpoints | constitution V | Both endpoints (`forgot-password`, duplicate `sign-up`) are idempotent by construction: repeating them sends another copy of the same email, which is the documented resend behavior, and the per-IP limiter bounds the volume. Recorded here so the send-agreement feature revisits it. |
| Redis-backed limiter | conventions §3 | Single replica, unchanged from Feature 001. Cloudflare's rule is the shared limiter. |
