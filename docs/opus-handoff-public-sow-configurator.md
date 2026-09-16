# Opus handoff: public SOW configurator

## Starting point

- Repository: `Pixels-Two/SoW`
- Local path: `/Users/andywang/Documents/ChatGPT/Pixelstwo SoW Website`
- Branch: `feature/005-gated-template-delivery`
- Stack: Astro marketing site, Go/Echo API, React/Vite authenticated app.
- The authenticated eight-step configurator at `/agreements/new` is out of
  scope. Do not change its route, request mapping, or document behavior.
- GitHub Actions deploy behavior is also out of scope. In particular, keep
  the existing Wrangler dry-run behavior unless the product owner gives a
  separate instruction.

The worktree is intentionally dirty. The changes already present belong to
the gated-delivery feature and earlier work. Do not reset or discard them.
Inspect the complete diff before committing.

## What is implemented in this turn

### Contract and generated types

`contracts/openapi.yaml` now defines two dedicated public operations:

- `POST /v1/public/sow-configurator/download`
- `POST /v1/public/sow-configurator/email`

The public request carries four typed answers and optional
`client_information` (`delivery_preference`, `party_type`, `company_name`).
The email request also carries an email address and `Idempotency-Key`.

Go types were regenerated into `backend/internal/models/dto/api.gen.go` with
request binders in `backend/internal/models/dto/public-sow-configurator-dto.go`.
The web API schema was regenerated into `web/src/lib/api/schema.d.ts` for the
repository's contract/codegen gate. No authenticated configurator behavior was
intended to change.

The feature spec was amended in
`specs/005-gated-template-delivery/spec.md` with the public four-question
flow, separate public matrix, TODO decisions, and the email exposure rule.

### Go API

New files:

- `backend/internal/controllers/public-sow-configurator-controller.go`
- `backend/internal/routes/public-sow-configurator-routes.go`
- `backend/internal/services/public-sow-configurator-service.go`
- `backend/internal/services/public-sow-configurator-validation.go`
- `backend/internal/services/public-sow-document-generator.go`
- `backend/internal/models/dto/public-sow-configurator-dto.go`

`backend/internal/boot/app.go` wires the public generator, service,
controller, and routes. The public routes use the existing origin check and
public delivery rate limiter.

Server behavior:

- Explicit `acknowledged: true` is required before generation or email.
- Every answer is validated against its enum.
- Field-limited exclusivity requires `exclusivity_field_scope`; scope text is
  bounded, normalized, stripped of XML-invalid characters, and XML-escaped by
  the writer.
- Company name is bounded and shown only as “client reference” text. It is not
  silently assigned to Buyer or Developer.
- A specified delivery preference must match the endpoint being called.
- The generator returns a deterministic genuine OOXML ZIP with
  `[Content_Types].xml` and `word/document.xml`.
- Filename is currently the safe fixed name
  `Robot_Sensor_Data_Content_Development_Agreement.docx`.
- Direct and email generation call the same `PublicSOWDocumentGenerator`.
- Email uses the existing mail abstraction and Resend adapter, sends the DOCX
  bytes as a Base64 attachment with the correct MIME type, and requires an
  accepted provider response plus an idempotency key.

The public document reuses the existing canonical Word styles/package writer,
but applies public branches for:

- subcontracting (prohibited, prior Buyer consent, or notice/objection),
- country jurisdiction rules (Agreement terms, SOW appendix, or rider),
- data/IP liability cap (editable dollar/multiplier blanks, multiplier blank,
  or general cap only), and
- exclusivity (global, field-limited, or non-exclusive).

The document also contains the fixed 24-hour security notice, rights/consents
and no-infringement warranties with protections for minors, nonconsenting
people, and sensitive information, and a general liability tier.

### Astro marketing UI

`marketing/src/components/TemplateDeliveryDialog.tsx` was replaced with a
three-step public flow:

1. `Customize Your SOW`: optional company/type fields and four radio questions.
2. `Before You Download`: exact two-paragraph disclaimer, real scroll
   container, two-pixel bottom tolerance, ResizeObserver recalculation,
   monotonic read state, disabled checkbox until read, and active
   acknowledgment requirement.
3. `Choose Delivery`: direct Word download, plus email field/button only when
   email delivery is explicitly enabled.

The component uses native semantic dialog markup, associated title/description,
focus placement, focus trapping, Escape/Cancel handling, trigger-focus restore,
keyboard-accessible radios and checkbox, live status/error messages, and
duplicate-request suppression while a request is pending. Configuration stays
in memory across close/reopen, while disclaimer and delivery state resets.

`marketing/src/components/TemplateModal.astro` passes the build-time
`PUBLIC_CONFIGURATOR_EMAIL_ENABLED` flag. Its default is false. The example
and deployment docs now point `PUBLIC_TEMPLATE_DELIVERY_URL` at
`/v1/public/sow-configurator`.

## Checks already run

- Marketing public configurator test file: **9 tests passed**.
- Marketing `astro check`: **0 errors, 0 warnings, 0 hints**.
- Go targeted tests: `go test ./internal/services ./internal/controllers
  ./internal/routes`: **passed**.
- Go public generator test covers all 81 combinations of the four public
  three-way questions and checks valid OOXML, fixed clauses, one branch per
  category, sanitization, and no `[TBD]`/`undefined`/`null`/`NaN` output.

The full repository test, lint, typecheck, production build, codegen diff,
and integration gate have **not** been rerun after the latest UI/API changes.
Docker is unavailable in the development environment, so Testcontainers-based
integration tests cannot run locally without external Docker support.

## Work still required

1. Add integration tests for the new public routes in
   `backend/tests/integration/template_delivery_test.go` or a dedicated file:
   origin rejection, false acknowledgment, unknown fields, malformed enums,
   missing field scope, download DOCX headers/package, email attachment byte
   equality, provider rejection, and idempotency conflict.
2. Verify the `fx` graph and run the complete backend test/race/vet/build gate.
3. Run marketing format check, all marketing tests, production build, and
   Wrangler dry-run. Run web typecheck/build/tests and confirm the regenerated
   schema creates no unrelated authenticated-app changes.
4. Review the large pre-existing `web/` diff carefully. The public marketing
   task must not modify `/agreements/new`; restore or reconcile unrelated web
   deletions only with evidence from the earlier feature work.
5. Exercise the real marketing page in the Codex browser against a running Go
   API. Confirm every header, hero, inline, footer, mobile, and 404 trigger
   opens the configurator, and confirm the download response is a real DOCX.
6. Update the deployed marketing environment value in the appropriate hosting
   configuration from the old generic base to
   `https://api.<stage>.<domain>/v1/public/sow-configurator`. Do not change the
   GitHub Actions deploy step itself.
7. Decide whether client information remains optional. The current code keeps
   it optional and does not map company name to a contract party.
8. Decide the exact liability dollar floor and fee multipliers. The current
   UI shows `[TBD]`, while the DOCX intentionally uses editable underscore
   blanks and contains no unresolved `[TBD]` token.
9. Decide whether jurisdiction overlays remain fixed content (current behavior)
   or become a fifth question. Do not invent a fifth question in code.
10. Configure and verify email before enabling it in production:
    `RESEND_API_KEY`, `EMAIL_FROM`, a verified Resend sending domain, API CORS
    allowlist for the marketing origin, and `PUBLIC_CONFIGURATOR_EMAIL_ENABLED=true`.
    The environment has no production credentials, so no real provider send has
    been claimed here. Keep the flag false until a real DOCX attachment send
    is accepted end to end.
11. Update owner-facing setup documentation if the provider or hosting
    decision changes. Do not commit secrets.

## Important implementation cautions

- Do not route public requests through `/agreements/new` or the old
  authenticated document mapping.
- Do not reintroduce a static PDF, browser PDF blob, direct `.docx` anchor, or
  generic-template endpoint as a bypass.
- Do not expose an email option merely because the Go mailer is constructed.
  The current UI intentionally hides it unless the explicit build flag is true.
- Do not use the displayed liability `[TBD]` strings as document output.
- Keep recipient email and configuration out of URLs and unnecessary logs.
- Preserve the existing Resend/provider abstraction and idempotency behavior.

## Suggested continuation sequence

1. Read this file, `specs/005-gated-template-delivery/spec.md`, the complete
   `git diff`, and the repository constitution.
2. Add the public route integration tests before broad refactoring.
3. Fix any compile/codegen/format issues found by the full gates.
4. Run the browser smoke test with the API running.
5. Review the deployment environment and email verification decision with the
   product owner.
6. Commit only after the complete gate is green and document any unavailable
   Docker/provider checks honestly in the PR.

## Continuation effort estimate

The remaining implementation and verification is approximately one focused
Opus-sized pass: public route integration tests, full codegen/build/lint/test
gates, a browser smoke test, and cleanup of any generated or pre-existing diff
issues. A practical estimate is 1–3 hours of wall-clock work when Docker and a
running API are available, or 2–4 agent turns if infrastructure failures need
diagnosis. A real Resend verification adds a separate product-owner step and
cannot be completed from this checkout without provider credentials and a
verified sending domain.
