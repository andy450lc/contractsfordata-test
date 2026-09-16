# UI routes and copy

| Route | Page | Auth state | Notes |
|---|---|---|---|
| `/` | Sign in | anonymous | `?redirect=` preserved; `?error=` notices from the Google return leg |
| `/sign-up` | Create account | anonymous | |
| `/forgot-password` | Forgot password | anonymous | |
| `/reset-password?token=` | Choose a new password | anonymous | token moved out of the URL on mount |
| `/callback` | Google return leg | anonymous → authenticated | unchanged |

Every page above except `/reset-password` sends an already-authenticated
visitor to their destination (`redirect` or `/dashboard`). The reset
page serves the link's account regardless of who is signed in: finishing
it replaces the current session with that account's.

## Sign in (`/`)

Card "Sign in". Fields: Email, Password. Button "Sign in". Below: link
"Forgot password?", divider "or", button "Continue with Google", footer
"New to SoW? Create an account".

| Outcome | UI |
|---|---|
| 200 | adopt session, navigate to destination |
| 202 | verify step (below) |
| 401 `invalid_credentials` | alert "That email and password don't match." email kept |
| 400 `validation_failed` | field errors |
| 503 / network | alert "Something went wrong on our side. Please try again in a moment." fields kept |

## Create account (`/sign-up`)

Card "Create an account". Fields: First name, Last name, Email, Password
(helper "At least 8 characters"). Button "Create account". Divider "or",
"Continue with Google", footer "Already have an account? Sign in". The
page reads `?redirect=` like the sign-in page and carries it through the
code step, the Google link, and the sign-in footer link. The sign-in
page's "Create an account" link carries its own `redirect` forward.

| Outcome | UI |
|---|---|
| 202 | verify step |
| 200 | adopt session, navigate to destination (the provider did not require verification) |
| 422 `weak_password` | password field error with `details.message` |
| 400 / 503 | as sign in |

## Verify step (shared)

Card "Check your inbox". Text "We emailed a 6-digit code to {email}."
Field: Code (inputmode numeric, autocomplete one-time-code). Button
"Verify". Secondary: "Send a new code" (disabled for 30 s after each
send, label shows the countdown). Link "Use a different email" returns
to the form.

| Outcome | UI |
|---|---|
| 200 | adopt session, navigate to destination |
| 400 `invalid_code` | alert "That code didn't work. It may have expired." code cleared |
| resend 202 | status "We sent a new code." cooldown restarts |

## Forgot password (`/forgot-password`)

Card "Forgot your password?". Field: Email. Button "Email me a reset
link". After 202: replace the form with "If that address has an account,
we've emailed a reset link. It expires in 15 minutes." and a "Back to
sign in" link. Same text for every address.

## Reset password (`/reset-password`)

Card "Choose a new password". Field: New password. Button "Save and sign
in".

| Outcome | UI |
|---|---|
| no token in URL, or 400 `invalid_reset_token` | card "This link is no longer valid" with "Request a new link" → `/forgot-password` |
| 422 `weak_password` | field error |
| 200 | adopt session, navigate `/dashboard` |

## Error notices on `/` (Google return leg, unchanged codes)

`flow_incomplete`, `invalid_state`, `provider_unavailable`,
`session_expired` keep their Feature 001 copy.
