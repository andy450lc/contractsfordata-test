# Research: First-Party Sign-In and Sign-Up Forms

**Date**: 2026-09-04 · **Feature**: 004-first-party-auth-forms

No NEEDS CLARIFICATION markers remained after Technical Context. The
decisions below were verified against the installed `workos-go/v10`
v10.3.0 sources (method names, request paths, grant types, error struct)
and the WorkOS API reference on 2026-09-04.

## R1. WorkOS facts the design rests on

- **Password sign-in**: `AuthenticateWithPassword` posts
  `grant_type=password` to `/user_management/authenticate`. On an
  unverified email WorkOS answers 4xx with `code:
  email_verification_required`, a `pending_authentication_token`, and an
  `email_verification_id`, and, when the dashboard email setting is on,
  **sends the 6-digit code itself** ("WorkOS will automatically send a
  one-time email verification code to the user's email address"). The
  SDK surfaces those fields on `*workos.APIError`.
- **Code exchange**: `AuthenticateWithEmailVerification` posts
  `grant_type=urn:workos:oauth:grant-type:email-verification:code` with
  the code and pending token and returns the same `AuthenticateResponse`
  as a code exchange (user, access token, refresh token).
- **Resend**: `SendVerificationEmail(userID)` posts to
  `/user_management/users/{id}/email_verification/send` and "sends an
  email that contains a one-time code". It needs the user id, so resend
  looks the user up by email first (`List` with an email filter).
- **Sign-up**: `Create` posts to `/user_management/users` with `email`,
  `first_name`, `last_name`, and a plaintext `Password`. WorkOS validates
  strength server-side. A duplicate email is a 4xx. The exact error code
  is not documented, so the service treats any 4xx from `Create` that is
  not a password-strength error as "email taken".
- **Password reset**: `ResetPassword(email)` posts to
  `/user_management/password_reset` and returns `password_reset_token`,
  `password_reset_url`, and `expires_at`. **WorkOS sends no email for
  this**: the older `SendPasswordResetEmail` is deprecated in favor of
  this token-issuing call, and the app emails the link.
  `ConfirmPasswordReset(token, newPassword)` returns the user, verifies
  the email as a side effect, and revokes the user's active sessions.
- **Google**: `GetAuthorizationURL` with `Provider: GoogleOAuth` and no
  `ScreenHint` produces a URL that sends the browser through WorkOS to
  Google's consent screen with no chooser page. The return leg is the
  registered redirect URI, unchanged.

## R2. Endpoint set

- **Decision**: Six `POST` operations under `/v1/auth`, all JSON, all in
  the existing group (stricter limiter), all `Origin`-checked and DTO
  validated:
  - `sign-in {email, password}` → `200 AccessToken` + cookie, or
    `202 PendingVerification {pending_token}` when the email is not yet
    verified, or `401 invalid_credentials`.
  - `sign-up {first_name, last_name, email, password}` →
    `202 PendingVerification` always (see R4), or `422 weak_password`.
  - `verify-email {pending_token, code}` → `200 AccessToken` + cookie,
    or `400 invalid_code`.
  - `resend-verification {email}` → `202` always.
  - `forgot-password {email}` → `202` always.
  - `reset-password {token, password}` → `200 AccessToken` + cookie, or
    `400 invalid_reset_token`, or `422 weak_password`.
  - `GET /auth/login` keeps its shape with `screen` replaced by
    `provider` (enum `google`, required). `callback`, `refresh`, and
    `logout` are unchanged.
- **Rationale**: Returning `AccessToken` (the refresh response shape)
  from the endpoints that start a session lets the SPA adopt the session
  in one step with no extra refresh round trip. `202` marks "accepted,
  no session yet" without a discriminated union in one `200` body.
  Sign-up and forgot-password answer identically for every email so
  nothing distinguishes registered addresses (spec FR-007, FR-012,
  FR-015).
- **Alternatives considered**: Returning the pending token inside a
  `200` with a `status` field (rejected: two success shapes under one
  code). A separate `GET /auth/google` (rejected: `login` already builds
  provider URLs and carries the signed state).

## R3. Error classification

- **Decision**: `providerError` keeps its 4xx→`ErrUnauthorized`,
  else→`ErrProviderUnavailable` rule. `AuthService` adds one classifier,
  `passwordAuthError`, that inspects `*workos.APIError`:
  `email_verification_required` with a pending token → the
  `PendingVerification` result (not an error). Any other 4xx →
  `ErrInvalidCredentials`. 5xx or transport → `ErrProviderUnavailable`.
  `createUserError` maps a 4xx whose field errors name `password` to
  `ErrWeakPassword` carrying the provider message in `details.message`,
  and every other 4xx to the "email taken" branch. Verification and
  reset confirmations map 4xx to `ErrInvalidCode` /
  `ErrInvalidResetToken`.
- **Rationale**: The SPA needs a closed set of codes. The provider's
  message is shown only for `weak_password`, where it is the reason the
  user needs and reveals nothing about other accounts.
- **Alternatives considered**: Passing provider codes through (rejected:
  leaks `user_not_found`-style distinctions and couples the SPA to
  WorkOS wording).

## R4. Duplicate sign-up and the decoy pending token

- **Decision**: When `Create` reports the email is taken, the service
  sends the "account exists" email through the mailer and returns a
  `PendingVerification` whose `pending_token` is 32 random bytes,
  base64url, shaped like a real one. A later `verify-email` with it
  reaches WorkOS, which rejects the unknown token, so the SPA sees
  `invalid_code` exactly as for a wrong code. `resend-verification` by
  email finds the existing verified user and sends nothing (WorkOS
  refuses to send a code to a verified address). The response is `202`
  in every branch.
- **Rationale**: Spec FR-012 and US2 scenario 6. No platform state is
  needed to make the decoy behave.
- **Alternatives considered**: Answering `409` (rejected: enumeration).
  Signing the decoy so `verify-email` can short-circuit (rejected: more
  code for the same observable behavior).

## R5. Platform-sent email

- **Decision**: New package `internal/mail` with `Mailer` (`Send(ctx,
  Message) error`), `Message{To, Subject, Text}`, a `ResendMailer` that
  posts `{from, to, subject, text}` to `https://api.resend.com/emails`
  with `Authorization: Bearer RESEND_API_KEY` through the shared
  `net/http` client, and a `LogMailer` that writes the message at info
  level. `boot.NewMailer` returns `ResendMailer` when `RESEND_API_KEY`
  is set and `LogMailer` otherwise. Config requires `RESEND_API_KEY` and
  `EMAIL_FROM` when `Deployed()`. Two plain-text templates in
  `templates.go`: password reset (link, expiry) and account exists
  (sign-in link, forgot-password link). Sending failures on
  `forgot-password` and duplicate `sign-up` are logged and still answer
  `202`.
- **Rationale**: R1 (WorkOS sends no reset email). Resend was chosen as
  the project's transactional provider because the send-agreement
  feature needs one, its free tier and single HTTP call fit, and no SDK
  is needed. The log mailer keeps local development usable with no
  account. The spec's Assumptions were amended to record this.
- **Alternatives considered**: `net/smtp` (rejected: frozen package, a
  TLS and auth surface for no gain). The Resend Go SDK (rejected: one
  endpoint does not earn a dependency). Sending reset emails from the
  SPA via a mail link (rejected: no).

## R6. Sign-in after password reset

- **Decision**: After `ConfirmPasswordReset` succeeds, the service calls
  `AuthenticateWithPassword` with the user's email from the response
  and the new password, seals the cookie, and answers `200 AccessToken`.
- **Rationale**: Spec FR-017 ("MUST sign the user in"). The confirm call
  returns no tokens. The password is in hand for one more call.
- **Alternatives considered**: Answer `204` and make the user sign in
  (rejected: spec).

## R7. Google link

- **Decision**: `AuthorizationURL(state, provider)` takes the provider
  name. `google` maps to `UserManagementAuthenticationProviderGoogleOAuth`
  with no screen hint. `LoginQueryDto.Provider` is `validate:"required,
  oneof=google"`. The SPA's `googleSignInUrl(returnTo)` builds
  `/v1/auth/login?provider=google&returnTo=…`. Callback, state, and
  `returnTo` handling are unchanged.
- **Rationale**: Spec FR-013. The `authkit` provider value is removed so
  no code path can reach the hosted page.
- **Alternatives considered**: Keep `screen` and add `provider`
  (rejected: `screen` has no meaning without the hosted page).

## R8. Identity linking

- **Decision**: Rely on WorkOS identity linking: a Google sign-in whose
  verified email matches an existing user attaches to that user. The
  platform's `UpsertFromProvider` then sees the same user id. Verified
  in the manual walk (quickstart step 6). No platform matching code.
- **Rationale**: Spec FR-014 and Principle IV.
- **Alternatives considered**: Matching by email in the users table
  (rejected: two sources of truth for identity).

## R9. Web session adoption

- **Decision**: `lib/auth/session.ts` exports `adoptSession(token:
  AccessToken)`: it stores the token, schedules the pre-expiry refresh,
  and clears the signed-out sentinel. The auth mutations call it on
  `200`. The cookie set by the API response is stored by the browser
  because the calls use `credentials: 'include'`, the same as refresh.
- **Rationale**: One round trip to sign in. The callback page keeps its
  refresh-based adoption for the Google leg.
- **Alternatives considered**: Navigate to `/callback` after every
  sign-in (rejected: an extra refresh call and a loader flash).

## R10. Forms and the two-step pages

- **Decision**: Each page is a `react-hook-form` form with `zodResolver`
  over a colocated schema in `features/auth/schemas.ts`: `email`
  (`z.email()`, trimmed, ≤254), `password` (8–256, untrimmed), `firstName`
  and `lastName` (trimmed, 1–100), `code` (`/^\d{6}$/`). Sign-in and
  sign-up hold a `step` state: `'form' | { kind: 'verify', email,
  pendingToken }`. The shared `VerifyCodeForm` owns the code input, the
  resend button with a 30-second cooldown, and the `invalid_code`
  notice. Server `validation_failed` details map onto fields through the
  existing `fieldErrorsFrom` helper. Submit buttons disable while the
  mutation is pending.
- **Rationale**: Frontend conventions §5. The step lives in component
  state, not the URL, because the pending token must not appear in the
  address bar or history.
- **Alternatives considered**: A `/verify` route (rejected: token in
  URL). shadcn `Form` wrappers (not installed. The feature-002 `Input` +
  `Label` + error paragraph pattern is reused).

## R11. Reset page token handling

- **Decision**: `/reset-password?token=…` reads `token` into state on
  mount, then calls `history.replaceState` to drop the query. Missing
  token renders the "link no longer valid" state with a link to
  forgot-password. Success adopts the session and navigates to
  `/dashboard`.
- **Rationale**: Spec FR-020.
- **Alternatives considered**: Hash fragment in the link (rejected:
  WorkOS builds the URL from the dashboard setting with `?token=`).

## R12. Validation caps and messages

- **Decision**: Backend DTOs: `email` `required,email,max=254`,
  `password` `required,min=8,max=256`, names `required,max=100`, `code`
  `required,len=6,numeric`, `pending_token` and `token` `required,
  max=512`. Names and email are trimmed in the service. The web zod
  schemas mirror the caps. User-facing copy per contracts/ui-routes.md.
- **Rationale**: Spec edge case "very long or unusual input" and FR-004.

## R13. Rate limiting

- **Decision**: Cloudflare rate-limiting rule on the API zone: match
  `http.request.method eq "POST" and starts_with(http.request.uri.path,
  "/v1/auth/")`, characteristic `ip.src`, 10 requests per 1 minute,
  action Block for 10 minutes, plus a second rule at 5 per minute with
  Managed Challenge on `sign-in`, `sign-up`, `forgot-password`, and
  `resend-verification`. Documented in `docs/deployment.md` under a new
  "Cloudflare rate limiting" heading. The API's `AUTH_RATE_LIMIT_*` per-IP
  limiter stays as the second line. The SPA's resend cooldown is a UX
  guard only.
- **Rationale**: Spec FR-019 and the user's instruction to lean on
  Cloudflare. Thresholds are a starting point recorded with the rule.
  WorkOS Radar was evaluated on 2026-09-04: it supports the headless
  API (`signals_id`, `ip_address`, `user_agent`, `radar_auth_attempt_id`
  on create and authenticate, `policy_denied` and
  `radar_email_challenge` answers, an `@workos/radar-signals` browser
  library that loads a script from the WorkOS CDN and posts to
  api.workos.com), but it is billed per check beyond 1,000 a month
  ($100 per 50K). The user chose Cloudflare only. Radar can be added
  later as its own feature.

## R14. Password method precondition

- **Decision**: No startup probe. WorkOS has no API to read the
  environment's enabled authentication methods. If password auth is off,
  `AuthenticateWithPassword` fails as a 4xx, which the classifier turns
  into `invalid_credentials`. That would mislead users, so
  `docs/deployment.md` "Standing up a new stage" gains the dashboard
  checklist (password on, email verification on, Google on, redirect
  URI, password-reset URL) and quickstart.md step 1 walks it before the
  first sign-in.
- **Rationale**: Spec FR-024 offers this option.
- **Observed 2026-09-04**: the password grant answers 400
  `sso_required` (OAuth-style `error` field, with `connection_ids`)
  when the address's domain matches an SSO connection. The WorkOS
  staging environment ships a test connection for `example.com`, so
  a bogus `@example.com` address triggers it even with Email +
  Password on. That code maps to `ErrProviderUnavailable` so the page
  shows the neutral "something went wrong" notice, never "wrong
  password". SSO sign-in is out of scope.
- **Observed 2026-09-04**: the password grant answers **403**
  `email_verification_required` with the pending token for an
  unverified account. The classifier checks that code before the
  status-based refusal filter, since 403 otherwise reads as a
  provider problem.

## R15. Marketplace cross-reference

- The marketplace repository has no password or mailer code to copy. Its
  issue #268 (token-mediating backend) and #1 (custom AuthKit domain)
  receive a comment pointing at this feature as the way to drop the
  hosted page and the custom domain in one move.

## R16. Review findings folded in (2026-09-04)

An adversarial read of the first implementation produced these
decisions:

- `providerRefusal` treats only business refusals as refusals. Provider
  answers 401, 403, 408, and 429 mean the platform's own credentials
  or quota are the problem and map to `ErrProviderUnavailable`.
- The registered and unregistered branches of `sign-up`,
  `forgot-password`, and `resend-verification` return at the same
  point. Outbound mail and the provider's resend call run on a
  goroutine with `context.WithoutCancel`, and the Resend HTTP client
  carries a 10 s timeout.
- Classifiers wrap a sanitized description (status and code) rather
  than the provider error string, which can carry a pending token.
- A sign-up that the provider accepts without verification starts the
  session and answers `200 SessionStarted`. The contract gains that
  response on `sign-up`.
- The reset page does not bounce a signed-in visitor (spec edge case
  wins over the first draft of ui-routes.md).
- The dev-only log mailer is a documented FR-020 exception.
- The decoy pending token matches the shape of a real provider token.
  The exact shape is confirmed in the browser walk and recorded on
  issue #115.
