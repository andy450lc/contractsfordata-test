# Feature Specification: First-Party Sign-In and Sign-Up Forms

**Feature Branch**: `feature/004-first-party-auth-forms`

**Created**: 2026-09-04

**Status**: Draft

**Input**: User description: "Feature 004: first-party sign-in and sign-up
forms (headless WorkOS). Replace the hosted AuthKit page with the app's own
sign-in and sign-up forms. The web app posts credentials to the SoW API; the
API talks to WorkOS User Management server-to-server (password
authentication, user creation, email verification codes, password reset) and
sets the same sealed session cookie it sets today. Refresh and logout are
unchanged. Keep two ways in: (1) email + password, with the WorkOS 6-digit
email verification code step on sign-up and on sign-in when the email is not
yet verified, plus forgot-password / reset-password; (2) "Continue with
Google", which sends the browser directly to Google via the WorkOS
authorization URL with the Google provider selected, never showing the
AuthKit chooser page, and returns through the existing API callback. The
browser never sees the AuthKit hosted UI for any flow. Brute-force and bot
protection is delegated to Cloudflare in front of the API (rate limiting
rules on the auth endpoints); the API keeps its existing per-IP auth rate
limit and returns generic error messages that do not reveal whether an email
is registered. No MFA, no magic link, no custom auth domain purchase. Update
the OpenAPI contract, generated client, and the constitution's auth section
(the constitution currently mandates the hosted login page)."

## Context

Feature 001 shipped sign-up and sign-in by handing the browser to the
identity provider's hosted login page, which lives on the provider's own
domain and carries the provider's branding. Removing that domain from the
address bar requires a paid custom-domain add-on. This feature removes the
hosted page instead: the product presents its own sign-in and sign-up
screens, and the platform completes every credential exchange with the
identity provider behind the scenes. The session model that Feature 001
established (sealed cookie, in-memory access token, silent refresh,
sign-out) is kept exactly as it is. Only the way a session begins changes.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Returning user signs in with email and password (Priority: P1)

An account holder opens the product's sign-in page, which is a page of the
product itself, not a page on another domain. They type their email
address and password and submit. They are signed in and land on the
dashboard (or on the page they were originally trying to reach). At no
point does the browser show the identity provider's hosted page or address.

**Why this priority**: Sign-in is the most frequent authentication action
and the one users see every time they return. It is the smallest slice that
proves the whole approach: the product's own form starts a session that
the existing refresh, guard, and sign-out machinery then carries.

**Independent Test**: With an account that already exists and has a
verified email, open the sign-in page, enter the email and password, submit,
and verify the browser lands on the dashboard signed in, the address bar
never left the product's origin, and a reload keeps the user signed in.

**Acceptance Scenarios**:

1. **Given** an account holder with a verified email, **When** they submit
   the correct email and password on the sign-in page, **Then** they are
   signed in and taken to the dashboard within the product, and the
   address bar shows only the product's own origin throughout.
2. **Given** an anonymous visitor who was redirected to sign-in from a
   protected page, **When** they sign in successfully, **Then** they land
   on the page they originally asked for.
3. **Given** a sign-in attempt with a wrong password, an unknown email, or
   a malformed email, **When** the form is submitted, **Then** the page
   shows one generic message ("That email and password don't match") that
   is identical for the wrong-password and unknown-email cases, keeps the
   typed email in the field, and lets the user try again.
4. **Given** an account holder whose email is not yet verified, **When**
   they submit the correct email and password, **Then** the page moves to
   a "check your inbox" step asking for the 6-digit code that has just
   been emailed, and entering the correct code completes sign-in.
5. **Given** a user who is already signed in, **When** they open the
   sign-in page, **Then** they are sent to the dashboard without seeing
   the form.
6. **Given** the identity provider is unreachable, **When** a user
   submits the form, **Then** the page shows the existing "temporarily
   unavailable" notice and the credentials are not lost from the fields.

---

### User Story 2 - New visitor signs up with email and password (Priority: P2)

A visitor with no account opens the product's sign-up page, enters their
first name, last name, email address, and a password, and submits. The
product tells them a 6-digit code has been emailed to them. They enter the
code, the account is confirmed, and they land on the dashboard signed in
and greeted by name.

**Why this priority**: Account creation is the gateway to everything else,
but it depends on the sign-in plumbing from Story 1 (a verified sign-up
ends by starting a session the same way).

**Independent Test**: Use a fresh email address, complete the sign-up form,
retrieve the code from the inbox, enter it, and verify the dashboard shows
the new user's name, the platform holds a user record for them, and the
address bar never left the product's origin.

**Acceptance Scenarios**:

1. **Given** a visitor with no account, **When** they submit a valid
   first name, last name, email, and password, **Then** the page moves to
   the code step and an email containing a 6-digit code arrives at that
   address.
2. **Given** the code step, **When** the visitor enters the correct code,
   **Then** they are signed in, land on the dashboard greeted by their
   first name, and a user record exists for them on the platform.
3. **Given** the code step, **When** the visitor enters a wrong or expired
   code, **Then** the page says the code didn't work, keeps them on the
   code step, and offers to send a new code.
4. **Given** the code step, **When** the visitor asks for a new code,
   **Then** a fresh code is emailed and the previous one no longer works.
5. **Given** a password the identity provider judges too weak, **When**
   the form is submitted, **Then** the page shows the provider's reason in
   plain language next to the password field and nothing else changes.
6. **Given** an email address that already has an account, **When** the
   sign-up form is submitted, **Then** the page shows the same "check your
   inbox" step it would show a new user, and the existing account holder
   receives an email telling them someone tried to sign up with their
   address and that they can sign in instead. The page never states that
   the address is taken.
7. **Given** a visitor who closes the tab during the code step, **When**
   they later sign in with the email and password they chose, **Then**
   Story 1 scenario 4 applies and they can still verify and get in.

---

### User Story 3 - Continue with Google (Priority: P3)

From either the sign-in page or the sign-up page, a visitor chooses
"Continue with Google". The browser goes straight to Google's account
chooser and consent screen, then comes back to the product signed in. The
identity provider's own chooser page is never shown between the product
and Google.

**Why this priority**: A second way in that needs no password and no code,
and the only one that still uses a redirect. It reuses the return path
Feature 001 built, so it is cheap to keep and costly to lose.

**Independent Test**: Click "Continue with Google" on the sign-in page,
complete Google's screens, and verify the browser lands on the product's
dashboard signed in, and that the only non-product hosts the browser
visited were Google's.

**Acceptance Scenarios**:

1. **Given** the sign-in or sign-up page, **When** the visitor clicks
   "Continue with Google", **Then** the next page the browser shows is
   Google's, with no intermediate chooser page from the identity provider.
2. **Given** a Google account whose email matches an existing account
   holder, **When** Google sign-in completes, **Then** the visitor is
   signed in as that existing account holder, not as a duplicate.
3. **Given** a Google account with no matching account, **When** Google
   sign-in completes, **Then** an account is created with the name and
   email from Google, no code step is required, and the dashboard greets
   them by name.
4. **Given** the visitor cancels on Google's screen, **When** the browser
   returns, **Then** the sign-in page shows the existing "sign-in didn't
   complete" notice and the visitor can try any method again.
5. **Given** a visitor who started from a protected page, **When** Google
   sign-in completes, **Then** they land on that page.

---

### User Story 4 - Forgot and reset password (Priority: P4)

An account holder who cannot remember their password chooses "Forgot
password?" on the sign-in page, enters their email, and is told to check
their inbox. The email contains a link back to the product's own
reset-password page, where they choose a new password and are then signed
in.

**Why this priority**: Without it, a password user who forgets their
password is locked out for good, since there is no longer a hosted page
offering recovery. It is last because Stories 1 and 2 must exist first and
Google users never need it.

**Independent Test**: Request a reset for an existing email, open the link
from the inbox, set a new password, and verify the old password no longer
works and the new one signs in.

**Acceptance Scenarios**:

1. **Given** the sign-in page, **When** the user submits an email on the
   forgot-password form, **Then** the page says "If that address has an
   account, we've emailed a reset link", identically whether or not the
   address is registered.
2. **Given** a registered email, **When** the reset is requested,
   **Then** an email arrives with a link to the product's reset page that
   works once and expires after a bounded time.
3. **Given** a valid reset link, **When** the user submits a new
   acceptable password, **Then** the password is changed, the user is
   signed in, and the link no longer works.
4. **Given** an expired or already-used link, **When** the reset page is
   opened or submitted, **Then** the page explains the link is no longer
   valid and offers to request a new one.

---

### User Story 5 - Operate the change safely (Priority: P5)

The people running the product can enable the new flows without breaking
existing account holders, know what protection sits in front of the
credential endpoints, and find the governing rules updated.

**Why this priority**: The switch changes the project's stated
authentication model, so the constitution, the API contract, and the
protective configuration must move with the code, or the next feature
builds on stale rules.

**Independent Test**: Read the constitution's authentication section and
the API contract and confirm they describe the flows in Stories 1 to 4;
confirm the edge-network rate limit on the credential endpoints exists;
confirm an account created under Feature 001 still signs in.

**Acceptance Scenarios**:

1. **Given** an account created through the previous hosted page,
   **When** its holder signs in with the new form, **Then** it works with
   no migration step.
2. **Given** the constitution, **When** this feature is complete, **Then**
   its authentication section no longer mandates a hosted login page and
   describes the first-party forms, the direct-to-Google path, and the
   split of brute-force protection between the edge network and the
   platform, with the version bumped accordingly.
3. **Given** the API contract, **When** this feature is complete, **Then**
   every credential exchange the forms perform is described there, and
   the generated client used by the web app is regenerated from it.
4. **Given** the edge network in front of the platform, **When** this
   feature is complete, **Then** a documented rate-limiting rule protects
   the credential endpoints, and the platform's own stricter per-address
   limit on those endpoints remains in place.

### Edge Cases

- **Email verified after a code was requested elsewhere**: a user who has
  two tabs open on the code step and verifies in one is simply signed in;
  entering a code in the other tab either also succeeds or reports the
  code as used. Neither tab errors in a way that loses the session.
- **Code step abandoned**: nothing is left half-created that blocks the
  same email from finishing later (Story 2 scenario 7).
- **Sign-in with a Google-only account and a password**: an account created
  via Google has no password. Submitting the password form for it gives
  the same generic mismatch message as any wrong password. "Forgot
  password?" for such an account sends the reset email, and completing
  the reset gives the account a password.
- **Password sign-up with an email that exists as a Google-only
  account**: treated as scenario 2.6 of Story 2. The existing holder is
  told to sign in with Google.
- **Repeated code requests**: the "send a new code" action cannot be
  triggered more than once every 30 seconds from the same page, and the
  platform's rate limit bounds it further.
- **Very long or unusual input**: names are limited to 100 characters,
  emails to 254, passwords to 256. Leading and trailing whitespace is
  trimmed from names and emails, never from passwords.
- **Session already present**: every page in this feature sends an
  already-signed-in user to the dashboard.
- **Reset link opened while signed in as someone else**: the reset page
  applies to the account the link was issued for, and finishing it
  replaces the current session with that account's.
- **Identity provider partially degraded**: if the account is created but
  the verification email fails to send, the code step still appears with
  a "send a new code" action, and that action retries the send.

## Requirements *(mandatory)*

### Functional Requirements

**Sign-in and sign-up pages**

- **FR-001**: The product MUST present sign-in and sign-up as pages of its
  own, at the same addresses Feature 001 uses for them, and no flow in
  this feature MUST ever navigate the browser to the identity provider's
  hosted pages. The only third-party pages the browser may show are
  Google's during "Continue with Google".
- **FR-002**: The sign-in page MUST offer, in this order: email and
  password fields with a submit action, a "Forgot password?" link, a
  "Continue with Google" action, and a link to sign-up. The sign-up page
  MUST offer: first name, last name, email, and password fields with a
  submit action, a "Continue with Google" action, and a link to sign-in.
- **FR-003**: Both pages MUST preserve the "return to where I was going"
  behavior from Feature 001 for every method, including Google.
- **FR-004**: Both pages MUST validate fields before submitting: email
  must look like an email, password must be at least 8 characters, and
  first and last name must be non-empty on sign-up. Validation messages
  appear next to the field.
- **FR-005**: Both pages MUST disable the submit action and show a busy
  state while a request is in flight, and MUST re-enable it on any
  outcome.

**Password sign-in**

- **FR-006**: The platform MUST exchange the submitted email and password
  with the identity provider on the user's behalf and, on success, start a
  session using exactly the sealed cookie, access token, and refresh
  behavior Feature 001 defines. The web app's session handling, route
  guard, refresh, and sign-out MUST need no change.
- **FR-007**: The platform MUST respond to a wrong password, an unknown
  email, and a password-less (Google-only) account with one and the same
  refusal, with the same timing profile as far as practical, so that
  nothing in the response reveals whether the email is registered.
- **FR-008**: When the identity provider requires the email to be
  verified first, the platform MUST tell the web app to continue with a
  verification step and MUST carry the provider's short-lived pending
  reference so that entering the code completes the same sign-in.

**Email verification**

- **FR-009**: Sign-up MUST create the account with the identity provider
  and immediately move the user to the verification step, and a 6-digit
  code MUST be sent to the email address.
- **FR-010**: The verification step MUST accept a 6-digit code, complete
  the sign-in on a correct code, and report a wrong or expired code
  without leaving the step.
- **FR-011**: The verification step MUST offer "send a new code". Each
  new code invalidates the previous one.
- **FR-012**: When sign-up is attempted with an email that already has an
  account, the platform MUST show the same verification step and MUST
  send that address an email saying an account already exists and how to
  sign in. The response to the browser MUST be indistinguishable from a
  successful new sign-up.

**Google**

- **FR-013**: "Continue with Google" MUST send the browser directly to
  Google's sign-in, with the identity provider's chooser page skipped,
  and MUST return through the callback and session logic Feature 001
  already provides.
- **FR-014**: A Google sign-in whose email matches an existing account
  MUST sign in as that account. One with no match MUST create the
  account with the name and email Google provides, with no verification
  step.

**Password reset**

- **FR-015**: The sign-in page MUST offer a forgot-password form that
  takes an email and always responds with the same neutral confirmation.
- **FR-016**: A registered email MUST receive a reset email with a link to
  the product's own reset page. The link MUST be single-use and MUST
  expire.
- **FR-017**: The reset page MUST accept a new password meeting the same
  rules as sign-up, MUST apply it, MUST sign the user in, and MUST report
  an invalid or expired link with an offer to request a new one.

**Errors and protection**

- **FR-018**: Every refusal the platform sends for these flows MUST be a
  fixed code from a short documented set, MUST carry no provider error
  text, and MUST never distinguish "no such account" from "wrong
  credentials". The web app maps codes to the user-facing messages in
  this specification.
- **FR-019**: The platform MUST keep its existing stricter per-address
  rate limit on the credential endpoints, and the edge network in front
  of the platform MUST apply a documented rate-limiting rule to those
  endpoints. The rule, its thresholds, and where it is configured MUST be
  written down in the repository's operations documentation.
- **FR-020**: Passwords, codes, and reset tokens MUST never appear in
  logs, error reports, or the browser's address bar. Two documented
  exceptions: the reset link's token appears once in the address bar,
  and the reset page MUST remove it once read; and in the local
  development stage only, with no mail provider configured, the
  platform writes outgoing emails (reset link included) to its own log
  so a developer can complete the flow. Deployed stages MUST refuse to
  start without a mail provider, so that path cannot exist there.

**Contract and governance**

- **FR-021**: The API contract MUST describe every request and response
  these flows use, and the web app MUST talk to the platform only through
  the client generated from that contract.
- **FR-022**: The constitution's authentication section MUST be amended to
  describe this model, with a minor version bump, before implementation
  is considered complete.
- **FR-023**: Accounts that exist before this feature MUST keep working
  with no migration and no action from their holders.
- **FR-024**: The platform MUST enforce that the identity provider's
  password method is enabled in the target environment before serving the
  password endpoints, failing fast at startup with a clear message if it
  is not, or the operations documentation MUST record the manual check.
  (Whichever the plan chooses, the failure mode must not be a confusing
  error to end users.)

### Key Entities

- **Account holder**: unchanged from Feature 001. Gains the notion of
  which methods can sign them in: password, Google, or both.
- **Pending verification**: a short-lived reference issued by the identity
  provider when a sign-in or sign-up must first prove control of the
  email. Held only by the web app for the duration of the code step and
  by the identity provider. Never persisted by the platform.
- **Password reset request**: issued when a reset is asked for; carries a
  single-use, expiring token in the emailed link. Owned by the identity
  provider; the platform only relays it.
- **Session**: unchanged from Feature 001 (sealed cookie, in-memory
  access token).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In a full walk of every flow (password sign-in, sign-up with
  code, Google, forgot and reset password, sign-out), the browser visits
  no origin other than the product's and Google's. Verified by the
  browser's navigation history.
- **SC-002**: A returning user with a verified email completes sign-in in
  one form submission and lands on the dashboard within 3 seconds of
  submitting, measured on a normal connection.
- **SC-003**: A new user completes sign-up, including retrieving and
  entering the code, in under 2 minutes.
- **SC-004**: A tester given only the sign-in and sign-up pages cannot
  determine whether an email address has an account, from any message,
  status, or visible timing difference across 20 attempts.
- **SC-005**: 100% of accounts created before this feature sign in
  successfully with the new form on the first attempt, with no migration.
- **SC-006**: The constitution's authentication section and the API
  contract match the shipped behavior, confirmed by a reviewer reading
  both against the running product.
- **SC-007**: Automated tests cover every acceptance scenario in Stories 1,
  2, and 4 without a live identity provider, and the Google flow is
  covered up to the point of leaving for Google and from the point of
  returning.
- **SC-008**: The edge network rate-limiting rule on the credential
  endpoints is in place and documented, and a burst above its threshold
  from one address is refused at the edge without reaching the platform.

## Assumptions

- **Name on sign-up**: first and last name are collected and required on
  the password sign-up form, because the dashboard from Feature 001
  greets the user by first name and the password route has no other
  source for it. Google sign-in takes the name from Google.
- **Verification is required for password accounts**: the identity
  provider is configured to require a verified email before a password
  sign-in succeeds. That is what makes the code step appear.
- **Existing account on sign-up**: the identity provider's headless
  API has no "account exists" email. The platform sends that email
  itself through its transactional mail provider (see Dependencies).
- **Google linking**: the identity provider links a Google sign-in to an
  existing account with the same verified email. The platform does not
  implement its own matching.
- **Reset link target**: the identity provider's headless API issues a
  reset token and a reset URL but sends no email. The platform builds
  the link to its own reset page and sends the email itself.
- **Password rules**: the identity provider decides password strength.
  The product enforces only the 8-character minimum locally and shows
  the provider's reason otherwise.
- **Edge network**: the platform and web app are already deployed behind
  Cloudflare, so the edge rate-limiting rule is configuration, not new
  infrastructure. Its thresholds are chosen during planning (a starting
  point: 10 requests per minute per address on the credential
  endpoints, with a 10-minute block).
- **Email sender**: the verification code email is sent by the identity
  provider from its default sender. The password reset and "account
  exists" emails are sent by the platform through Resend, chosen here
  as the project's transactional mail provider because the send-agreement
  feature needs one anyway and it has a free tier and a plain HTTP API.
  In local development the platform writes those emails to its log
  instead of sending them.
- **No MFA, no magic link, no passkeys, no custom auth domain** in this
  feature. Adding a method later is a new feature.
- **Sign-out and refresh** are untouched. The Feature 001 callback
  endpoint remains, now used only by the Google path.
- **Webhooks** from Feature 001 keep syncing user records; sign-up via the
  new form still produces the same user-created event.

## Dependencies

- Feature 001: session cookie, refresh, sign-out, route guard, callback
  endpoint, user webhook sync, and the existing sign-in and sign-up page
  addresses.
- Identity provider configuration in the SoW staging environment:
  password method enabled, email verification required, Google as an
  enabled social provider, password-reset link pointing at the web app.
- Cloudflare access to add a rate-limiting rule on the credential
  endpoints.
- A Resend account with a verified sending domain (the free tier sends
  only to the account owner's own address until a domain is verified),
  its API key, and a from address, for the reset and "account exists"
  emails in deployed stages.
- A test inbox for the verification, "account exists", and reset emails.
