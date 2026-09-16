# Tasks: First-Party Sign-In and Sign-Up Forms

**Input**: Design documents from `/specs/004-first-party-auth-forms/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included. The constitution mandates TDD: every test task is written and failing before the implementation task that follows it. Backend tests boot the real fx graph against the fake WorkOS server and a capturing fake mailer. Web tests mount the real routes with MSW at the HTTP boundary.

**Organization**: Phase 1 lands the contract, the constitution amendment, and the config. Phase 2 builds the shared plumbing (mailer, provider client methods, error codes, session adoption, fake provider extensions) that every story needs. Phases 3 to 7 map to the spec's user stories in priority order, backend first then web within each. Phase 8 is docs, the gate, and the browser walk. GitHub issues are created from this file with `/speckit-taskstoissues`.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1 to US5)
- Paths are repository-relative.

## Path Conventions

- Backend: `backend/` (Go, module `github.com/pixels-two/sow/backend`)
- Web: `web/` (Vite + React SPA). Feature code under `web/src/features/auth/`
- Contract: `contracts/openapi.yaml`. Target additions in `specs/004-first-party-auth-forms/contracts/auth-endpoints.yaml`
- Governance: `.specify/memory/constitution.md`, `docs/`

---

## Phase 1: Setup (contract, governance, config)

**Purpose**: Put the contract and the rules in place before any code, and add the configuration every later task reads.

- [X] T001 Create branch `feature/004-first-party-auth-forms` from `main`; commit the `specs/004-first-party-auth-forms/` folder as the first commit
- [X] T002 Merge `specs/004-first-party-auth-forms/contracts/auth-endpoints.yaml` into `contracts/openapi.yaml`: replace the `/auth/login` operation (query `provider` enum `[google]` required, `returnTo` optional, `400 ValidationFailed`), add the six `POST /auth/*` operations, the `SessionStarted`, `VerificationPending`, `ValidationFailed`, `InvalidCredentials`, `WeakPassword` responses, and the `Credentials`, `SignUpRequest`, `PendingVerification`, `VerifyEmailRequest`, `EmailRequest`, `ResetPasswordRequest` schemas; bump `info.version` to `0.2.0`; update the `info.description` sentence about session endpoints to also cite `specs/004-first-party-auth-forms/contracts/`
- [X] T003 [P] Regenerate both clients: `cd backend && make oapi-generate` (updates `backend/internal/models/dto/api.gen.go`) and `cd web && npm run generate:api` (updates `web/src/lib/api/schema.d.ts`); confirm `git diff --stat` shows only those two generated files plus the contract
- [X] T004 [P] Amend `.specify/memory/constitution.md` to 1.2.0: rewrite Authentication items 1 and 2 (the web app presents its own sign-in, sign-up, verification, and reset screens; the API exchanges credentials, codes, and reset tokens with WorkOS server-side through six JSON endpoints under `/v1/auth` and seals the same cookie; social sign-in starts at `GET /v1/auth/login?provider=` which sends the browser straight to the provider's consent screen and returns through the API callback), extend item 7 with "WorkOS's hosted pages are not used", add item 9 (brute-force and bot protection on `/v1/auth/*` is a Cloudflare rate-limiting rule in front of the API plus the API's own per-address limiter; refusals never reveal whether an address is registered), add item 10 (the API sends the password-reset and account-exists emails through the project's transactional mail provider; WorkOS sends verification codes), add to the Development Workflow "Subagent model and effort match the task" bullet the sentence "The orchestrating session delegates implementation: it runs the repository's implement workflow (`.claude/workflows/speckit-parallel-implement.js`) or spawns implementers, marks progress, and owns the final gate. It does not write feature code itself.", update the Sync Impact Report (1.1.0 → 1.2.0, MINOR, rationale: hosted page removed to avoid the custom-domain dependency, companion docs), and set `**Version**: 1.2.0 | **Ratified**: 2026-09-03 | **Last Amended**: 2026-09-04`
- [X] T005 [P] Add `ResendAPIKey string` (`env:"RESEND_API_KEY"`) and `EmailFrom string` (`env:"EMAIL_FROM"`, `validate:"omitempty,email"`) to `AppConfig` in `backend/internal/config/config.go`, with a `Validate`-time rule that both are required when `Deployed()`; document both in `backend/.env.example` under a new "Transactional email" block (empty locally means emails are written to the log)
- [X] T006 [P] Add the four sentinels to `backend/internal/common/errors.go`: `ErrInvalidCredentials` (401, code `invalid_credentials`), `ErrInvalidCode` (400, `invalid_code`), `ErrInvalidResetToken` (400, `invalid_reset_token`), and a `WeakPasswordError{Message string}` type (422, code `weak_password`, `details` = `{"message": ...}`) wired into `HTTPStatus`, `ErrorCode`, and the central error handler's details path

---

## Phase 2: Foundational (blocking prerequisites)

**Purpose**: Shared plumbing every story depends on. Each test file is written first and fails before its implementation.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T007 [P] Create failing `backend/internal/mail/mail_test.go`: `LogMailer.Send` writes one info line carrying the subject and the recipient's domain only; `ResendMailer.Send` posts JSON `{from, to:[to], subject, text}` with `Authorization: Bearer <key>` to the configured base URL (an `httptest` server), returns nil on 200, and returns a wrapped error naming the status on 4xx/5xx without the body; `PasswordResetMessage(webAppURL, to, resetURL, expiresAt)` and `AccountExistsMessage(webAppURL, to)` produce the exact subjects and bodies in `specs/004-first-party-auth-forms/contracts/emails.md`
- [X] T008 Implement `backend/internal/mail/mailer.go` (`Mailer` interface, `Message{To, Subject, Text}`), `backend/internal/mail/log.go` (`LogMailer`), `backend/internal/mail/resend.go` (`ResendMailer{client *http.Client, baseURL, apiKey, from}` with `NewResendMailer`), and `backend/internal/mail/templates.go` (the two message builders)
- [X] T009 Add `NewMailer(cfg config.AppConfig, logger *slog.Logger) mail.Mailer` to `backend/internal/boot/mail.go` (Resend when `RESEND_API_KEY` is set, log otherwise) and register it in the "Identity provider and session primitives" `fx.Provide` block of `backend/internal/boot/app.go`
- [X] T010 [P] Extend the `WorkOSClient` interface and `sdkWorkOSClient` in `backend/internal/services/workos-client.go`: `AuthorizationURL(state, provider string) string` (maps `google` to `UserManagementAuthenticationProviderGoogleOAuth`, no screen hint), `AuthenticateWithPassword(ctx, email, password, ip, userAgent) (*workos.AuthenticateResponse, error)`, `CreateUser(ctx, email, firstName, lastName, password) (*workos.UserCreateResponse, error)`, `AuthenticateWithEmailVerification(ctx, pendingToken, code, ip, userAgent) (*workos.AuthenticateResponse, error)`, `SendVerificationEmail(ctx, userID) error`, `FindUserByEmail(ctx, email) (*workos.User, error)` (via `List` with the email filter, `ErrUserNotFound` when empty), `CreatePasswordReset(ctx, email) (*workos.PasswordReset, error)`, `ConfirmPasswordReset(ctx, token, newPassword) (*workos.ResetPasswordResponse, error)`
- [X] T011 [P] Extend `backend/tests/integration/fake_workos_test.go`: users keyed by email with `Password` and `EmailVerified` fields; `POST /user_management/users` (409-style 4xx `{"code":"user_creation_error","errors":[{"code":"email_not_available"}]}` on a duplicate, 422 `{"code":"password_strength_error","message":"Password is too weak"}` when the password contains `weak`, else creates unverified and returns the user); `grant_type=password` (unknown email or wrong password → 400 `{"error":"invalid_credentials"}`, unverified → 422 `{"code":"email_verification_required","pending_authentication_token":"pat_N","email_verification_id":"ev_N"}` and records a code `"123456"` for the user, verified → tokens); `grant_type=urn:workos:oauth:grant-type:email-verification:code` (matching pending token and code → marks verified, returns tokens; else 400 `{"code":"invalid_one_time_code"}`); `POST /user_management/users/{id}/email_verification/send` (rotates the code, counts sends per user); `GET /user_management/users?email=` (list with `data`); `POST /user_management/password_reset` (known email → `{"password_reset_token":"prt_N","password_reset_url":"…","expires_at":…}`, unknown → 404 `{"code":"entity_not_found"}`); `POST /user_management/password_reset/confirm` (valid token → sets the password, marks verified, consumes the token, returns `{"user":…}`; else 400 `{"code":"invalid_password_reset_token"}`); accessors `SentCodes(userID) int`, `CurrentCode(email) string`, `LastResetToken(email) string`, `PasswordOf(email) string`
- [X] T012 [P] Create `backend/tests/integration/fake_mailer_test.go`: `fakeMailer` capturing `[]mail.Message` under a mutex with `Sent()` and `Reset()`, installed through `fx.Replace(fx.Annotate(fm, fx.As(new(mail.Mailer))))` by a `startAuthAppWithMail(t, fake, overrides) (*testApp, *fakeMailer)` helper added next to `startAuthApp` in `backend/tests/integration/auth_test.go`
- [X] T013 [P] Add failing tests to `web/src/lib/auth/session.test.ts` (create the file if absent): `adoptSession({access_token, expires_at})` sets the store to authenticated with that token, schedules a refresh `expires_at - 60 s` out (fake timers), and clears the `sow.signed-out` sentinel; `googleSignInUrl('/agreements/7')` equals `${VITE_API_URL}/v1/auth/login?provider=google&returnTo=%2Fagreements%2F7` and an unsafe path falls back to `/dashboard`
- [X] T014 Implement `adoptSession` and `googleSignInUrl` in `web/src/lib/auth/session.ts` and remove `signInUrl` and the `Screen` type
- [X] T015 [P] Add MSW handlers to `web/src/test/msw.ts` for the six endpoints with contract-conformant defaults and a `recordedRequests()` accessor: `signInHandler(outcome)` where outcome is `'session' | 'pending' | 'invalid' | 'unavailable'`, `signUpHandler('pending' | 'weak')`, `verifyEmailHandler('session' | 'invalid')`, `resendHandler()`, `forgotPasswordHandler()`, `resetPasswordHandler('session' | 'invalid' | 'weak')`; defaults installed in `server`: sign-in `invalid`, sign-up `pending`, verify `invalid`, reset `invalid`
- [X] T016 [P] Create `web/src/features/auth/schemas.ts` (zod: `emailSchema` trimmed `z.email().max(254)`, `passwordSchema` `z.string().min(8).max(256)`, `nameSchema` trimmed `min(1).max(100)`, `codeSchema` `/^\d{6}$/`, and the composed `signInSchema`, `signUpSchema`, `verifyCodeSchema`, `emailOnlySchema`, `newPasswordSchema`) with `web/src/features/auth/schemas.test.ts` covering trim, caps, and the code pattern
- [X] T017 [P] Create `web/src/features/auth/messages.ts` exporting `authNotice(code: string): Notice` for `invalid_credentials`, `invalid_code`, `invalid_reset_token`, `provider_unavailable`, `network`, and the four Feature 001 codes, with the copy from `specs/004-first-party-auth-forms/contracts/ui-routes.md`; move the `notices` table out of `web/src/features/auth/pages/sign-in-page.tsx` into it
- [X] T018 [P] Create `web/src/features/auth/components/auth-shell.tsx` (the `<main>` + "SoW" heading + `Card` frame the four pages share, props `title`, `description`, `children`, `footer`) and `web/src/features/auth/components/google-button.tsx` (`<a>` styled as an outline button, text "Continue with Google", `href={googleSignInUrl(returnTo)}`)
- [X] T019 Add `useAuthMutation`-style hooks to `web/src/features/auth/api.ts`: `useSignIn`, `useSignUp`, `useVerifyEmail`, `useResendVerification`, `useForgotPassword`, `useResetPassword`, each a TanStack `useMutation` calling `api.POST('/auth/…', { body, credentials: 'include' })`, throwing `ApiError(status, code, details)` on error, and returning the typed data (`AccessToken`, `PendingVerification`, or void); export them from `web/src/features/auth/index.ts`

**Checkpoint**: `cd backend && go test ./internal/mail/ ./tests/integration/ -run 'TestLogin|TestFake'` and `cd web && npm run test -- session schemas` green.

---

## Phase 3: User Story 1 - Returning user signs in with email and password (Priority: P1) 🎯 MVP

**Goal**: The sign-in page is a first-party form. Correct credentials start a session in one round trip. Unverified emails get the code step.

**Independent Test**: With a verified account, submit email and password on `/`, land on the dashboard, reload and stay signed in, address bar never leaves the app origin.

### Tests for User Story 1

- [X] T020 [P] [US1] Create failing `backend/tests/integration/auth_headless_test.go` with `TestSignInStartsSession` (verified user → 200 `{access_token, expires_at}`, `Set-Cookie sow_session` with the Feature 001 attributes, the fake saw `grant_type=password` with the email and password, `POST /v1/auth/refresh` with the cookie then succeeds), `TestSignInRefusesIdentically` (wrong password and unknown email both → 401 body `{"error":"invalid_credentials","request_id":…}` byte-identical except `request_id`, no cookie), `TestSignInUnverifiedPending` (unverified user → 202 `{"pending_token":"pat_1"}`, no cookie), `TestSignInValidation` (malformed email, 7-char password, 257-char password → 400 `validation_failed` with the field named), `TestSignInOriginAndLimiter` (missing or foreign `Origin` → 403 `forbidden_origin`; with `AUTH_RATE_LIMIT_BURST=2` the third call → 429), `TestSignInProviderDown` (fake answering 500 → 503 `provider_unavailable`, no cookie)
- [X] T021 [P] [US1] Add failing `TestVerifyEmailStartsSession` (pending token and the fake's current code → 200 with cookie, user marked verified, `users` row provisioned) and `TestVerifyEmailRejectsCode` (wrong code, reused token → 400 `invalid_code`, no cookie; malformed code → 400 `validation_failed`) to `backend/tests/integration/auth_headless_test.go`
- [X] T022 [P] [US1] Rewrite `web/src/features/auth/sign-in.integration.test.tsx`: renders email and password fields, "Sign in" button, "Forgot password?" link to `/forgot-password`, the Google link with the expected href carrying `?redirect=`, and "Create an account" to `/sign-up`; submitting with the `session` handler adopts the token (store authenticated) and navigates to the redirect target; submitting with the default `invalid` handler shows the alert "That email and password don't match." and keeps the email value; a 503 shows the unavailable notice and keeps both values; client validation blocks a malformed email and a 7-char password with field messages and sends no request; the submit button is disabled while pending; `?error=` notices from the Google leg still render; an authenticated visitor is redirected
- [X] T023 [P] [US1] Create failing `web/src/features/auth/components/verify-code-form.test.tsx`: renders "Check your inbox" with the email, a numeric code input, "Verify", "Send a new code", and "Use a different email"; a 5-digit code is blocked client-side; submitting `123456` posts `{pending_token, code}` and on `session` calls `onSignedIn` with the token; on `invalid` shows "That code didn't work. It may have expired." and clears the input; "Send a new code" posts `{email}`, shows "We sent a new code.", and stays disabled with a countdown for 30 s (fake timers) before re-enabling; "Use a different email" calls `onRestart`
- [X] T024 [P] [US1] Add to `web/src/features/auth/sign-in.integration.test.tsx` a case where the `pending` handler moves the page to the code step showing the typed email, and a correct code then lands on the redirect target

### Implementation for User Story 1

- [X] T025 [US1] Add `SignInDto{Email, Password}` and `VerifyEmailDto{PendingToken, Code}` with the validation tags from research R12 to `backend/internal/models/dto/auth-dto.go`; change `LoginQueryDto.Screen` to `Provider string \`query:"provider" json:"provider" validate:"required,oneof=google"\``
- [X] T026 [US1] Add to `backend/internal/services/auth-service.go`: `Session` and `PendingVerification` result types, `SignIn(ctx, email, password, ip, ua) (Session, *PendingVerification, error)`, `VerifyEmail(ctx, pendingToken, code, ip, ua) (Session, error)`, the `passwordAuthError` classifier (research R3: `email_verification_required` with a pending token → pending, other 4xx → `ErrInvalidCredentials`, else `ErrProviderUnavailable`), a `startSession(ctx, resp)` helper that upserts the user and seals the cookie (shared with `CompleteLogin`), and `verificationError` (4xx → `ErrInvalidCode`); trim the email; change `LoginURL(returnTo, provider)` to pass the provider through
- [X] T027 [US1] Add `SignIn` and `VerifyEmail` handlers to `backend/internal/controllers/auth-controller.go` (read the validated DTO, call the service, on a session set the cookie with `session.NewCookie` and answer 200 `dto.AccessToken`, on pending answer 202 `dto.PendingVerification`, otherwise return the wrapped error for the central handler); a private `respondSession(c, Session)` helper shared by every session-starting handler; update `Login` to pass `query.Provider`
- [X] T028 [US1] Register `POST /sign-in` and `POST /verify-email` in `backend/internal/routes/auth-routes.go` with `appmw.RequireOrigin(cfg.CORSAllowedOrigins)` and `appmw.ValidateRequest(new(dto.SignInDto))` / `new(dto.VerifyEmailDto)`; update the `/login` validation to the new DTO
- [X] T029 [US1] Update `backend/tests/integration/auth_test.go`: `TestLoginRedirectsToHostedAuthKit` becomes `TestLoginRedirectsToGoogle` asserting `provider=GoogleOAuth`, no `screen_hint`, and the signed state; `TestLoginDefaultsAndValidation` asserts `provider` missing or `authkit` → 400 `validation_failed`; keep the callback, refresh, and logout tests unchanged
- [X] T030 [US1] Create `web/src/features/auth/components/verify-code-form.tsx` (props `email`, `pendingToken`, `onSignedIn(token)`, `onRestart()`; `useVerifyEmail` and `useResendVerification`; 30 s cooldown with `setTimeout`; `role="status"` for the resend confirmation, `role="alert"` for the invalid-code notice; input `inputMode="numeric" autoComplete="one-time-code" maxLength={6}`)
- [X] T031 [US1] Rewrite `web/src/features/auth/pages/sign-in-page.tsx`: `AuthShell`, `useForm` with `zodResolver(signInSchema)`, `Input` + `Label` + error paragraph per field, `useSignIn`, step state `'form' | { kind: 'verify', email, pendingToken }`, on 200 `adoptSession` then `navigate(returnTo, { replace: true })`, on 202 switch to `VerifyCodeForm`, `ApiError` → `authNotice(code)` alert, network or 5xx → the unavailable notice, `GoogleButton`, links per `contracts/ui-routes.md`, authenticated redirect preserved

**Checkpoint**: `cd backend && go test ./tests/integration/ -run 'TestSignIn|TestVerifyEmail|TestLogin'` and `cd web && npm run test -- sign-in verify-code` green. Story 1 is demonstrable on its own.

---

## Phase 4: User Story 2 - New visitor signs up with email and password (Priority: P2)

**Goal**: The sign-up page creates the account, moves to the code step, and the code lands the new user on the dashboard. Duplicate addresses are indistinguishable and get the "account exists" email.

**Independent Test**: Fresh email through `/sign-up`, code from the inbox, dashboard greets by first name, `users` row exists.

### Tests for User Story 2

- [X] T032 [P] [US2] Add failing `TestSignUpCreatesAndPends` (names, email, password → 202 `{"pending_token":"pat_N"}`, the fake holds an unverified user with those names and password, one code sent, no cookie, no `users` row yet), `TestSignUpDuplicateIsIndistinguishable` (existing email → 202 with a `pending_token` of the same length class, no second user, no code sent, the fake mailer holds one message to that address with the "account exists" subject and body from `contracts/emails.md`; a later `verify-email` with that token → 400 `invalid_code`), `TestSignUpWeakPassword` (password containing `weak` → 422 `{"error":"weak_password","details":{"message":"Password is too weak"}}`, no user created), `TestSignUpValidation` (empty first name, 101-char last name, malformed email → 400 with the field named), `TestSignUpThenVerifyProvisions` (sign-up then verify with the current code → 200, cookie, `users` row with the joined name) to `backend/tests/integration/auth_headless_test.go`
- [X] T033 [P] [US2] Add failing `TestResendVerificationRotatesCode` (known unverified email → 202, the fake's code changed and send count incremented) and `TestResendVerificationNeutral` (unknown email and verified email → 202, nothing sent) to `backend/tests/integration/auth_headless_test.go`
- [X] T034 [P] [US2] Rewrite `web/src/features/auth/sign-up.integration.test.tsx`: renders first name, last name, email, password (with the "At least 8 characters" helper), "Create account", the Google link, and "Sign in" to `/`; client validation blocks empty names, a malformed email, and a short password with no request sent; a valid submit posts `{first_name, last_name, email, password}` (names trimmed) and shows the code step with the email; a correct code adopts the session and lands on `/dashboard`; the `weak` handler shows the server message under the password field and keeps every value; an authenticated visitor is redirected

### Implementation for User Story 2

- [X] T035 [US2] Add `SignUpDto{FirstName, LastName, Email, Password}` and `EmailDto{Email}` to `backend/internal/models/dto/auth-dto.go` with the research R12 tags
- [X] T036 [US2] Add to `backend/internal/services/auth-service.go`: `SignUp(ctx, firstName, lastName, email, password, ip, ua) (PendingVerification, error)` (trim names and email; `CreateUser`; `createUserError` per research R3: a 4xx naming `password` → `WeakPasswordError` with the provider message, any other 4xx → the duplicate branch that sends `mail.AccountExistsMessage` through the mailer, logs a warn on send failure, and returns a decoy token from `crypto/rand` (32 bytes, base64url); on success call `AuthenticateWithPassword` to obtain the pending token and the provider's code email), and `ResendVerification(ctx, email) error` (`FindUserByEmail`; not found or already verified → return nil; else `SendVerificationEmail`); inject `mail.Mailer` and `cfg.WebAppURL` into `AuthService`
- [X] T037 [US2] Add `SignUp` (202 `dto.PendingVerification`) and `ResendVerification` (202 no body) handlers to `backend/internal/controllers/auth-controller.go` and register `POST /sign-up` and `POST /resend-verification` in `backend/internal/routes/auth-routes.go` with `RequireOrigin` and the DTO validators
- [X] T038 [US2] Rewrite `web/src/features/auth/pages/sign-up-page.tsx`: `AuthShell`, `useForm` with `zodResolver(signUpSchema)`, four fields, `useSignUp`, on 202 switch to `VerifyCodeForm` with `returnTo` fixed at `/dashboard`, on `weak_password` call `setError('password', { message: details.message })`, `GoogleButton`, footer link, authenticated redirect

**Checkpoint**: `cd backend && go test ./tests/integration/ -run 'TestSignUp|TestResend'` and `cd web && npm run test -- sign-up` green.

---

## Phase 5: User Story 3 - Continue with Google (Priority: P3)

**Goal**: The Google button on both pages goes straight to Google and returns through the existing callback. Same-email accounts link.

**Independent Test**: Click "Continue with Google" on `/`, finish on Google, land on the dashboard; the only non-app host visited is Google's.

### Tests for User Story 3

- [X] T039 [P] [US3] Add failing `TestGoogleRoundTrip` to `backend/tests/integration/auth_test.go`: `GET /v1/auth/login?provider=google&returnTo=%2Fagreements%2F1` → 302 to the fake's authorize endpoint with `provider=GoogleOAuth`; the callback with a code the fake issued for an existing user → cookie and redirect to `/callback?returnTo=%2Fagreements%2F1`, `users` row unchanged in id; a code for a brand-new Google user → row created with Google's names
- [X] T040 [P] [US3] Add to `web/src/features/auth/sign-in.integration.test.tsx` and `sign-up.integration.test.tsx` an assertion that the Google link is an anchor (a top-level navigation, no fetch), carries `provider=google`, and on the sign-in page carries the `?redirect=` path; keep `web/src/features/auth/callback.integration.test.tsx` green unchanged

### Implementation for User Story 3

- [X] T041 [US3] Confirm `AuthorizationURL` in `backend/internal/services/workos-client.go` sets `Provider: GoogleOAuth`, no `ScreenHint`, `RedirectURI`, and `State`; add a unit test `backend/internal/services/workos-client_test.go` that parses the produced URL and asserts those four query values and the absence of `screen_hint`
- [X] T042 [US3] Update `web/src/features/auth/pages/callback-page.tsx` comment and the `flow_incomplete` path so a cancelled Google consent lands on `/?error=flow_incomplete` with the Feature 001 notice (behavior unchanged, comment updated to name the Google leg)

**Checkpoint**: `cd backend && go test ./tests/integration/ -run 'TestGoogle|TestLogin|TestCallback' ./internal/services/` and `cd web && npm run test -- sign-in sign-up callback` green.

---

## Phase 6: User Story 4 - Forgot and reset password (Priority: P4)

**Goal**: A forgotten password is recovered through the app's own pages and a platform-sent email.

**Independent Test**: Request a reset for a known email, open the emailed link, set a new password, old fails and new works.

### Tests for User Story 4

- [X] T043 [P] [US4] Add failing `TestForgotPasswordEmailsLink` (known email → 202, the fake mailer holds one message with the reset subject whose body contains `http://localhost:5173/reset-password?token=prt_N` and the RFC 1123 expiry; the API log contains no `prt_`), `TestForgotPasswordNeutral` (unknown email → 202, nothing sent), `TestForgotPasswordMailFailureStill202` (mailer returning an error → 202 and one warn log line), `TestResetPasswordSignsIn` (valid token and a new password → 200 with cookie, the fake's stored password changed, the token consumed, the user verified), `TestResetPasswordRejectsToken` (reused or unknown token → 400 `invalid_reset_token`, no cookie), `TestResetPasswordWeak` (password containing `weak` → 422 `weak_password`) to `backend/tests/integration/auth_headless_test.go`
- [X] T044 [P] [US4] Create failing `web/src/features/auth/password-reset.integration.test.tsx`: `/forgot-password` renders the email field and "Email me a reset link"; submit posts `{email}` and replaces the form with the neutral confirmation and "Back to sign in"; `/reset-password?token=abc` renders "Choose a new password" and the address bar (router location) no longer carries `token` after mount; submitting a new password posts `{token: 'abc', password}` and on `session` adopts the token and lands on `/dashboard`; the default `invalid` handler shows "This link is no longer valid" with "Request a new link" to `/forgot-password`; `/reset-password` with no token shows the same invalid state; `weak` shows the server message under the field; a short password is blocked client-side

### Implementation for User Story 4

- [X] T045 [US4] Add `ResetPasswordDto{Token, Password}` to `backend/internal/models/dto/auth-dto.go`
- [X] T046 [US4] Add to `backend/internal/services/auth-service.go`: `ForgotPassword(ctx, email) error` (`CreatePasswordReset`; 4xx → return nil; success → build `WebAppURL + "/reset-password?token=" + url.QueryEscape(token)`, send `mail.PasswordResetMessage`, warn on send failure with the user id only), `ResetPassword(ctx, token, password, ip, ua) (Session, error)` (`ConfirmPasswordReset`; 4xx naming `password` → `WeakPasswordError`, other 4xx → `ErrInvalidResetToken`; then `AuthenticateWithPassword(user.Email, password)` and `startSession`)
- [X] T047 [US4] Add `ForgotPassword` (202) and `ResetPassword` (200 via `respondSession`) handlers to `backend/internal/controllers/auth-controller.go` and register `POST /forgot-password` and `POST /reset-password` in `backend/internal/routes/auth-routes.go`
- [X] T048 [P] [US4] Create `web/src/features/auth/pages/forgot-password-page.tsx` (`AuthShell`, `useForm` with `emailOnlySchema`, `useForgotPassword`, `sent` state swapping in the confirmation text and the "Back to sign in" link, authenticated redirect)
- [X] T049 [P] [US4] Create `web/src/features/auth/pages/reset-password-page.tsx` (reads `token` from `useSearchParams` into `useState` on mount then `setSearchParams({}, { replace: true })`; `invalid` state when the token is absent or the API answers `invalid_reset_token`; `useForm` with `newPasswordSchema`; `useResetPassword`; on 200 `adoptSession` and `navigate('/dashboard', { replace: true })`; `weak_password` → `setError`)
- [X] T050 [US4] Register `/forgot-password` and `/reset-password` in `web/src/app/router.tsx` and export both pages from `web/src/features/auth/index.ts`

**Checkpoint**: `cd backend && go test ./tests/integration/ -run 'TestForgot|TestReset'` and `cd web && npm run test -- password-reset` green.

---

## Phase 7: User Story 5 - Operate the change safely (Priority: P5)

**Goal**: Docs, deployment notes, and the edge rule match the shipped behavior. Old accounts keep working.

**Independent Test**: A reviewer reads the constitution and the contract against the running app; a Feature 001 account signs in with the new form.

- [X] T051 [P] [US5] Update `docs/backend-conventions.md` §10 "Auth: WorkOS through a token-mediating backend": replace the four-endpoint list with the ten operations (login with `provider`, callback, refresh, logout, sign-in, sign-up, verify-email, resend-verification, forgot-password, reset-password), state that the hosted page is not used, add the error-code rule (fixed codes, no provider text except `weak_password`, no enumeration), and add a bullet on `internal/mail` (interface, Resend, log-only in dev)
- [X] T052 [P] [US5] Update `docs/frontend-conventions.md` §9 "Auth (WorkOS through the API)": sign-in and sign-up are first-party forms posting JSON to `/v1/auth/*` with `credentials: 'include'`; `adoptSession` is the only way a mutation starts a session; the pending token and reset token live in component state only; Google is the one top-level navigation to `/v1/auth/login?provider=google`; list the auth calls that send cookies (callback, refresh, logout, and the six credential endpoints)
- [X] T053 [P] [US5] Update `docs/deployment.md`: WorkOS section gains the dashboard checklist from `specs/004-first-party-auth-forms/quickstart.md` §1 (password on, verification on, Google on, redirect URI, password-reset URL, homepage); the env table gains `RESEND_API_KEY` (secret, required in deployed stages) and `EMAIL_FROM`; the "Transactional email" row names Resend; a new "Cloudflare rate limiting" section records the two rules from research R13 with match expressions, thresholds, and actions; "Standing up a new stage" gains the Resend domain verification and the rate-limit rules
- [X] T054 [P] [US5] Add `TestLegacyAccountSignsIn` to `backend/tests/integration/auth_headless_test.go`: a user provisioned through the callback path (Feature 001 flow) then signs in with `POST /sign-in` and receives a session for the same user id
- [X] T055 [US5] Update `backend/README.md` and `web/README.md` architecture sections to describe the first-party forms and the mailer; update `web/.env.example` comment (the app still holds no provider configuration; sign-in posts to the API)

---

## Phase 8: Gate and browser walk

**Purpose**: The single full run of every gate, then the constitution's mandatory look at the pages.

- [X] T056 Run the full gate set once: `cd backend && make lint && go build ./... && make test`; `cd web && npm run generate:api && git diff --exit-code src/lib/api/schema.d.ts && npm run typecheck && npm run lint && npm run format:check && npm run test:coverage && npm run build`; `cd backend && make oapi-generate && git diff --exit-code internal/models/dto/api.gen.go`; fix anything red
- [X] T057 Check gocyclo (≤ 10) and comment voice across the files touched in `backend/internal/` and `web/src/features/auth/`; fix any function over the limit
- [ ] T058 Walk `specs/004-first-party-auth-forms/quickstart.md` §3 steps 1 to 5, 7, and 8 in the preview browser against the local API and the SoW WorkOS staging environment (after asking the user to complete the dashboard checklist in §1); record screenshots and the API log excerpts (with tokens redacted) on this task's issue; step 6 (Google) needs the user to complete Google's screens in the preview browser, so ask for that hand-off and record the outcome
- [X] T059 Mark completed tasks `[X]`, commit, push `feature/004-first-party-auth-forms`, open a PR to `main` titled "Feature 004: first-party sign-in and sign-up forms" whose body names every issue it closes; post the marketplace cross-reference comment on `Pixels-Two/marketplace#268` and `#1` if not already posted

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: T001 first. T002 before T003. T004, T005, T006 parallel after T001.
- **Phase 2 (Foundational)**: after Phase 1. T007→T008→T009 in order. T010, T011, T012, T013→T014, T015, T016, T017, T018 parallel. T019 after T003 and T014.
- **Phases 3 to 6 (US1 to US4)**: after Phase 2. Backend tasks of each story precede its web tasks (the MSW handlers exist from T015, so web work may start once the contract types exist). Stories are ordered by priority. US2, US3, and US4 depend on US1's `respondSession`, `startSession`, `VerifyCodeForm`, and `AuthShell`.
- **Phase 7 (US5)**: docs tasks can run alongside Phases 3 to 6. T054 after Phase 3.
- **Phase 8**: after everything.

### User Story Dependencies

- **US1 (P1)**: only Phase 2. Delivers the MVP.
- **US2 (P2)**: US1 (shared code step, session helpers, mailer from Phase 2).
- **US3 (P3)**: US1 (Google button lives on the rewritten pages). Backend side is independent.
- **US4 (P4)**: US1 (session helpers). Web pages are new files.
- **US5 (P5)**: reads the finished behavior. Docs can be drafted early.

### Within Each User Story

- Tests written and failing first. Then DTO → service → controller/routes → web component → web page.
- Backend checkpoint before web work in the same story.

### Parallel Opportunities

- Phase 1: T004, T005, T006 together after T001.
- Phase 2: T007, T010, T011, T012, T013, T015, T016, T017, T018 together.
- US1: T020, T021, T022, T023, T024 together. Then T025→T026→T027→T028, T029 alongside; T030 and T031 after T019.
- US2: T032, T033, T034 together. Then T035→T036→T037; T038 alongside T037.
- US3: T039, T040, T041 together.
- US4: T043, T044 together. T045→T046→T047; T048, T049 together; then T050.
- US5: T051, T052, T053, T054 together.

## Parallel Example: User Story 1

```bash
# Tests first, in parallel (five files):
#   T020 auth_headless_test.go (sign-in), T021 (verify-email),
#   T022 sign-in.integration.test.tsx, T023 verify-code-form.test.tsx, T024
# Then backend in order: T025 → T026 → T027 → T028 (T029 alongside)
# Then web: T030 → T031
```

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phase 1 and Phase 2.
2. Phase 3 (US1).
3. **STOP and validate**: a verified account signs in on the first-party page, the code step works for an unverified one, the Google link exists but is exercised in US3.

### Incremental Delivery

1. Phase 1 + 2 → plumbing proven by the fake provider and fake mailer.
2. US1 → sign-in demo.
3. US2 → sign-up demo with the "account exists" email in the log.
4. US3 → Google walk with the user.
5. US4 → reset walk from the log link.
6. US5 + Phase 8 → docs, gate, PR.

## Notes

- Sentinel names, comment voice, and the wrapcheck rule follow the constitution. No FR numbers in code comments.
- The Feature 001 `screen` query parameter is removed, not deprecated. The SPA is the only caller and changes in the same PR.
- Every credential endpoint answers `202` for both registered and unregistered addresses where the spec requires neutrality. Tests assert byte-identical bodies apart from `request_id`.
- Tokens, codes, and passwords never appear in test log assertions except to assert their absence.
