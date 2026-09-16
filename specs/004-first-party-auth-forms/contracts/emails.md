# Platform-sent emails

Both are plain text, sent through `mail.Mailer` from `EMAIL_FROM`.
Subjects and bodies are fixed strings in `backend/internal/mail/templates.go`.

## Password reset

Trigger: `POST /v1/auth/forgot-password` when WorkOS issued a reset token
for the address.

```text
Subject: Reset your SoW password

Someone asked to reset the password for this SoW account.

Choose a new password here:
{WEB_APP_URL}/reset-password?token={token}

The link works once and expires at {expires_at, RFC 1123, UTC}.

If you did not ask for this, you can ignore this email. Your password
stays the same.
```

## Account exists

Trigger: `POST /v1/auth/sign-up` when WorkOS reports the address already
has an account.

```text
Subject: You already have a SoW account

Someone tried to create a SoW account with this email address, but one
already exists.

Sign in here:
{WEB_APP_URL}/

Forgot your password? Reset it here:
{WEB_APP_URL}/forgot-password

If this was not you, you can ignore this email.
```

## Rules

- Tokens appear only in the reset email body. They never appear in logs
  (the Resend mailer logs the provider message id only).
- In `dev` with no `RESEND_API_KEY`, the log mailer writes the full
  message to the application log so a developer can copy the link.
- A send failure is logged at warn level with the recipient's user id
  when known and never the address. The endpoint still answers `202`.
