# Inbound Contract: WorkOS Webhook (`POST /v1/webhooks/workos`)

**Date**: 2026-09-03 · **Feature**: 001-auth-landing-dashboard (carried over from the marketplace's feature 003)

WorkOS is the caller; this contract states what the receiver guarantees.
Delivery semantics verified against WorkOS docs on 2026-08-27 and unchanged on 2026-09-03.

## Authentication

- Header `WorkOS-Signature: t=<issued-unix-ms>, v1=<hex>` where `<hex>` is
  HMAC-SHA256 over the string `"{t}.{raw request body}"` keyed with
  `WORKOS_WEBHOOK_SECRET`.
- Verification uses the raw, unmodified body bytes (before any JSON
  binding) and rejects timestamps outside a ±5 minute tolerance (replay
  protection).
- Constant-time comparison. Failure → `401 {"error":"unauthorized"}`; the
  payload is not parsed, nothing is logged from it beyond its size.
- Bearer-token auth middleware is NOT applied to this route.

## Events handled

| Event | Effect |
|---|---|
| `user.created` | Order-safe upsert of `users` row (id, email, name, timestamps) from `data` (see data-model.md). |
| `user.updated` | Same upsert; stale events (provider `updated_at` older than stored) change nothing. |
| `user.deleted` | Delete the `users` row; absent row is a no-op. |
| anything else | Acknowledged with 200 and ignored (debug log with event type + id). |

## Delivery semantics the receiver tolerates

- **At-least-once**: duplicate deliveries of the same event id are no-ops
  by construction (idempotent upsert / delete).
- **Out of order**: the provider's `updated_at` inside `data` is the
  ordering authority; older payloads never overwrite newer state.
- **Retries**: WorkOS retries non-2xx with exponential backoff (up to 6
  retries over 3 days in production). Therefore: 200 only after the event
  is durably applied (or deliberately ignored); 5xx on transient store
  failure so the delivery retries.

## Responses

| Status | When | Body |
|---|---|---|
| 200 | Verified and applied, or verified and deliberately ignored | empty |
| 400 | Verified signature but body is not a parseable event envelope | `{"error":"bad_request"}` |
| 401 | Signature missing/malformed/expired/invalid | `{"error":"unauthorized"}` |
| 5xx | Transient processing failure (store unavailable) | `{"error":"internal_server_error"}` |

## Observability

- Log line per processed event: event type, event id, user id — never
  email or name values (PII rule: ids only). The email is stored, not
  logged.
- Signature failures log at Warn with remote IP and reason category only.
