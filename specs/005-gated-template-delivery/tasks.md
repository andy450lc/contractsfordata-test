# Tasks: Gated Word Template Delivery

## Contract and setup

- [x] T001 Amend OpenAPI with download/email operations and complete schemas.
- [x] T002 Regenerate Go and TypeScript API models with no unrelated diff.

## Backend

- [x] T003 Add failing nested/cross-field validation tests and implementation.
- [x] T004 Add failing OOXML integrity, formatting, clause, sanitization, and
  deterministic filename tests, including the 81-case matrix.
- [x] T005 Implement the neutral model, canonical template, and deterministic
  DOCX package generator.
- [x] T006 Add failing Resend attachment/idempotency/acceptance tests and extend
  the mail abstraction without logging document data.
- [x] T007 Add failing delivery service/controller/route tests for acknowledgment,
  direct download, email equality, cancellation-equivalent no-call behavior,
  provider failure, CORS, rate limiting, and duplicate protection.
- [x] T008 Implement delivery service, controllers, routes, DI, errors, CORS, and
  the dedicated limiter.

## Web app

- [x] T009 Add failing scroll-gate and delivery-dialog tests covering the full
  disclaimer, focus, reset, configuration changes, loading, error, and success.
- [x] T010 Implement request mapping, typed API mutations, and accessible dialog.
- [x] T011 Replace review/list PDF actions, remove or redirect the document route,
  and remove pdfmake/PDF dependencies and obsolete tests.
- [x] T012 Add integration tests proving every configured-document entry point is
  gated and no direct route succeeds.

## Marketing

- [x] T013 Add a test harness and failing tests for header, hero, inline, global
  page, scroll, reset, direct download, email, and failure states.
- [x] T014 Implement the global accessible delivery island and remove all static
  PDF links/fallbacks.
- [x] T015 Update Astro env validation, CSP origin stamping, responsive styles,
  and marketing documentation.

## Documentation and gates

- [x] T016 Update deployment and root documentation, including Resend variables,
  sender-domain verification, CORS, and local unconfigured behavior.
- [x] T017 Run format, lint, typecheck, unit/integration, coverage, production
  builds, codegen-diff, Cloudflare dry-runs, and a repository bypass scan.
- [ ] T018 Inspect both live frontends at mobile and desktop sizes and exercise
  the changed surface with keyboard and pointer input.
