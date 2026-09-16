# Data Model: Sign-Up, Sign-In, Landing Page, and Dashboard

**Date**: 2026-09-03 · **Feature**: 001-auth-landing-dashboard

## Entity: User

The platform's record of a registered person. WorkOS owns credentials and
email-verification state. The platform stores what it needs to greet the
user now and to name agreement parties later.

### Table: `users` (migration `000001_users.sql`)

| Column | Type | Constraints | Meaning |
|---|---|---|---|
| `id` | `TEXT` | `PRIMARY KEY` | WorkOS user id (`user_…`). Opaque text. |
| `email` | `TEXT` | `NOT NULL` | The address the provider holds for the user. Kept for the account's lifetime (spec FR-007). |
| `name` | `TEXT` | `NOT NULL DEFAULT ''` | Display name. Empty until the provider supplies one. |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL` | Creation time at the provider. |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL` | The provider's `updated_at`. Ordering authority for sync. |

Index: `users_email_idx` on `lower(email)` (non-unique; future agreement
features look parties up by email, and uniqueness is the provider's
promise, not ours).

### Field derivation (from WorkOS user objects)

- `id` ← `id`.
- `email` ← `email`.
- `name` ← `first_name` + `" "` + `last_name`, trimmed; `''` when both
  are absent.
- `created_at` / `updated_at` ← provider values (ISO 8601 → `timestamptz`).
  Converted to Unix milliseconds at the DTO boundary.

### Sync rules

One upsert serves the callback, the webhook, and the `/me` catch-up:

```sql
INSERT INTO users (id, email, name, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (id) DO UPDATE
  SET email = excluded.email,
      name = excluded.name,
      updated_at = excluded.updated_at
  WHERE excluded.updated_at >= users.updated_at
RETURNING id, email, name, created_at, updated_at;
```

- *Idempotent*: replaying an event changes nothing.
- *Order-safe*: a stale `user.updated` cannot regress `name` or `email`.
- `user.deleted` → `DELETE FROM users WHERE id = $1`; absent row is a
  no-op.
- Catch-up: `GET /v1/me` with a valid token but no row → `GetUser` from
  WorkOS → same upsert → return the row.

### State transitions

```
(absent) --callback | user.created | catch-up--> present
present --user.updated (newer)--> present (fields refreshed)
present --user.updated (stale)--> present (unchanged)
present --user.deleted--> (absent)
```

## Entity: Session cookie (not a table)

Held by the browser, sealed by the API. See
[contracts/session-cookie.md](contracts/session-cookie.md) for the wire
format.

| Field | Meaning |
|---|---|
| `rt` | WorkOS refresh token. Rotates on every refresh. |
| `sid` | WorkOS session id (`sid` claim of the access token). Used for revocation. |
| `uid` | WorkOS user id. Attached to the logger on cookie endpoints. |
| `iat` | Seal time, Unix ms. Informational. |

Lifecycle: created at callback; re-sealed on every successful refresh;
cleared on logout, on refresh rejection, and on callback failure. Expires
by `Max-Age` regardless of activity.

## Conceptual: Access token

A WorkOS-issued RS256 JWT. Held in SPA memory only. Validated per request
against the WorkOS JWKS. Claims used: `sub` (user id), `sid` (session
id), `exp` (drives the refresh schedule). Never persisted anywhere on the
platform.

## Conceptual: State token

Signed, stateless, ten-minute value carried through the hosted sign-in
round trip. Fields: `nonce`, `returnTo`, `exp`. See
[contracts/session-cookie.md](contracts/session-cookie.md).
