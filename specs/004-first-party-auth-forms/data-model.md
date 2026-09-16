# Data Model: First-Party Sign-In and Sign-Up Forms

**Feature**: 004-first-party-auth-forms · No database change.

## Request and response shapes (API)

Full definitions in [contracts/auth-endpoints.yaml](contracts/auth-endpoints.yaml).

| Schema | Fields | Rules |
|---|---|---|
| `Credentials` | `email`, `password` | email ≤ 254 and well-formed, password 8–256 |
| `SignUpRequest` | `first_name`, `last_name`, `email`, `password` | names 1–100 after trim, others as above |
| `PendingVerification` | `pending_token` | opaque, ≤ 512, single use, short-lived at the provider |
| `VerifyEmailRequest` | `pending_token`, `code` | code exactly six digits |
| `EmailRequest` | `email` | as above |
| `ResetPasswordRequest` | `token`, `password` | token opaque ≤ 512, password 8–256 |
| `AccessToken` | `access_token`, `expires_at` | unchanged from Feature 001 |
| `Error` | `error`, `request_id`, `details` | unchanged. `weak_password` carries `details.message` |

Error codes introduced: `invalid_credentials` (401), `invalid_code`
(400), `invalid_reset_token` (400), `weak_password` (422). Existing:
`validation_failed` (400), `forbidden_origin` (403), `too_many_requests`
(429), `provider_unavailable` (503).

## Session cookie

Unchanged from `specs/001-auth-landing-dashboard/contracts/session-cookie.md`.
Every endpoint that answers `200 AccessToken` also sets the cookie the
callback sets, from the same `AuthenticateResponse` fields.

## Pending verification (client-held)

Held in the sign-in or sign-up page's component state as
`{ kind: 'verify', email, pendingToken }` for the life of the code step.
Never written to the URL, `localStorage`, or the session store. Dropped on
success, on navigation away, and on "start over".

## Password reset token

Issued by WorkOS on `forgot-password`. Reaches the user once, in the
emailed link `WEB_APP_URL/reset-password?token=<token>`. The reset page
moves it from the URL into component state on mount. The platform never
stores it and never logs it.

## Mail message

```text
Message { To string, Subject string, Text string }
```

Two templates, see [contracts/emails.md](contracts/emails.md). Sent
through `mail.Mailer`. The log mailer (dev only) writes the whole message
at info level. The Resend mailer posts it and logs only the provider's
message id.

## Backend service results

```text
SignInResult   = Session | PendingVerification
Session        { CookieValue, AccessToken, ExpiresAt, UserID }
PendingVerification { PendingToken }
```

`SignUp` always yields `PendingVerification`. `VerifyEmail` and
`ResetPassword` yield `Session`. `ResendVerification` and
`ForgotPassword` yield nothing.

## State transitions (web)

```text
sign-in page:   form --200--> session adopted --> returnTo
                form --202--> verify step --200--> session adopted --> returnTo
                verify step --400 invalid_code--> verify step (notice)
                verify step --resend (202)--> verify step (cooldown 30 s)
sign-up page:   form --202--> verify step (same as above, returnTo /dashboard)
                form --422 weak_password--> form (password field error)
forgot page:    form --202--> "check your inbox" (always)
reset page:     mount without token --> invalid-link state
                form --200--> session adopted --> /dashboard
                form --400 invalid_reset_token--> invalid-link state
                form --422 weak_password--> form (password field error)
any page:       status === 'authenticated' --> Navigate to returnTo
```
