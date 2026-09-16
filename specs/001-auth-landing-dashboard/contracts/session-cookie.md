# Contract: Session Cookie, State Token, and Origin Rule

**Date**: 2026-09-03 · **Feature**: 001-auth-landing-dashboard

## Cookie

| Attribute | Value |
|---|---|
| Name | `sow_session` |
| Value | `v1.<kid>.<base64url(nonce ‖ ciphertext)>` |
| Cipher | AES-256-GCM, 12-byte random nonce, no additional data |
| Plaintext | `{"rt":"…","sid":"…","uid":"…","iat":<unix ms>}` |
| `Path` | `/v1/auth` |
| `HttpOnly` | always |
| `SameSite` | `Strict` |
| `Secure` | `staging` and `prod`; omitted in `dev` |
| `Max-Age` | `SESSION_MAX_AGE_SECONDS`, default 604800 |
| `Domain` | none (host-only) |

`SESSION_COOKIE_KEY` is a comma-separated list of `kid:base64(32 bytes)`.
The first entry seals. Every entry opens. A cookie sealed with an unknown
`kid`, a bad tag, or an unparseable payload is treated as absent.

Clearing the cookie sends the same name and path with `Max-Age=0`.

## Endpoints that touch the cookie

| Endpoint | Reads | Writes | Requires `Origin` |
|---|---|---|---|
| `GET /v1/auth/login` | no | no | no |
| `GET /v1/auth/callback` | no | sets or clears | no |
| `POST /v1/auth/refresh` | yes | re-seals or clears | yes |
| `POST /v1/auth/logout` | yes | clears | yes |

No other endpoint reads the cookie. Bearer middleware ignores it.

## Origin rule

`RequireOrigin` compares the `Origin` header byte-for-byte with the
entries in `CORS_ALLOWED_ORIGINS`. Missing or unmatched → `403
{"error":"forbidden_origin"}` before any cookie is opened. CORS on
`/v1/auth` responds with `Access-Control-Allow-Credentials: true` only for
those origins.

## State token

`state = base64url(payload) + "." + base64url(HMAC-SHA256(payload, key))`

`payload = {"n": <16 random bytes, base64url>, "r": <returnTo>, "e": <unix ms>}`

- Key: the active `SESSION_COOKIE_KEY` entry.
- `e` is ten minutes after issue. Expired, malformed, or bad-signature
  states fail the callback with `error=invalid_state`.
- `r` passes `safeReturnTo` on issue and again on verification: must start
  with a single `/`, must not start with `//`, must not contain `\`.
  Anything else becomes `/dashboard`.

## Callback outcomes

| Condition | Redirect |
|---|---|
| Success | `WEB_APP_URL/callback?returnTo=<r>` with the cookie set |
| Missing `code` or `state` | `WEB_APP_URL/?error=flow_incomplete`, cookie cleared |
| State invalid or expired | `WEB_APP_URL/?error=invalid_state`, cookie cleared |
| WorkOS rejects the code | `WEB_APP_URL/?error=flow_incomplete`, cookie cleared |
| WorkOS unreachable | `WEB_APP_URL/?error=provider_unavailable`, cookie cleared |

The callback never renders a page and never exposes provider error
text; the reason category goes to the log with the request id.

## Refresh response

```json
{ "access_token": "<jwt>", "expires_at": 1756250000000 }
```

`expires_at` is the token's `exp` claim in Unix milliseconds. Refresh
never returns the refresh token or the user object.

## Logging

Cookie endpoints log `user_id` from `uid` and the outcome category. The
cookie value, the refresh token, the access token, and the `code` never
appear in logs, metrics labels, or error responses.
