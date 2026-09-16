# Feature Specification: Gated Word Template Delivery

**Feature Branch**: `feature/005-gated-template-delivery`

**Created**: 2026-09-15

**Status**: Approved for implementation

**Input**: Product-owner amendment replacing every existing contract-template
download and email path with a legal-disclaimer gate followed by Word download
or Word email delivery.

This feature supersedes the customer-facing PDF generation, preview, and
download requirements in feature 003. It does not change the later signed-copy
architecture, where an executed canonical PDF may still be created and stored.

## User Scenarios and Testing

### User Story 1 - Reader acknowledges the disclaimer (Priority: P1)

Every action that can supply a contract opens a modal titled **Before You
Download**. The modal presents the following two paragraphs exactly:

> This template is provided for general reference only and does not constitute
> legal advice. It has not been reviewed by a licensed attorney for your
> specific situation, and no attorney-client relationship is created by
> downloading or using it.

> Data-collection and data-transfer laws vary by jurisdiction and change over
> time. You are responsible for reviewing, adapting, and validating this
> document — including consulting a lawyer licensed in each relevant
> jurisdiction — before relying on it or using it in any transaction.

The acknowledgment is initially unavailable. The reader must reach the bottom
of the actual disclaimer container before the following checkbox becomes
available:

> I acknowledge that this is not legal advice, and that I am responsible for my
> own legal review before use.

Reaching the bottom does not continue the flow. The reader must actively check
the checkbox, then choose Continue. The next step presents exactly two primary
delivery actions: **Download Word document** and **Email Word document**.

**Acceptance scenarios**:

1. Every marketing and configurator delivery trigger opens the disclaimer and
   supplies no file before acknowledgment.
2. A scrollable disclaimer unlocks within a two-pixel rounding tolerance only
   after its bottom is reached. A non-scrollable disclaimer unlocks when the
   modal is measured.
3. The read state is monotonic during one attempt and is recalculated when the
   modal opens or resizes.
4. The checkbox begins unchecked and cannot be selected until unlocked.
5. Continue stays disabled until the checkbox is checked.
6. Cancel, Escape, and close produce no document and send no email.
7. Closing, reopening, completing delivery, or changing configuration resets
   all disclaimer and delivery state while preserving contract configuration.
8. The modal has an associated title and description, traps focus, supports the
   keyboard, announces unlock/error/success states, and restores trigger focus.

### User Story 2 - Reader downloads the configured Word document (Priority: P1)

After acknowledgment, the reader chooses **Download Word document**. The
service validates the acknowledgment and all submitted configuration, creates
one deterministic OOXML package, and responds with the genuine `.docx` file.

**Acceptance scenarios**:

1. Missing or false acknowledgment is rejected by the server.
2. The response has MIME type
   `application/vnd.openxmlformats-officedocument.wordprocessingml.document`, a
   sanitized `.docx` filename, and `Cache-Control: no-store`.
3. The ZIP contains `[Content_Types].xml` and `word/document.xml` and opens as a
   Word document.
4. The document contains the fixed canonical language, the active configured
   values, exactly the selected conditional clauses, and editable blanks where
   the user intentionally left a party for later completion.
5. The document contains no unresolved tokens, internal instructions,
   `undefined`, `null`, or `NaN`.
6. Rapid repeated clicks make one in-flight request. A failed request produces
   an honest retryable error. The UI claims no success before the file response
   exists.
7. No configuration, acknowledgment, or personal information enters a URL.

### User Story 3 - Reader receives the same Word document by email (Priority: P1)

After acknowledgment, the reader enters or edits a labeled work email and
chooses **Email Word document**. The server normalizes and validates the
address, invokes the same generator used by direct download once, and attaches
the returned bytes to a concise transactional email.

**Acceptance scenarios**:

1. Email is required only for this option and is validated on client and
   server.
2. The attachment bytes, MIME type, and filename are the same artifact the
   download path produces for the same configuration.
3. The Resend provider receives the attachment as Base64 and receives an
   `Idempotency-Key`; duplicate submission is suppressed while pending and by
   the provider for its idempotency window.
4. Success appears only after a configured provider returns an accepted message
   id. Provider rejection, malformed success, and unconfigured development
   transport produce an honest retryable error.
5. Provider credentials and sender configuration stay server-side. Recipient
   addresses and contract contents are absent from logs, metrics, URLs, and
   stored state.

### User Story 4 - Public marketing reader configures a Word agreement (Priority: P1)

The static marketing site offers a public, no-login configurator that is fully
separate from the authenticated eight-step configurator at `/agreements/new`.
It collects optional client information and four fixed clause choices before
showing the same disclaimer gate. Header, hero, inline, mobile, error-page, and
future global-header triggers cannot link to a static file.

The optional client information is a delivery preference, party type
(`supplier`, `buyer`, or `other`), and company name. Client information remains
optional until the product owner decides otherwise. An email address is used
only for email delivery and is never inserted into the agreement. A company
name is not silently assigned to either contract party.

The four fixed questions are:

1. Subcontracting: never allowed; allowed only with the Buyer's prior written
   consent (default); or allowed with notice and a Buyer objection right.
2. Jurisdiction rules: added to the Agreement by country; placed in each SOW's
   country appendix (default); or placed in a separate country rider.
3. Data and intellectual-property liability cap: the greater of an editable
   dollar blank or editable fee-multiplier blank (default); an editable
   fee-multiplier blank only; or no separate special cap.
4. Exclusivity: fully exclusive worldwide; exclusive only in a supplied field;
   or non-exclusive. Field-limited exclusivity requires bounded field text.

The generated agreement includes fixed security language with 24-hour incident
notice, fixed rights-and-consents representations and warranties, and a general
liability tier plus the selected data/IP tier. It includes exactly one selected
branch for each of the four questions. Editable blanks are visible contract
fields and are not unresolved template tokens.

Public email controls are included in the implementation but remain hidden in
production builds unless `PUBLIC_CONFIGURATOR_EMAIL_ENABLED` is explicitly set
after a real provider-backed end-to-end send succeeds. An unverified or
unconfigured provider cannot be exposed as a delivery choice.

**Acceptance scenarios**:

1. The public form defaults subcontracting and jurisdiction to option B and the
   data/IP cap to option A.
2. The form sends the four answers and optional client information in a JSON
   body to a new public API resource. No form value appears in a URL.
3. Field-limited exclusivity requires field-scope text on client and server.
4. Company name and field-scope text are bounded, normalized, stripped of XML-
   invalid characters, and escaped when rendered into OOXML.
5. Every combination of the four three-way choices produces a valid DOCX and
   contains only the chosen language for each category.
6. The authenticated `/agreements/new` route, its request mapping, and its API
   behavior are unchanged.

## Functional Requirements

- **FR-001**: All contract-delivery triggers MUST enter the disclaimer flow.
- **FR-002**: Disclaimer content, paragraph structure, title, and acknowledgment
  MUST match this specification exactly.
- **FR-003**: Scroll unlock MUST use container measurements with a small browser
  rounding tolerance and a resize-aware recalculation. It MUST NOT use time.
- **FR-004**: Read, acknowledgment, method, email, loading, error, and success
  state MUST remain distinct and reset for every new attempt.
- **FR-005**: Both server operations MUST require `acknowledged: true` and MUST
  reject malformed or conditionally invalid configuration.
- **FR-006**: A single request-scoped document generator MUST create the DOCX
  artifact used by both direct and email delivery.
- **FR-007**: User text MUST be bounded, stripped of XML-invalid control
  characters, and escaped by the OOXML writer.
- **FR-008**: Email delivery MUST require a valid `Idempotency-Key`, use a
  dedicated limiter, and forward the key to Resend.
- **FR-009**: The server MUST retain no acknowledgment, recipient, contract
  configuration, or generated document after a request finishes.
- **FR-010**: `/template.pdf`, browser PDF blobs, direct download anchors, and
  the old document route MUST NOT remain as alternate contract-delivery paths.
- **FR-011**: Existing routing, responsive styles, semantic controls, Cloudflare
  CSP stamping, npm lockfiles, and provider abstraction MUST be preserved.
- **FR-012**: Public configured generation MUST use dedicated
  `/public/sow-configurator` operations and dedicated DTO validation. It MUST
  NOT call or alter the authenticated configurator endpoint.
- **FR-013**: The public generated document MUST contain the selected
  subcontracting, jurisdiction, data/IP cap, and exclusivity branches, along
  with fixed security, representations and warranties, and general liability
  language.
- **FR-014**: Liability amounts and multipliers remain editable blanks pending
  product-owner instruction. The output MUST NOT contain `[TBD]` tokens.
- **FR-015**: Jurisdiction-specific overlays remain part of the four-choice
  jurisdiction question. A fifth question is deferred pending product-owner
  instruction.

## Authenticated Document Combination Matrix

The repository did not contain the referenced 81-case suite at feature start.
This feature establishes it as a deterministic cross-product of three states in
each of four clause groups:

1. Agreement/scope: baseline, exclusive, or ambient-audio permitted.
2. Pricing: baseline, firm deadline, or deposit required.
3. Collection/technical: baseline, collection plan, or payment condition.
4. Acceptance: baseline, deemed acceptance, or rejected material remains
   developer-owned.

The 3 × 3 × 3 × 3 suite verifies valid OOXML, expected chosen language,
absence of unselected alternatives, and absence of unresolved values for all 81
documents. Focused tests cover combinations of flags that share a group.

The public configurator has a separate 3 × 3 × 3 × 3 matrix for its four
public clause questions. It verifies package integrity, one selected clause per
category, fixed clauses, and absence of internal placeholders for all 81
documents.

## Success Criteria

- Every delivery route is gated in both frontends and on the server.
- Direct and email artifacts are genuine, deterministic DOCX files with equal
  bytes for equal configuration.
- Both 81-case matrices and focused clause tests pass.
- No PDF or static contract download remains.
- Full backend, web, and marketing checks pass, subject to locally available
  infrastructure.

## Assumptions

- No canonical `.docx` asset exists in the repository. The new OOXML template
  derives its structure, wording, tables, and typography from the checked-in
  `content-development-1` implementation.
- Marketing delivery is intentionally public because the blank template is a
  public resource and the static site has no account session. These operations
  accept no server-owned resource id, persist nothing, require an allowed
  Origin, and receive a dedicated rate limiter. This is the narrow exception to
  the constitution's account-authentication rule.
- Unsigned, request-scoped template bytes pass through the API so direct
  download and direct email attachment can share one authoritative generator.
  This is the narrow exception to the constitution's R2 document-flow rule.
  Executed canonical PDFs and signature evidence remain subject to the existing
  immutable R2 architecture.
- Exact liability dollar figures and fee multipliers remain undecided. The
  public agreement uses editable underscore blanks.
- Client information is optional. Whether to require it is a product-owner
  decision.
- Jurisdiction overlays remain fixed template content controlled by the four-
  choice question. Whether to add a fifth question is deferred.
