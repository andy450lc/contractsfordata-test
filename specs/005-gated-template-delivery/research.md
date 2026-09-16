# Research: Gated Word Template Delivery

## R1. Generator placement

The generator runs in the Go backend. The marketing and web deployments are
static Cloudflare assets, while Resend credentials already live in the Railway
service. Server placement permits authoritative validation and one generator for
download and email.

## R2. DOCX implementation

The generator writes a deterministic OOXML ZIP package with standard-library
`archive/zip` and escaped WordprocessingML. This avoids a second server runtime,
commercial document-library licensing, and a binary template that is absent
from the repository.

## R3. Email provider

Resend remains the provider. Its send-email API accepts Base64 attachment
content and a filename. The attachment also declares the DOCX content type.
Resend's `Idempotency-Key` header suppresses repeated sends for 24 hours. The
service treats only a successful response containing a provider message id as
acceptance.

## R4. Scroll gate

Unlock when `scrollTop + clientHeight >= scrollHeight - 2`. Also unlock when
`scrollHeight <= clientHeight + 2`. Measure on open, scroll, and ResizeObserver
notifications. Read state changes only from false to true during an attempt.

## R5. Public marketing template

The public site submits no configuration. The service creates an internal blank
model with visible editable deal blanks and documented default clause choices.
No file exists at a guessable static URL.

## R6. Matrix

No 81-combination suite existed at inspection time. The feature defines a
reviewable 3⁴ matrix across four conditional clause groups and supplements it
with focused tests for same-group flag combinations.

