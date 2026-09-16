# Frontend Conventions (SoW web)

Drafted 2026-08-27 for the PixelsTwo marketplace and carried over to SoW
on 2026-09-03. Rules derive from the project constitution (TDD, integration-first, security-paramount, human-readable
without agents) and the decisions resolved with the user the same day.
Stack: **React + Vite + TypeScript, shadcn/ui + Tailwind, react-router-dom,
TanStack Query, Zustand, react-hook-form + zod, Vitest/Testing Library/MSW +
Playwright, ESLint + Prettier.** Pure SPA on static hosting; the Go backend
is the only server. The frontend lives at `web/` in the monorepo (see
`docs/monorepo-and-ci.md`).

---

## 1. Project Layout: feature modules

- Code is organized by **feature, not by layer**:

  ```
  src/
    app/                  # app shell: router, providers, error boundaries
    features/
      agreements/
        components/       # feature-private components
        hooks/            # feature-private hooks
        api.ts            # feature's query/mutation hooks (wraps generated client)
        types.ts          # feature-private types (API types come generated)
        index.ts          # the feature's public surface
    components/ui/        # shadcn primitives (generated, then owned)
    components/           # shared non-shadcn components (used by 2+ features)
    lib/                  # cross-cutting: api client setup, auth, utils
    stores/               # Zustand stores (thin — see §4)
    test/                 # test setup, MSW handlers, factories
  ```

- A feature imports from another feature **only via its `index.ts`** public
  surface; deep imports across features are an ESLint error
  (`eslint-plugin-import` rules). Shared code lives in `components/` or
  `lib/`, promoted there only when a second consumer appears.
- Everything for a feature is greppable in one folder — the frontend
  mirror of the backend's entity-per-file rule.

## 2. Routing & URL State (react-router-dom)

- `react-router-dom` in browser (SPA) mode. Routes are declared in one
  place (`app/router.tsx`) so the app's page inventory is readable in a
  single file — the frontend equivalent of the backend's route files.
- **The URL is the primary state container for navigation state.**
  Anything a user would expect to survive refresh, back-button, or a
  shared link MUST live in the URL: active filters, search terms, sort
  order, pagination, selected tab, opened detail panel. Rule of thumb: if
  losing it on F5 would annoy the user, it belongs in the URL.
- Search params are read/written through a typed helper per route (zod
  schema validating `useSearchParams`) — no raw string plucking scattered
  through components, and invalid params degrade to defaults, never crash.
- Modals/panels that represent a resource (`/agreements/123` as a route, not
  `isOpen` state) get routes; purely ephemeral UI (dropdown open, hover)
  stays in component state.
- Route-level code splitting via `lazy()` per feature is the default.

## 3. Data Layer: generated client + TanStack Query

- **The API client and all request/response types are generated from
  `contracts/openapi.yaml`** at the monorepo root (orval or openapi-ts).
  Hand-writing an API
  type or fetch call for an endpoint the spec covers is banned — the
  spec-first backend makes drift impossible to justify. Regeneration runs
  in CI and diffs (same gate pattern as the backend).
- **TanStack Query owns all server state.** Rules:
  - No `useEffect` + `fetch` data loading, ever. No copying server data
    into `useState`/Zustand — the query cache IS the client-side copy.
  - Query keys are declared per feature in a typed key factory
    (`agreementKeys.list(filters)`, `agreementKeys.detail(id)`) — never inline
    string arrays, so invalidation is greppable and typo-proof.
  - Mutations invalidate or update the affected keys in `onSuccess`;
    optimistic updates only where UX demands it, always with rollback.
  - Feature `api.ts` files export ready-made hooks (`useAgreements(filters)`,
    `useCreateAgreement()`) wrapping the generated client — components never
    call the client directly.
- Global defaults set once in `app/`: sensible `staleTime`, retry policy
  (never retry 4xx; limited retry on 5xx/network), and errors surfacing to
  the nearest error boundary rather than being swallowed per-call.
- **All API timestamps are Unix milliseconds** (project-wide rule).
  Formatting for display goes through one date util module (`lib/dates.ts`,
  date-fns underneath) — no ad-hoc `new Date(x).toLocaleString()` scattered
  in components.

## 4. Client State: URL → Query → Zustand → local

State has a strict hierarchy; each piece lives in the *first* tier that
fits, and putting it lower is a review defect:

1. **URL** — navigation state (§2).
2. **TanStack Query cache** — everything that came from the server.
3. **Zustand** — the thin remainder of genuinely global client state:
   auth/session status, cross-page UI (sidebar collapsed, theme), in-flight
   multi-step flows (draft uploads). Stores are small, per-domain
   (`stores/session.ts`), actions colocated with state, devtools
   middleware on in dev.
4. **Component state** — ephemeral UI local to one component.

If a Zustand store starts caching server data or mirroring the URL, it's
in the wrong tier.

## 5. Forms: react-hook-form + zod

- All non-trivial forms use `react-hook-form` with `zodResolver`, through
  shadcn's `Form` components (which assume exactly this combo).
- The zod schema is the single source of client-side validation for a
  form, colocated with it. Where the OpenAPI spec defines the constraint,
  the zod schema mirrors it (generated zod schemas from the spec where the
  generator supports it) — client validation is UX; **the server remains
  the authority** and 400 responses with field details map back onto form
  errors via a shared helper.
- Submissions go through TanStack Query mutations; buttons disable while
  pending; server field errors render inline, never only as a toast.

## 6. Components & Styling (shadcn + Tailwind)

- shadcn components are generated into `components/ui/` and **owned** —
  edits allowed, but recorded (the component stops tracking upstream).
  App-specific composites build on top in `components/` or feature
  folders; primitives in `ui/` stay app-agnostic.
- Tailwind utilities are the styling mechanism; `cn()` for conditional
  classes. No CSS-in-JS, no separate CSS files beyond the Tailwind entry
  and genuine keyframe/global needs. Design tokens (colors, radii) live in
  the Tailwind theme / CSS variables — no hex literals in class strings.
- Components are function components with typed props; one exported
  component per file for anything non-trivial; files kebab-case matching
  the backend convention (`agreement-card.tsx`).
- Accessibility is not optional: interactive elements are real
  `<button>`/`<a>`, forms have labels, `eslint-plugin-jsx-a11y` runs as
  errors. shadcn/Radix gives keyboard/ARIA behavior for free — don't
  rebuild primitives by hand and lose it.

## 7. TypeScript: strict, enforced

- `strict: true` plus `noUncheckedIndexedAccess`. `tsc --noEmit` is a CI
  gate.
- `any` is banned (`@typescript-eslint/no-explicit-any`: error). Type
  assertions (`as`) require a comment justifying why the compiler is
  wrong. `zod.parse` at runtime boundaries (URL params, localStorage,
  anything not covered by the generated client) instead of trusting casts.
- Generated API types are the source of truth for server shapes — never
  re-declare them locally.

## 8. Errors & Debuggability

The frontend mirror of the backend's error-log contract: a production bug
report must be traceable to a specific request, user, and release.

- **Error boundaries per route segment** (react-router `errorElement`) so
  a crash degrades one page, not the app; a root boundary catches the
  rest. Boundaries show a recovery UI (retry/reload), never a blank
  screen.
- **Sentry browser SDK wired from day 1, enabled only in deployed
  environments** (DSN set per environment; local dev reports nothing).
  Mandatory from the first environment real users touch. Wired to the
  router for release + route context. Unexpected errors and error-boundary
  catches report to Sentry; expected/validation errors do not (same signal
  discipline as the backend). **Session Replay stays off** — no third-party
  recordings of users viewing confidential agreements. Maintain the
  ignore-list for browser-extension noise.
- **Distributed tracing joins the backend**: the API client attaches
  W3C `traceparent` (Sentry's browser tracing or OTel web SDK) so a slow
  or failing user action links to the exact backend trace and its logs —
  the browser is where the §5-backend correlation chain begins.
- The generated client normalizes API errors into one typed error object
  (status, code, field details, request id); components branch on it,
  never on raw `fetch` responses. The backend's `request_id` from error
  responses is displayed in the error UI ("reference: abc123") so a user
  report carries the correlation key.
- Dev debuggability: TanStack Query devtools and Zustand devtools enabled
  in dev builds; React StrictMode on.

## 9. Auth (WorkOS through the API)

- The app never calls WorkOS. Sign-in and sign-up are first-party forms
  that post JSON to `/v1/auth/sign-in` and `/v1/auth/sign-up` (then
  `/v1/auth/verify-email`, `/v1/auth/resend-verification`) with
  `credentials: 'include'`. The one top-level navigation to WorkOS is
  "Continue with Google", a plain link to
  `/v1/auth/login?provider=google&returnTo=...`; the return leg lands on
  the API's callback, which redirects to the app's `/callback` route. The
  app holds no WorkOS client id or secret and never renders the hosted
  AuthKit page.
- **`adoptSession` is the only way a mutation starts a session.**
  `lib/auth/session.ts` exports `adoptSession(token: AccessToken)`, which
  stores the token, schedules the pre-expiry refresh, and clears the
  signed-out sentinel. Every mutation that receives a `200 AccessToken`
  (sign-in, verify-email, reset-password) calls it directly instead of
  navigating to `/callback` or issuing an extra refresh.
- **Access tokens live in JS memory only** (session store) — never
  `localStorage`, never non-httpOnly cookies. On app load and shortly
  before expiry the session store calls `POST /v1/auth/refresh` with
  `credentials: 'include'`; the API answers with a fresh access token
  minted from the HttpOnly refresh cookie it owns. A failed refresh routes
  to sign-in. (Closing the tab does NOT log the user out: the refresh
  cookie lives on the API host and survives, and the app-load refresh
  mints a fresh access token from it — users re-authenticate only when
  that cookie expires or the session is revoked.)
- **The pending verification token and the password-reset token live in
  component state only** — never the URL (so they never land in history
  or logs) and never storage. The reset page reads `?token=` once on
  mount and strips it with `history.replaceState` before rendering the
  form.
- Nine calls send cookies: the callback (a browser navigation), refresh,
  logout, and the six credential endpoints (sign-in, sign-up,
  verify-email, resend-verification, forgot-password, reset-password).
  Every other API call is a plain bearer request with no credentials, so
  CSRF exposure is confined to the auth endpoints.
- The generated client injects `Authorization: Bearer <token>` from the
  session store and handles a single in-flight refresh on 401 (queueing
  concurrent requests, then retrying once).
- `@workos-inc/authkit-react` is not used. Its browser-side refresh relies
  on a cookie on the WorkOS host, which browsers treat as third-party from
  the app origin unless a paid WorkOS custom domain is configured.
- Route guards: authenticated app routes live under a layout route that
  requires a session and preserves the intended destination
  (`?redirect=`); guard logic exists in exactly one place. Magic-link
  signing routes are public routes outside that layout (§10).
- No secrets in the frontend: `VITE_`-prefixed env vars are public by
  definition — anything sensitive stays behind the API.

## 10. Security

- Agreement documents never embed via raw URLs to storage: PDFs render
  and download through short-lived presigned URLs obtained from the API,
  requested per view — matching the backend rule that documents never
  flow through the API itself.
- **Uploads are direct-to-storage, never through the API** (the backend's
  2MB body limit makes this structural, not optional): the client requests
  a presigned upload from the API, uploads directly to storage with
  progress, then confirms completion to the API, which validates (size,
  type, checksum) before the object becomes usable. Upload state
  (in-flight, progress) is exactly the kind of multi-step flow state
  Zustand exists for (§4).
- **Signing pages are public routes authenticated by the magic-link
  token**: the signing route lives outside the authenticated layout, reads
  its token from the URL once on load, keeps it in memory for the visit,
  and never writes it to `localStorage`, cookies, or analytics. The page
  renders only what the token grants and sends the token to the API in a
  header, never as a query parameter on API calls.
- CSP delivered via hosting headers (the Workers static-assets config is
  part of the repo): `default-src 'self'`, explicit allowlist for the API
  origin, WorkOS, Sentry, and the R2 document origin; no `unsafe-inline`
  scripts.
- All user-generated content (agreement fields, party names, notes)
  renders as text by default; any rich-text rendering requires
  sanitization (DOMPurify) and a review flag. `dangerouslySetInnerHTML` is
  banned by lint except in that sanitized path.
- Dependency hygiene: `npm audit` (or socket/dependabot) in CI; lockfile
  committed; no postinstall-script packages without review.

## 11. Testing & CI Gates

Integration-first, mirroring the backend (constitution: integration >
unit, TDD, 100% meaningful aim / 85% new-code hard gate).

- **Primary suite: integration-style component tests** — Vitest + Testing
  Library rendering a whole feature (router + providers included), with
  the network mocked at the **HTTP boundary via MSW** using handlers that
  conform to the OpenAPI spec. Module-mocking application code
  (`vi.mock` of stores/hooks/components) is banned in these tests — if
  the test can't tell the real code ran, it isn't an integration test.
  Tests interact like a user (`userEvent`, role/label queries — no
  test-ids where a role works) and assert visible outcomes.
- **Unit tests** for logic-dense pure code (schemas, helpers, reducers).
- **Playwright E2E** for the critical flows — sign-up/login, creating
  and sending an agreement, and the magic-link signing flow — run against a real
  backend (compose/preview env) on PR merge; smoke subset on every PR.
- MSW handlers and data factories live in `test/`, shared across tests
  and (optionally) dev mode.
- **CI gates, all blocking**: `tsc --noEmit` · ESLint (a11y, import
  boundaries, no-explicit-any, hooks rules) · Prettier check · Vitest with
  85% new-code coverage (generated client and `components/ui` excluded
  from denominator) · OpenAPI client regen diff · `vite build` ·
  Playwright smoke.

## 12. Tooling

- **ESLint + Prettier** (flat config): `typescript-eslint`,
  `eslint-plugin-react-hooks`, `eslint-plugin-jsx-a11y`,
  `@tanstack/eslint-plugin-query`, import-boundary rules (§1). Lint rules
  are errors, not warnings — warnings rot.
- Vite for dev/build; Vitest shares its config. Path alias `@/` → `src/`.
- Node version pinned (`.nvmrc`); package manager pinned via
  `packageManager` field; lockfile committed.
- Pure SPA deploys as static assets on Cloudflare Workers (`wrangler.jsonc`
  in `web/` with `not_found_handling: "single-page-application"`); the
  hosting/CSP headers configuration is versioned in the repo. SEO is not a
  requirement of the app: the public landing site lives in `marketing/`
  as a separate Astro site with static output (see the constitution's
  Project Structure & Stack and `marketing/README.md`).
