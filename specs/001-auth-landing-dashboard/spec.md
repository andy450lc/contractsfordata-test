# Feature Specification: Sign-Up, Sign-In, Landing Page, and Dashboard

**Feature Branch**: `feature/001-auth-landing-dashboard`

**Created**: 2026-09-03

**Status**: Draft

**Input**: User description: "create the login/sign up flows, simple landing
page, and a "you've logged in" dashboard. encouraged to copy from marketplace
instead writing full thing from scratch, but up to you. we'll have different
accounts from those in marketplace btw, not pixelstwo."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - New visitor signs up and lands on the dashboard (Priority: P1)

A visitor with no account opens the product's landing page, reads what the
product does, and chooses to get started. They are taken to the product's
hosted authentication experience, where they register with an email address
and password. After completing registration (including any email
verification the authentication service requires), they are returned to
the application already signed in and land on a dashboard that greets them
by name and confirms they are logged in.

**Why this priority**: Account creation is the gateway to everything else.
Nothing an account holder does (creating or sending an agreement) can exist
before this loop works: landing page → hosted authentication → back to the
app → recognized user with a record on the platform.

**Independent Test**: Start on the landing page, complete a brand-new
registration with a fresh email address, and verify the browser returns to
the application on the dashboard in a signed-in state showing that
person's name, and that the platform now holds a user record for them.

**Acceptance Scenarios**:

1. **Given** a visitor with no session, **When** they open the product's
   public address, **Then** they see a landing page with the product name,
   a one-paragraph description of what it does, and clearly visible
   actions to sign in and to get started.
2. **Given** the landing page, **When** the visitor chooses to get started,
   **Then** they are taken to the hosted registration experience offering
   email/password registration.
3. **Given** the hosted registration form, **When** the visitor completes it
   successfully, **Then** they are returned to the application in a
   signed-in state and see the dashboard greeting them by name.
4. **Given** a completed registration, **When** the platform processes the
   new identity, **Then** a user record exists with the identity
   provider's user identifier, the user's name, the user's email address,
   and creation and update timestamps.
5. **Given** a visitor who abandons or fails the hosted flow (they cancel,
   or the authentication service reports an error), **When** they are
   returned to the application, **Then** they see the sign-in page with a
   clear, non-technical notice. They never see a broken page or a silent
   failure.

---

### User Story 2 - Returning user signs in and stays signed in (Priority: P2)

A person who already has an account opens the application, signs in with
their email and password through the hosted experience, and lands on the
dashboard. If they reload the page, close the tab and reopen it later, or
leave the app open past the point where their short-lived credential
expires, they remain signed in without re-entering credentials, on every
mainstream browser and in private browsing modes.

**Why this priority**: Sign-in is the everyday path for every existing
user. Silent session renewal that works everywhere is the specific
behavior the marketplace product struggled with, and the reason the
constitution's authentication section was rewritten before this feature.

**Independent Test**: Sign in with a known-good account and verify arrival
at the dashboard. Reload the page, then close the tab and reopen the
application, then wait past the short-lived credential's lifetime while
the app is open. Verify no credential prompt appears in any of the three
cases, in Chrome incognito, Safari, and Firefox.

**Acceptance Scenarios**:

1. **Given** an existing account, **When** the user completes the hosted
   sign-in with correct credentials, **Then** they land on the dashboard
   in a signed-in state.
2. **Given** an incorrect password, **When** the user attempts to sign in,
   **Then** the hosted experience shows the error and the user can retry.
   The application never shows a success state.
3. **Given** a signed-in user, **When** they reload the application,
   **Then** they remain signed in and reach the dashboard without
   re-entering credentials.
4. **Given** a signed-in user, **When** they close the tab and reopen the
   application within the session's lifetime, **Then** they remain signed
   in and reach the dashboard without re-entering credentials.
5. **Given** a signed-in user with the app open, **When** their
   short-lived credential expires, **Then** it is renewed quietly and
   their next action succeeds with no redirect and no prompt.
6. **Given** any of scenarios 3 to 5, **When** performed in Chrome
   incognito, Safari, or Firefox with default privacy settings, **Then**
   the outcome is the same.
7. **Given** a signed-in user whose session has fully expired or been
   revoked, **When** they next interact with the application, **Then**
   they are sent to sign in rather than seeing errors or stale data.

---

### User Story 3 - Signed-in state is enforced everywhere (Priority: P3)

The dashboard is reachable only by authenticated users, and every
server-side request is accepted only with proof of identity. A signed-in
user can sign out and is returned to the sign-in page with their session
fully ended.

**Why this priority**: Enforcement is what makes sign-in meaningful. It is
testable on its own but depends on Stories 1 and 2 existing to have
sessions to enforce.

**Independent Test**: Request the dashboard address with no session and
verify redirection to sign-in. Call a protected server endpoint without
credentials and verify refusal. Sign out and verify the old session no
longer works anywhere.

**Acceptance Scenarios**:

1. **Given** no session, **When** someone navigates directly to the
   dashboard address, **Then** they are redirected to the sign-in page,
   and after signing in they arrive at the address they first requested.
2. **Given** no valid credential, **When** any protected server endpoint
   is called, **Then** the request is refused as unauthenticated and no
   data is returned.
3. **Given** a signed-in user, **When** they sign out, **Then** they are
   returned to the sign-in page, and neither the application nor the
   server accepts their old session afterwards.
4. **Given** a signed-in user, **When** the server handles their request,
   **Then** the server can identify which user record the request belongs
   to.
5. **Given** a session-renewal request arriving from an origin that is not
   the application, **When** it reaches the server, **Then** it is
   refused and no credential is issued.

---

### User Story 4 - Operate a runnable, probeable foundation (Priority: P4)

An operator, or the hosting platform acting for them, starts the server
and needs to know at any moment whether the process is alive and whether
it can serve real traffic. A developer picking up the next feature finds
every structural part assembled: configuration that fails fast when
incomplete, a place for each kind of code, database migrations, request
validation, a central error path, and the versioned request surface. A
production error can be traced from a single request identifier to every
log line it produced.

**Why this priority**: This is the first feature in the repository, so it
also carries the foundation the marketplace built in its first two
features. It is P4 because a user never sees it directly, not because it
is optional. The constitution requires it before the first feature ships.

**Independent Test**: Start the server with valid configuration and a
reachable database and probe both health endpoints. Take the database
away and observe readiness fail while liveness succeeds. Start with a
required setting blank and observe refusal to start naming that setting.
Trigger an error and verify exactly one error log line carrying the
request and trace identifiers.

**Acceptance Scenarios**:

1. **Given** the server is running with all dependencies reachable,
   **When** the liveness probe is called, **Then** it succeeds without
   consulting any dependency, **and When** the readiness probe is called,
   **Then** it succeeds after verifying the database is usable.
2. **Given** the database becomes unreachable, **When** the readiness
   probe is called, **Then** it fails while liveness still succeeds.
3. **Given** required configuration is missing or blank in either
   application, **When** it starts, **Then** it refuses to start with a
   message naming the missing setting.
4. **Given** a fresh empty database, **When** migrations are applied,
   **Then** the schema reaches the current version, **and When** applied
   again, **Then** nothing changes.
5. **Given** a request that fails unexpectedly, **When** the failure is
   handled, **Then** the client receives a generic error carrying a
   reference identifier and no internal details, and exactly one error
   log line is written carrying the request and trace identifiers.
6. **Given** a change is pushed, **When** the automated checks run,
   **Then** every constitution-mandated gate runs for the paths touched
   and blocks on failure.

---

### Edge Cases

- The identity provider notifies the platform about the same user more
  than once, or out of order: the user record ends up correct, with no
  duplicates.
- A user authenticates successfully before their user record exists on
  the platform (notification lag): the user can still use the
  application, and the record catches up without user-visible errors.
- The identity provider updates a user's name or email after
  registration: the platform's record reflects the change and its update
  timestamp moves.
- The authentication service is unreachable when a visitor tries to sign
  in or when a session renews: the application shows a clear failure
  notice, keeps any still-valid session, and recovers when the service
  returns.
- A forged or tampered notification about user changes arrives: it is
  rejected and no data changes.
- The return leg of sign-in arrives with a missing, expired, or tampered
  state value: the sign-in is rejected with a notice, and no session is
  created.
- Two tabs of the application renew the session at the same moment: both
  end up signed in, and neither is logged out by the other's renewal.
- A session credential issued for one deployment stage is presented to
  another: it is refused.
- A visitor opens the landing page while already signed in: the actions
  still work and lead to the dashboard without a second sign-in.
- A visitor navigates to an address that does not exist inside the
  application: a simple "page not found" view is shown with a way back.
- The landing page and the application are viewed on a mobile-width
  screen: both remain fully usable without horizontal scrolling.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The product MUST serve a public landing page at its root
  address showing the product name, a short description of what the
  product does, and actions to sign in and to get started. The landing
  page MUST load without any session and without any server-side call.
- **FR-002**: Visitors MUST be able to create an account and sign in using
  an email address and password through the product's hosted
  authentication experience. Email/password is the only method offered in
  this feature.
- **FR-003**: Accounts MUST belong to this product alone. A person with an
  account on the marketplace product does not have an account here, and
  registering here creates nothing there.
- **FR-004**: After successful sign-up or sign-in, the user MUST land on a
  dashboard page. In this feature the dashboard shows the signed-in
  user's name and email, a statement that they are logged in, and a
  sign-out action, and nothing else.
- **FR-005**: The dashboard and all future authenticated pages MUST be
  unreachable without a valid session. Unauthenticated visitors are
  redirected to the sign-in page, and after signing in they arrive at the
  page they first requested.
- **FR-006**: Every protected server endpoint MUST refuse requests that
  lack valid proof of identity and MUST resolve a valid request to the
  platform's user record for that caller without a database lookup on
  the ordinary request path.
- **FR-007**: The platform MUST keep its own user record per registered
  user: the identity provider's user identifier as the primary key
  (opaque text), the user's name, the user's email address, a creation
  timestamp, and a last-updated timestamp. The email address is kept for
  the lifetime of the account because agreements will name parties by
  email. It is deleted when the account is deleted.
- **FR-008**: User records MUST stay in sync with the identity provider.
  Creation and later profile changes reach the platform through provider
  notifications that are verified for authenticity and safe to receive
  more than once or out of order.
- **FR-009**: Sessions MUST be owned by the platform's server, following
  the constitution's authentication section: the server performs the
  sign-in exchange with the identity provider, keeps the long-lived
  renewal credential in a protected cookie that scripts cannot read, and
  issues short-lived credentials to the application on request. The
  application holds short-lived credentials in memory only.
- **FR-010**: Sessions MUST survive a page reload and a closed-and-reopened
  tab while still valid, and short-lived credentials MUST renew quietly
  before they expire while the application is open. This MUST hold in
  Chrome, Safari, and Firefox at default privacy settings and in private
  browsing modes, with no dependency on a custom authentication domain.
- **FR-011**: A signed-in user MUST be able to sign out from the
  application. Afterwards their prior session is not accepted anywhere.
- **FR-012**: The session endpoints (sign-in start, return leg, renewal,
  sign-out) MUST accept requests only from the application's own origins
  and MUST be rate limited more strictly than other endpoints.
- **FR-013**: Authentication failures (cancelled flow, provider error,
  invalid return, expired session) MUST surface as clear, non-technical
  notices on the sign-in page, never as broken pages, silent failures, or
  fake success states.
- **FR-014**: Both applications MUST read every setting needed to connect
  to the identity provider, the database, and each other from
  environment configuration, documented in each application's
  environment template and never committed as real values. An
  environment missing a required setting fails at startup with a message
  naming the setting.
- **FR-015**: The server MUST expose a liveness probe that succeeds
  whenever the process is up and a readiness probe that succeeds only
  when the database is usable, MUST apply schema changes through a
  repeatable migration mechanism, and MUST shut down gracefully, letting
  in-flight requests finish within a bounded grace period.
- **FR-016**: Every request MUST be traceable from a single request
  identifier to every log line it produced, with request and trace
  identifiers attached by construction. Unexpected failures MUST be
  logged exactly once and returned to the client as a generic error
  carrying a reference identifier and no internal details.
- **FR-017**: Every change MUST pass the constitution's automated gates
  for the paths it touches before it can merge.

### Key Entities

- **User**: A person with an account on the platform. Identified by the
  identity provider's opaque text identifier. Carries name, email
  address, and creation and last-updated timestamps. Credentials are held
  by the identity provider. Organisations, teams, and roles are future
  features.
- **Session** *(conceptual)*: The signed-in state connecting a browser to
  a user. Consists of a long-lived renewal credential held by the
  platform's server on the browser's behalf and a short-lived credential
  the application presents on each request. Validated on every protected
  request. Not stored as platform data beyond the cookie.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new visitor can go from the landing page through
  registration to the dashboard in under 3 minutes, including email
  verification.
- **SC-002**: A returning user can sign in and reach the dashboard in
  under 30 seconds.
- **SC-003**: The landing page becomes readable within 2 seconds on a
  typical broadband connection.
- **SC-004**: A signed-in user who reloads, closes and reopens the tab, or
  waits past the short-lived credential's lifetime remains signed in
  100% of the time while the session is valid, with no credential
  prompt, across Chrome incognito, Safari, and Firefox.
- **SC-005**: 100% of requests to protected server endpoints without a
  valid credential are refused, and no protected data is returned to
  them.
- **SC-006**: After registration, the platform's user record for the new
  user exists within 60 seconds, and repeated delivery of the same
  provider notification produces exactly one record.
- **SC-007**: After sign-out, 0 subsequent requests using the old session
  succeed.
- **SC-008**: An operator can go from a single request identifier in an
  error report to every log line that request produced in one search.
- **SC-009**: Silent session renewal works with no paid custom
  authentication domain on any deployment stage.

## Assumptions

- The hosted authentication experience handles the registration and
  sign-in forms, password rules, and email verification. The platform
  builds no credential forms of its own.
- Email verification is required before a new account's first completed
  sign-in, using the authentication service's standard behaviour.
- Additional sign-in methods (Google, single sign-on) are a configuration
  change in the identity provider's dashboard and are not offered in this
  feature. Offering one later needs no new specification unless it
  changes what the platform stores.
- The product name is the placeholder "SoW" on the landing page and in
  the application until a real name is chosen. Renaming is a text change.
- The landing page is a single static page with no forms, analytics, or
  content beyond the description and the two actions. Pricing, blog, and
  documentation pages are out of scope.
- The dashboard is intentionally minimal. Agreement creation, lists, and
  team features are future specifications.
- The user's name and email are collected during hosted registration and
  delivered to the platform by the identity provider. If the name is
  absent, the record's name may be empty until the provider supplies one,
  and the dashboard greets by email instead.
- Account lifecycle operations beyond creation and profile updates
  (deletion, deactivation, password reset) follow the identity provider's
  defaults. Platform-side account deletion, including deletion of the
  stored email, is specified when agreement retention is defined.
- A maintainer supplies real values for the new environment settings
  (identity provider, database, cookie key, origins) before end-to-end
  verification. Work pauses for that hand-off after the environment
  templates land.
- Structure, conventions, and tooling are copied from the marketplace
  repository wherever they fit, adapted only where this project's
  constitution differs (session ownership, hosting, database provider).

## Dependencies

- An identity provider account and environment dedicated to this product,
  separate from the marketplace's, configured for email/password
  authentication and supplying the credentials and identifiers both
  applications need.
- A database instance and a deployment target for each application, per
  `docs/deployment.md`.
