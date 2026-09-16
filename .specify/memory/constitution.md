<!--
Sync Impact Report
==================
Version change: 1.3.0 → 1.3.1
Rationale: PATCH — clarifies how the landing site may ship client-side
JavaScript now that it has its first interactive element (the template
dialog). A scoped `<script>` in the Astro component that needs it is the
default; a React island is for components that hold state. Scripts are
always bundled files, so the CSP stays at `script-src 'self'` with no
inline scripts, and any origin the site fetches from is stamped into
`connect-src` at build time.

Amended sections:
  - Project Structure & Stack: the "Marketing" bullet's JavaScript and
    CSP sentences.
Companion changes: `marketing/public/_headers`, `marketing/README.md`,
`docs/deployment.md`.

Previous report (1.2.0 → 1.3.0):
Version change: 1.2.0 → 1.3.0
Rationale: MINOR — the landing page moves from hand-written HTML/CSS to
an Astro site under `marketing/`. The apex domain must rank in search,
and a single static file stops scaling at the second page (shared
header and footer, per-page metadata, a sitemap). Astro renders every
page to HTML at build time, so the output is as static and as fast as
before; React is available for islands but no page ships client-side
JavaScript unless it needs interactivity. Server-side rendering is not
adopted: nothing on the site varies per request.

Amended sections:
  - Project Structure & Stack: the "Marketing" bullet rewritten for
    Astro with static output, SEO metadata requirements per page,
    shared design tokens, the no-client-JS default, and the conditions
    under which a route may render on the server.
Companion changes: `docs/monorepo-and-ci.md` (layout and build step),
`docs/frontend-conventions.md` §10 (pointer to the marketing site),
`docs/deployment.md` (landing page build and env vars), `README.md`,
and `.github/workflows/marketing.yml`.

Previous report (1.1.0 → 1.2.0):
Version change: 1.1.0 → 1.2.0
Rationale: MINOR — the hosted WorkOS login page is removed in favor of
first-party sign-in, sign-up, verification, and password reset screens
in the web app. The API gains six JSON endpoints under `/v1/auth` that
exchange credentials, verification codes, and reset tokens with WorkOS
server-side and seal the existing session cookie. Social sign-in keeps
using a redirect to the provider's own consent screen, now selected by
a `provider` query parameter, still returning through the API callback.
This avoids the WorkOS custom-domain dependency the hosted page would
otherwise reintroduce for a first-party look and feel.

Amended sections:
  - Authentication: items 1 and 2 rewritten for first-party screens and
    the six JSON endpoints. Item 7 notes WorkOS's hosted pages are not
    used. New item 9 (rate limiting and enumeration protection on
    `/v1/auth/*`) and item 10 (the project's transactional mail
    provider sends password-reset and account-exists email, WorkOS
    sends verification codes).
  - Development Workflow: the "Subagent model and effort match the
    task" bullet gains a sentence on the orchestrating session
    delegating implementation to the implement workflow or spawned
    implementers rather than writing feature code itself.
Companion changes: `docs/backend-conventions.md`, `docs/frontend-
conventions.md`, and `contracts/openapi.yaml` updated to match.

Previous report (1.0.0 → 1.1.0):
Version change: 1.0.0 → 1.1.0
Rationale: MINOR — the Authentication section is redefined around a
token-mediating backend (the BFF pattern for OAuth). The WorkOS browser
SDK refreshed sessions with a cookie on the WorkOS host, which browsers
treat as third-party from the app's origin. Silent refresh then fails
unless each production environment buys a $99/month WorkOS custom
domain, and staging environments cannot have one at all. Moving the
code exchange and refresh onto the Go API keeps the refresh token in a
first-party HttpOnly cookie on the API host, so refresh works on every
browser and every stage with no custom domain.

Amended sections:
  - Authentication: the API owns login, callback, refresh, and logout.
    The web app holds only the in-memory access token and calls
    `/v1/auth/refresh`. Bearer validation against the WorkOS JWKS is
    unchanged. The WorkOS browser SDK and `devMode` are not used.
Companion changes: `docs/backend-conventions.md` §10 Auth,
`docs/frontend-conventions.md` §9, and `docs/deployment.md` (env vars,
WorkOS redirect URI) updated to match.

Previous report (1.0.0):
  Version change: new document → 1.0.0. Initial ratification for the
  SoW project, derived from the PixelsTwo Marketplace constitution 1.9.0
  (last amended 2026-09-01).
Every principle, the comment-voice rules, and every Development
Workflow rule carry over unchanged in substance. Adapted for this
project:
  - Title and project description.
  - Principle III (Security): the sensitive data is signed commercial
    agreements, the parties' PII, and signature evidence rather than
    egocentric video. Presigned storage access, PII-free logs,
    encryption, retention, secrets, and dependency-scanning rules are
    kept. New rules: signature evidence is append-only and
    tamper-evident; magic-link tokens are single-purpose, expiring,
    stored hashed, and never logged.
  - Principle V: idempotency keys protect endpoints that send an
    agreement, record a signature, or send email (the marketplace rule
    covered money and orders).
  - Project Structure & Stack: Go backend on Railway, Supabase
    Postgres, Cloudflare R2 for documents, React SPA on Cloudflare
    Workers static assets, and a static `marketing/` landing page.
  - Authentication: unchanged for account holders. New section on
    signer access by magic link with no account.
  - Binding Convention Documents: the same three documents, copied and
    adapted, plus `docs/deployment.md` and `docs/project-context.md`
    as informative references.
Templates: `.specify/templates/*.md` are the stock Spec Kit templates
and need no change.
Deferred items / TODOs: the product name is the placeholder "SoW"
until the user picks one. Domain, email provider, and PDF renderer
are decided in feature specs and recorded in `docs/deployment.md`.
-->

# SoW Constitution

SoW lets a sender create a Statement of Work from a structured
template, send it to one or more signers by email, and collect legally
meaningful signatures on a canonical PDF. It is to Statements of Work
what the YC SAFE is to seed investment: one standard document, filled
in and signed online, stored with its evidence.

## Core Principles

### I. Spec-Driven Development

Every feature MUST begin as a written specification under `specs/` before
any implementation code is written. For features with an API surface, the
flow extends: feature spec → change to `contracts/openapi.yaml` → implement
to the contract — all on one branch, one PR. Code that has no governing
spec MUST NOT be merged. When implementation reveals a spec is wrong, the
spec (and contract) is amended first and the change re-derived from it.

**Rationale**: intent written before code keeps scope explicit and lets any
contributor — human or agent — verify behavior against a stated contract.

### II. Test-First & Meaningful Coverage

TDD is mandatory from day 1: the test is written and failing before the
implementation exists. **Integration tests outrank unit tests** — the
primary suites exercise real request paths (backend: route → service →
store → real Postgres via testcontainers; frontend: full feature render
with the network mocked only at the HTTP boundary per the OpenAPI
contract). Unit tests supplement where logic density earns them.

The aim is **100% meaningful coverage**; CI hard-gates at **85% on
new/changed code**, excluding generated code (sqlc, oapi-codegen, mocks,
generated API client, shadcn `components/ui`) and pure wiring (`main.go`,
`boot/`). **Test hacking is banned**: tests MUST assert behavior — outputs,
state changes, error values — not merely execute lines. Assertion-free,
tautological, or mock-what-you-test tests are review-blocking defects.
Bug fixes MUST include a regression test that fails on the pre-fix code.

**Rationale**: integration-first testing verifies the system users
actually hit; the meaningful-coverage rule keeps the 100% aim from
degenerating into the filler tests it exists to prevent.

### III. Security is Paramount

This project stores signed commercial agreements: their contents, the
names, emails, and signatures of the parties, and the evidence that a
signature happened. Security is weighed in every decision.

- Every endpoint MUST implement authentication, authorization, input
  validation, and rate limiting — enforced structurally via middleware,
  not per-handler discipline. Magic-link endpoints are authenticated by
  the link token and rate limited more strictly than account routes.
- Documents MUST NOT flow through the API: rendered PDFs live in R2 and
  are read via short-lived presigned URLs; any upload (attachments,
  logos) goes direct-to-storage via presigned flows with server-side
  validation on completion.
- Signature evidence is append-only and tamper-evident: a recorded
  signature (signer identity, timestamp, IP, user agent, hash of the
  exact document signed) is never updated or deleted, and the rendered
  PDF is immutable once the first signature lands. Every later state
  change references the prior record.
- Magic-link tokens are single-purpose, expire, are stored hashed, and
  never appear in logs, analytics, or API query strings.
- Sensitive data is encrypted at rest; access to agreements is audit
  logged; PII never appears in logs, metrics labels, or error messages
  (ids only). Data retention and deletion MUST be specified in the
  governing feature spec before any such data is collected.
- Secrets never enter the repository or the frontend bundle; frontend
  tokens live in memory only, never `localStorage`.
- Dependency scanning (`govulncheck`, npm audit) runs in CI; third-party
  session-recording of user content (e.g. Sentry Session Replay) is
  prohibited.

**Rationale**: a product whose whole value is "this person signed this
document at this time" has nothing left if the evidence can be altered
or the document leaked.

### IV. Simplicity, Patterns & Human Readability

Start with the simplest design that satisfies the current spec; speculative
generality is prohibited and added complexity MUST be justified against a
simpler rejected alternative. Within that constraint, use proper design
patterns — repository (stores), dependency injection (fx), factory (client
factories), and others — where they carry their weight. The bar for all
code: **the maintainer can read, reason about, and maintain it without help
from agents.** Cyclomatic complexity per function MUST be ≤ 10 (to 15 only
with written justification; above 15 is refactor-first), enforced in CI.

**Comment voice**: a comment describes what the code does, in isolation.
Doc comments on exported identifiers keep the godoc form — they begin
with the identifier name and read as a sentence — and everything after
that opening states behavior plainly.

Sentence form:

- Simple sentences. Present tense, active voice. One idea per sentence
- A compound sentence has at most two clauses
- Contrastive negation is banned — no "X, never Y", no "X, not Z".
  State the positive fact. A negative fact that matters gets its own
  sentence
- Conditions come before actions when a comment gives guidance

Word choice:

- Plain words in their primary sense. No idioms, no figurative language,
  no placeholder phrases, no "simply", "easy", or "just"
- At most two stacked noun modifiers
- Serial comma. Standard American spelling

Banned in all comments, project-wide:

- semicolons — write separate sentences instead
- references to specs, requirements, or convention documents — no
  FR-numbers, no section citations, no research or decision ids.
  Traceability lives in `specs/`, never in code
- justifying or defending the design — no contrasts with rejected
  alternatives, no cross-project context, no compliance notes. Rationale
  lives in the spec, the plan, or the PR

Line wrapping:

- A sentence that fits on one line stays on one line
- A wrapped sentence leaves no continuation line of one to three words.
  Rebalance the break or shorten the sentence

The Google developer documentation style guide
(https://developers.google.com/style) is incorporated by reference as
the tie-breaker for anything these rules leave unstated. This
constitution wins conflicts.

A comment earns its place only by stating behavior or a constraint the
code cannot show. Everything else is deleted, not reworded.

**Rationale**: patterns and simplicity are not in tension — patterns applied
where necessary are what keep a codebase legible; patterns applied
everywhere are what YAGNI exists to stop. Comments that narrate, cite, or
defend force the reader to hold the whole project in their head to
understand one function — the opposite of readable in isolation.

### V. Contract Integrity & RESTful API

The API follows REST principles strictly: proper HTTP methods, status
codes, and resource naming; stateless endpoints; idempotency where
appropriate (idempotency keys MUST protect any endpoint that sends an
agreement, records a signature, or sends email). All routes live under
`/v1`. The OpenAPI document at `contracts/openapi.yaml` is the single
source of truth, written before implementation; both apps prove
conformance via CI generation-diff gates, and all request and response
DTOs on both sides are generated from it. API timestamps are Unix
milliseconds (int64) everywhere. Breaking contract changes MUST be
flagged in the feature spec with a migration path — never shipped
silently.

**Rationale**: the contract is where two codebases and every future client
meet; it is the one place where drift is unaffordable.

### VI. Observability & Debuggability

The observability stack — structured JSON logs, OTel tracing, Prometheus
RED metrics + saturation gauges, and Sentry — is wired at boot, before the
first feature ships. The error-log contract is binding: every log line
carries `request_id`, `trace_id`, `span_id`, and `user_id` (by
construction, via context — not manual discipline); errors are wrapped
with operation context at every boundary (`wrapcheck`-enforced), logged
exactly once at the central handler, and carry the ids needed to
reproduce. Traces propagate browser → backend → downstream services via
W3C `traceparent`, so one trace id reconstructs any request's full story
across the system. Sentry receives unexpected errors only, and is enabled
only in deployed environments. Errors are surfaced, never swallowed.

**Rationale**: an error you cannot act on is a bug in itself; correlation
by construction is what makes a multi-service system debuggable by one
person.

## Project Structure & Stack

- **Monorepo** (see `docs/monorepo-and-ci.md`): `contracts/` (OpenAPI
  source of truth), `backend/` (Go), `web/` (React SPA), `marketing/`
  (static landing page), `docs/` (conventions), `.specify/` (this
  constitution + specs). CI is path-filtered per app; `contracts/**`
  changes trigger the backend and web pipelines.
- **Backend**: Go — echo v5, slog, uber fx, goose migrations, pgx v5,
  sqlc (typed SQL only), cockroachdb/errors for unexpected-error stacks.
  Deployed on Railway. Postgres is Supabase. Documents (rendered PDFs,
  attachments) live in Cloudflare R2, reached through its S3 API.
- **Frontend**: React + Vite + TypeScript (strict, no `any`), shadcn/ui +
  Tailwind, react-router-dom (URL is the primary navigation state),
  TanStack Query + generated OpenAPI client, Zustand, react-hook-form +
  zod. Pure SPA served as Cloudflare Workers static assets.
- **Marketing**: an Astro site under `marketing/`, rendered to static
  HTML at build time (`output: 'static'`) and served as Cloudflare
  Workers static assets on the apex domain. Tailwind with the site's
  own theme in `marketing/src/styles/global.css`; it shares the body
  typeface and radius scale with the app, not its palette. Every page
  declares a title and description, and the
  shared layout derives the canonical URL, Open Graph and Twitter tags
  from them; the sitemap and `robots.txt` are generated from the site
  origin. No page ships client-side JavaScript unless it needs
  interactivity, and then as a scoped `<script>` in the component that
  needs it, or a React island (`client:*`) when the component holds
  state. Scripts are bundled files, never inline, so the CSP stays at
  `script-src 'self'`; any origin the site fetches from is stamped into
  `connect-src` at build time. The web app's origin, the site's own
  origin, and any endpoint the site posts to are build-time environment
  variables, validated so a malformed value fails the build. A route may
  render on the server only when its HTML must differ per request, by
  adding the Cloudflare adapter and opting that route out of
  prerendering; the rest stays static.
- Sentinel errors use standard Go `ErrXxx` naming.

## Authentication

WorkOS (AuthKit) is the identity provider for account holders (senders).
The Go API is the OAuth client, following the **token-mediating backend**
variant of the Backend for Frontend (BFF) pattern from the IETF "OAuth 2.0
for Browser-Based Apps" best current practice. The browser talks to WorkOS
only for the hosted login page; every token exchange happens on the API.

1. The web app presents its own sign-in, sign-up, email verification, and
   password reset screens. Social sign-in starts at
   `GET /v1/auth/login?provider=`, which sends the browser straight to the
   chosen provider's consent screen with no intermediate page, and
   returns through the API's `/v1/auth/callback`.
2. The API exchanges credentials, verification codes, and password reset
   tokens with WorkOS server-side through six JSON endpoints under
   `/v1/auth` (`sign-in`, `sign-up`, `verify-email`,
   `resend-verification`, `forgot-password`, `reset-password`) and seals
   the same encrypted, HttpOnly, Secure, SameSite=Strict cookie the
   callback sets, scoped to the API host and to the `/v1/auth` path.
3. The web app holds the access token **in memory only**. On load and
   shortly before expiry it calls `POST /v1/auth/refresh` with
   credentials. The API reads the cookie, obtains a new access token from
   WorkOS through its server API, rotates the refresh token and cookie,
   and returns the access token as JSON. A failed refresh clears the
   cookie and the app routes to sign-in.
4. Every other API request carries the access token in the
   `Authorization: Bearer` header. Auth middleware validates it against
   the WorkOS JWKS endpoint and resolves the user with no database lookup
   on the hot path. Cookies are ignored outside `/v1/auth`.
5. Sign-out is `POST /v1/auth/logout`: the API ends the WorkOS session
   and expires the cookie.
6. The cookie endpoints verify the `Origin` header against the allowed
   web origins, allow credentialed CORS for those origins only, and are
   rate limited more strictly than other routes.
7. Refresh tokens never reach browser JavaScript, `localStorage`, or
   logs. The WorkOS browser SDK and its `devMode` are not used. No WorkOS
   custom domain is required for silent refresh to work. WorkOS's hosted
   pages are not used.
8. User records sync via signature-verified, idempotent WorkOS webhooks.
9. Brute-force and bot protection on `/v1/auth/*` is a Cloudflare
   rate-limiting rule in front of the API plus the API's own per-address
   limiter. Refusals never reveal whether an address is registered.
10. The API sends the password-reset and account-exists emails through
    the project's transactional mail provider. WorkOS sends verification
    codes.

Signers do not need an account:

1. Each signer receives a unique, expiring, single-purpose link by
   email. Opening the link proves control of that inbox, and the
   recorded signer identity is the address the link was sent to.
2. The link token authenticates signing-page API calls through a header.
   The backend stores only a hash of the token and revokes it once the
   signature is recorded or the agreement is voided.
3. The signature record captures the evidence listed in Principle III.

## Development Workflow

- Features follow the Spec Kit lifecycle: `/speckit-specify` →
  `/speckit-clarify` → `/speckit-plan` → `/speckit-tasks` →
  `/speckit-taskstoissues` → `/speckit-implement`, with the OpenAPI
  contract change as part of planning.
- **GitHub issues are the task source**: after task generation,
  `/speckit-taskstoissues` turns the feature's tasks into
  dependency-ordered issues, and from that point the issues — not
  tasks.md — are the authoritative record of what remains and what is
  done. Every unit of work MUST have a GitHub issue before work on it
  begins: feature tasks above all, and also bug fixes, decisions, and
  work deferred from a feature's task list, each getting its issue at
  the moment it is identified.
- **Agent outputs land on the issues**: an agent working a task writes
  its work products to that task's issue — progress updates, findings,
  completion notes, and blockers as issue comments; scope changes as
  edits to the issue body. Chat is the conversation with the user; the
  issue is the durable record other sessions and contributors read.
- **Approval comes from the user, in the session or on the issue**:
  writing to an issue is not approval. The user approves work products
  either in the working session or with a comment from their GitHub
  account on the issue saying "approved" (optionally with conditions).
  Agent-authored comments never count as approval. Only approved work
  closes an issue. Commits and squash merges name the issues they
  complete (`Closes #N`) so merging closes them. A completed task is
  checked off in tasks.md at completion time. Its issue stays open
  until the work merges.
- **Merged work closes its issues**: after an approved merge, the
  agent MUST confirm that every issue the merge completes is closed.
  Any issue the platform did not close automatically is closed by
  hand with a completion comment naming the merge.
- **Branch naming**: feature work on `feature/NNN-kebab-slug` (Spec Kit
  number + descriptive slug, e.g. `feature/001-create-agreement`);
  bug fixes on `fix/kebab-slug`. Kebab-case after the prefix; names
  clearly describe the change.
- Every PR passes the full CI gate set for the paths it touches (build,
  lint with the mandated rules, tests with coverage gate, vulnerability
  scan, contract/codegen diffs, migration smoke) — all blocking. The
  default branch is always releasable.
- Reviews verify constitution compliance: spec coverage (I), test
  meaningfulness (II), security posture (III), justified complexity (IV).
- **Subagent model and effort match the task**: every subagent an
  agent spawns — workflow agents, explorers, implementers, reviewers,
  verifiers — is launched with its model and its reasoning effort set
  explicitly, chosen by the effort the task requires. Mechanical work
  (scaffolding, formatting, mechanical edits, verification sweeps)
  runs on a cheaper tier such as Sonnet or Haiku at low or medium
  effort. Demanding work (complex implementation, tricky queries,
  adversarial review) runs on Claude Opus at the effort it needs.
  Opus is the ceiling for any fan-out. The orchestrating session's
  model and effort MUST NOT cascade into a fan-out. Running any
  subagent on a model above Opus requires the user asking for it in
  the session. The orchestrating session delegates implementation: it
  runs the repository's implement workflow
  (`.claude/workflows/speckit-parallel-implement.js`) or spawns
  implementers, marks progress, and owns the final gate. It does not
  write feature code itself.
- **Tests are targeted during iteration**: while iterating, an agent
  runs only the tests that cover the code it is changing — single
  files, single packages, or single test names. An implementer hands
  off with a scoped confirmation run. The full test suites, lint,
  typecheck, and format checks run once, at the end of the run, as a
  single gate owned by the orchestrating session. Review agents read
  code and MUST NOT execute test suites.
- **Frontend changes are looked at before they are reported done**: a
  user-visible web change is complete only after the agent has opened
  the running app in a browser, exercised the changed surface, and
  confirmed with its own eyes what the page renders — screenshots or
  live page reads of the real app against the real backend. Passing
  test suites, type checks, lint, and MSW-boundary integration tests
  MUST NOT substitute for looking at the page. When authentication
  gates the surface, the agent asks the user to sign in to the shared
  preview browser once and then drives that session. A change the
  agent could not look at is reported as unverified, in those words —
  never as done.
- **Asking the user questions**: when an agent needs a decision from the
  user — spec clarifications, choosing between options, scope calls — it
  MUST ask via the AskUserQuestion tool (structured options with an
  escape hatch for custom answers) whenever that tool is available in
  the running environment. Free-text question lists in chat are the
  fallback only where the tool does not exist. One question per
  decision; each option states its implications.

## Binding Convention Documents

The following documents are incorporated by reference and carry the force
of this constitution; conflicts resolve in favor of the constitution:

- `docs/backend-conventions.md` — backend layout, DI, HTTP/data layers,
  logging and error-log contract, complexity and style, observability,
  testing and CI gates, outbound resilience, API contract, auth.
- `docs/frontend-conventions.md` — frontend structure, routing and URL
  state, data and state hierarchy, forms, components, TypeScript rules,
  errors and debuggability, auth, security, testing and CI gates.
- `docs/monorepo-and-ci.md` — repository layout, path-filtered CI,
  required-checks handling, branch strategy.

`docs/deployment.md` and `docs/project-context.md` are informative
references: they record where things run and what the product is, and
they are kept current, but they do not add rules.

Amendments to these documents follow the same procedure as constitution
amendments.

## Governance

This constitution supersedes all other development practices in this
repository. Where guidance elsewhere conflicts with it, the constitution
wins.

- **Amendments**: proposed as a change to this file (or a binding
  convention document), stating motivation, version bump, and any
  migration for in-flight specs or code. Amendments take effect when
  merged.
- **Versioning**: semantic versioning. MAJOR for removals or redefinitions
  of principles that invalidate existing practice; MINOR for new
  principles or materially expanded guidance; PATCH for clarifications.
  Every amendment updates the version line and the Sync Impact Report.
- **Compliance review**: all PRs MUST verify compliance with the Core
  Principles. Deviations MUST be justified in writing in the plan's
  Complexity Tracking (or equivalent); unjustified deviations block merge.

**Version**: 1.3.1 | **Ratified**: 2026-09-03 | **Last Amended**: 2026-09-04
