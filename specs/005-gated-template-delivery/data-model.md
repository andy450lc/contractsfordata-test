# Data Model: Gated Word Template Delivery

## Delivery request

- `acknowledged`: required literal true.
- `configuration`: optional complete contract configuration. Absence selects
  the blank public template.
- `email`: required only for email delivery; normalized server-side.
- `Idempotency-Key`: required only for email delivery; opaque ASCII token with
  no personal data.

## Contract configuration

- `template_version`
- `organization`
- `steps.agreement`
- `steps.scope`
- `steps.pricing`
- `steps.environment`
- `steps.technical`
- `steps.delivery`
- `steps.counterparty`

The server rechecks all required text, lengths, dates, numbers, ISO-like country
and currency codes, row bounds, difficulty total, one final milestone, deposit
condition, date ordering, and counterparty-mode requirements.

## Document model

A sanitized, presentation-ready value model derived from validated
configuration or the internal blank model. It contains no email-delivery state
and no unresolved template tokens.

## Artifact

- `bytes`: one complete OOXML ZIP package.
- `filename`: sanitized ASCII-safe `.docx` filename containing no email.
- `mime_type`:
  `application/vnd.openxmlformats-officedocument.wordprocessingml.document`.

The artifact exists only in request memory.

## Dialog state

- `open`
- `step`: disclaimer or delivery
- `has_reached_bottom`
- `acknowledged`
- `email`
- `request_status`: idle, downloading, emailing, success, or error
- `error_message`

Contract configuration is separate state and survives dialog reset.

