# Implementation Plan: Gated Word Template Delivery

**Branch**: `feature/005-gated-template-delivery`

## Summary

Add two OpenAPI-defined request-body operations to the Go service: direct DOCX
download and DOCX email delivery. Both validate an optional complete contract
configuration plus explicit acknowledgment, build a neutral document model, and
invoke one deterministic OOXML generator. A missing configuration selects the
public blank-template model used by marketing. The existing Resend abstraction
gains attachments, provider idempotency, availability reporting, and strict
acceptance parsing.

Replace the React PDF page and every Astro `/template.pdf` path with responsive
disclaimer/delivery dialogs. Both interfaces measure the real scroll container,
keep unlock monotonic during an attempt, and submit JSON bodies rather than URL
state.

## Implementation Order

1. Add the feature specification and API/UI contracts.
2. Amend `contracts/openapi.yaml`, then regenerate Go and TypeScript models.
3. Add backend validation, document model/template, OOXML package generator,
   mail attachments, delivery service, controllers, routes, and tests.
4. Add the React configurator dialog and API mutations; replace PDF entry points
   and retire PDF code/dependencies.
5. Add the global Astro delivery island, remove static download fallbacks, and
   update CSP/env handling.
6. Update deployment/readme material, run full gates, inspect both live UIs, and
   scan the repository for bypasses.

## Architecture

- `TemplateDeliveryService` owns request validation, blank/configured model
  selection, single generation per request, and email dispatch.
- `DocumentGenerator` returns `{bytes, filename, mimeType}`. The OOXML
  implementation uses Go's ZIP and XML facilities, deterministic entry order,
  and stable timestamps.
- `Mailer` stays provider-neutral. Attachments and provider idempotency are
  message properties. `ResendMailer` is the production implementation.
- Direct/email endpoints are body-only. The email operation requires the
  `Idempotency-Key` header. CORS exposes `Content-Disposition` and allows that
  header.
- Both frontends keep disclaimer state local. Contract configuration remains in
  its existing Zustand store and is serialized only for the delivery request.

## Constitution Check

| Principle | Result | Notes |
|---|---|---|
| Spec-driven | Pass | This feature and OpenAPI contract precede code. |
| Test-first | Pass target | Each implementation lane starts with focused failing tests. |
| Security | Pass with two narrow deviations | Public blank-template delivery has no account auth because marketing has no session. Generated, unsigned template bytes pass through the API because the amended product contract explicitly requires immediate download and direct email attachment. Allowed Origin, strict validation, rate limiting, no storage, and provider idempotency limit the surface. Configured data is caller-supplied and is never persisted. The existing R2 rule continues to govern executed canonical PDFs and signed records. |
| Simplicity | Pass | Standard-library OOXML avoids a second runtime and licensing dependency. |
| Contract integrity | Pass | Both operations are defined in OpenAPI before handlers. |
| Observability | Pass | Logs contain request/provider ids only, with no recipient or document data. |

## Operational Notes

- Production email remains Resend and requires `RESEND_API_KEY` and
  `EMAIL_FROM`; the sender domain must be verified in Resend.
- Marketing requires `PUBLIC_TEMPLATE_DELIVERY_URL` pointing at the API's
  `/v1/template-deliveries` base.
- The marketing origin must appear in backend `CORS_ALLOWED_ORIGINS`.
- Resend documents a 40 MB post-Base64 email size limit and a 24-hour provider
  idempotency window. Generated contracts are expected to remain far below the
  limit.

## Known Process Constraint

The connected GitHub application cannot access `Pixels-Two/SoW`, and the local
GitHub CLI credential is invalid. The repository-mandated issue cannot be
created from this environment. Work remains isolated on the required feature
branch until repository access is restored.
