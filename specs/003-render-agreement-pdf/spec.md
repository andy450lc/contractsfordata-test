# Feature Specification: Render Agreement PDF

**Feature Branch**: `feature/003-render-agreement-pdf`

**Created**: 2026-09-03

**Status**: Draft

**Input**: User description: "lets now implement the rendering PDF. no need
for actually allowing to sign. just generate the PDF and show it after form
is filled." Followed by: "here's the actual SoW we used for feature-002.
save it but gitignore it. base it on that."

**Builds on**: feature 002 (Create Agreement Form). That feature captured
every variable value of the source document in an eight-step wizard and
recorded a template version on each draft "so the renderer built later
knows which wording the values belong to". This feature is that renderer.
Sending, signing links, signatures, and signature evidence stay out of
scope.

**Source document**: the executed "Sieve – Pixels2 India Content
Development Agreement + SOW 1 – Stereo Pilot (Updated)". A copy lives at
`specs/003-render-agreement-pdf/source/source-document.pdf`, excluded from
version control because it is a real executed agreement. It is nine US
Letter pages: a master Content Development Agreement in twelve numbered
sections with a signature block, then Exhibit A, a Statement of Work in
ten numbered sections with its own signature block. Its fixed sentences
are the template wording. Its variable values are the form's fields. The
section "Reference: structure of the source document" at the end lists
every section, table, and value position.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sender generates and reads the document (Priority: P1)

A sender who has completed every step of the wizard and saved their
organization info reaches the Review step. Beside the disabled send action
they see a "Generate PDF" action. They press it, watch a short progress
state, and the finished document opens in the app: the master Content
Development Agreement followed by Exhibit A, the Statement of Work, with
every value they typed in the place the form's tooltips promised. They
page through it, zoom in on a table, and confirm it reads like the source
document.

**Why this priority**: the document is the product. Until a sender can see
the agreement their answers produce, the form is a data-entry exercise
with no output.

**Independent Test**: recreate the source document in the form, generate
the PDF, and compare it page by page against the source document. Every
variable value appears once in its place, every fixed sentence matches
word for word, and the sections come in the same order.

**Acceptance Scenarios**:

1. **Given** a draft with all seven steps saved and organization info
   saved, **When** the sender opens the Review step, **Then** a primary
   action "Generate PDF" is enabled next to the disabled "Send agreement &
   signing links" action.
2. **Given** the Review step, **When** the sender presses "Generate PDF",
   **Then** the action shows a busy state, and within the time limit in
   SC-003 the document page opens showing page 1 of the rendered
   agreement.
3. **Given** the document page, **When** the sender reads page 1, **Then**
   it opens with the heading "CONTENT DEVELOPMENT AGREEMENT" and a
   preamble naming the effective date, both parties by legal name, entity
   type, and jurisdiction, the Developer's address, and the defined terms
   for each party, and each page carries "Page N of M" and the word
   "DRAFT".
4. **Given** the document page, **When** the sender uses the page
   controls, **Then** they can move to any page, zoom, and return to the
   Review step or the agreements list without losing the draft.
5. **Given** a draft whose values match the source document, **When** the
   sender reads the generated document beside the source document,
   **Then** every value entered on steps 1 to 7 and every organization
   field appears at the location its field tooltip names, and the
   document is within two pages of the source document's nine.

---

### User Story 2 - Document wording follows the form's choices (Priority: P1)

The wizard holds yes/no choices and lists that change which sentences the
document carries. The sender flips a choice, regenerates, and sees the
matching wording appear or disappear. In invite-only counterparty mode,
where the Developer's details are unknown, the document shows clearly
marked blanks where those details will go.

**Why this priority**: a document that prints the exclusivity clause for a
non-exclusive deal is wrong in a way a lawyer will catch and a sender may
not. Conditional wording is what makes the template a template.

**Independent Test**: for each conditional choice, generate once with yes
and once with no, and confirm the governed sentences are present in one
document and absent in the other. Generate once in invite-only mode and
confirm every Developer detail is a labelled blank.

**Acceptance Scenarios**:

1. **Given** Exclusive engagement is yes, **When** the sender generates,
   **Then** the SOW title ends in "(EXCLUSIVE)", the Overview describes
   the footage as "exclusive", and the sentence "The engagement and all
   Deliverables are exclusive." appears. **Given** it is no, **Then** all
   three are absent.
2. **Given** Final deadline is firm is yes, **When** the sender generates,
   **Then** the Delivery Schedule paragraph ends with the firm-obligation
   sentence, the reduce-or-terminate remedy, and the written-extension
   sentence. **Given** it is no, **Then** all three are absent.
3. **Given** Deposit or advance is yes with an amount, **When** the sender
   generates, **Then** the Pricing section states the deposit and its
   amount. **Given** it is no, **Then** it states "There is no deposit,
   advance payment, or hardware contribution."
4. **Given** Additional prohibited content or Additional excluded settings
   holds items, **When** the sender generates, **Then** those items follow
   the fixed list in the same sentence of Collection Requirements, and
   the fixed list is unchanged when they are empty.
5. **Given** counterparty mode is "Invite by email only", **When** the
   sender generates, **Then** the Developer's legal name, entity and
   jurisdiction, and address in the preamble, and the name and title in
   both signature blocks, each render as a bracketed blank naming the
   field, for example "[Developer legal name]".
6. **Given** the Environment step's verticals, difficulty mix, and
   diversity caps, **When** the sender generates, **Then** the verticals
   appear as the Environment and Task Mix table with the same rows in the
   same order, and the mix and caps appear in the two paragraphs beneath
   it with the unit label from the Scope step.
7. **Given** Collection plan required is no, **When** the sender
   generates, **Then** the Collection Plan subsection is absent and the
   remaining Collection Requirements subsections are renumbered without a
   gap.

---

### User Story 3 - Sender regenerates and returns to the document (Priority: P2)

A sender who generated once goes back, changes a value, and generates
again. The new document reflects the change and no stale copy is offered.
From the agreements list, a draft that has been fully filled in offers
"View PDF", which opens the current document for that draft. A sender can
also save the document to their own machine.

**Why this priority**: senders iterate. The YC model lets a draft be
reopened and reviewed any number of times before sending, and the document
must always match what the form holds.

**Independent Test**: generate, change the unit fee, generate again, and
confirm the new fee and the new maximum total appear in the Pricing table.
Open the same draft from the list, choose "View PDF", and confirm the same
document opens. Save the document and confirm the saved file opens outside
the app with the same pages.

**Acceptance Scenarios**:

1. **Given** a draft that was generated earlier, **When** the sender
   changes a value on any step and generates again, **Then** the document
   shows the new value and the previous document is no longer reachable.
2. **Given** a fully filled draft in the agreements list, **When** the
   sender chooses "View PDF" on its row, **Then** the document page opens
   with the current values.
3. **Given** the document page, **When** the sender chooses Download,
   **Then** a file named "SOW {number} - {SOW title}.pdf" is saved and
   opens in a standard PDF reader with the same page count.
4. **Given** a draft that has not been fully filled in, **When** the sender
   looks at its row in the list, **Then** "View PDF" is absent.

---

### Edge Cases

- A step is unsaved, or organization info is missing: "Generate PDF" is
  disabled and a note beneath it names what is missing, for example
  "Complete Counterparty and add your organization info to generate the
  PDF." Every incomplete item links to its step or page.
- A table is longer than one page: rows continue on the next page with
  the column headings repeated, and no row is split across pages.
- A long free-text value (manifest format, integrity requirements,
  address, a vertical's details) wraps within its cell or paragraph and
  never overflows the page.
- A conditional subsection is omitted (Collection Plan): the subsections
  after it renumber, and no fixed cross-reference in the template points
  at a subsection that can be omitted. The fixed cross-references are to
  master Sections 4, 5, 6.2, and 8.3 and SOW Sections 4, 8, and 9, all of
  which always render.
- The draft's template version is one the renderer does not know: the
  action shows "This draft uses a template this version cannot render."
  and nothing is generated.
- Generation fails for any other reason: the sender sees "Couldn't
  generate the PDF. Try again." with a retry control, and the draft is
  unchanged.
- The sender leaves the page while generation is in progress: nothing is
  shown later, and pressing the action again starts a fresh generation.
- Unit fee times target volume exceeds the money limit: the form already
  blocks this, and the renderer refuses to render a value above the
  limit rather than printing a truncated number.
- Two browser tabs hold the same draft: each generation uses the values
  stored at the moment the action is pressed.
- A currency other than US dollars: amounts print with the currency code
  in place of "US $", for example "EUR 20.00", and the Invoicing sentence
  names the currency by its code.
- The sender's short name equals "Developer": the document still renders,
  and both short names are visible on the review step so the sender can
  fix it.
- A milestone's Target volume is free text such as "Completed data as
  available": it prints as typed with no unit appended. A numeric value
  prints with the unit label.
- A milestone has notes: the Deadline cell shows the notes followed by
  the date, as the source's "Daily after collection begins, no later than
  September 6, 2026".

## Requirements *(mandatory)*

### Functional Requirements

**Entry points**

- **FR-001**: The Review step MUST show a primary action "Generate PDF"
  beside the disabled "Send agreement & signing links" action. The action
  MUST be enabled only when steps 1 to 7 are saved and organization info
  is saved. When disabled, a note beneath it MUST name each missing step
  or the missing organization info, each as a link to that step or page.
- **FR-002**: The agreements list MUST show a "View PDF" action on every
  draft row whose steps 1 to 7 are saved and whose sender has organization
  info. The action opens the document page for that draft.
- **FR-003**: While a document is being generated, the action MUST show a
  busy state and MUST ignore repeated presses until generation finishes or
  fails.

**Document page**

- **FR-004**: The document page MUST show the generated document inside
  the app with page navigation, a page indicator "N of M", zoom, a
  Download control, a "Back to review" link, and a "Back to my
  agreements" link. Nothing on the page offers signing.
- **FR-005**: Download MUST save a file named "SOW {number} - {SOW
  title}.pdf" whose contents equal the document shown.
- **FR-006**: Every generation MUST use the values stored for the draft at
  the moment the action is pressed. A document generated from earlier
  values MUST NOT be offered after a later generation completes.

**Document content**

- **FR-007**: The document MUST consist of, in order: the heading "CONTENT
  DEVELOPMENT AGREEMENT", the preamble, master Sections 1 to 12 with
  their numbered subsections, the "IN WITNESS WHEREOF" sentence and the
  two-column signature block, then on a new page the heading "EXHIBIT A",
  the SOW title line, the SOW preamble, SOW Sections 1 to 10, and the
  "IN WITNESS WHEREOF" sentence and signature block again. Section
  numbers, headings, and order MUST match the reference at the end of
  this document.
- **FR-008**: Every fixed sentence MUST match the source document's
  wording for the template version recorded on the draft, word for word,
  apart from the corrections listed in Assumptions.
- **FR-009**: Every value from steps 1 to 7 and every organization field
  MUST appear at the position listed in the reference at the end of this
  document, which is the position the form's help tooltips name. Values
  MUST appear exactly as entered, with the formatting in FR-011.
- **FR-010**: Conditional wording MUST follow the form's choices:
  - Exclusive engagement governs the "(EXCLUSIVE)" title suffix, the word
    "exclusive" in the Overview's first sentence, and the sentence "The
    engagement and all Deliverables are exclusive."
  - Final deadline is firm governs the last three sentences of the
    Delivery Schedule paragraph: the firm-obligation sentence, the
    reduce-or-terminate remedy, and the written-extension sentence.
  - Deposit or advance governs the Pricing section's last sentence: the
    no-deposit sentence when no, or a sentence naming the deposit amount
    when yes.
  - Ambient audio permitted governs the last sentence of Deliverables: the
    no-audio sentence when no, or "Ambient audio is permitted." when yes.
  - Collection plan required governs the whole Collection Plan
    subsection, including the change implementation window.
  - Every item is a condition of payment governs the sentence "Every
    technical item is a condition of acceptance and payment." When no, the
    sentence reads "Every technical item is a condition of acceptance."
  - Deemed acceptance governs the sentence beginning "A batch not rejected
    within the ... review period is deemed accepted".
  - Rejected material stays Developer-owned governs the Overview's
    "subject to the rejected-material carveout in Section 9" and the last
    two sentences of Acceptance. When yes, the deletion-window sentence
    carries the deletion window in days.
  - Additional prohibited content appends to the list of prohibited
    captures in Privacy and Sensitive Content. Additional excluded
    settings appends to the list of excluded settings in Natural Activity.
- **FR-011**: Values MUST be formatted for reading:
  - Money in US dollars prints as "US $" followed by the amount with
    thousands separators and two decimals for unit fees and deposits, and
    no decimals for whole-dollar totals, as in "US $20.00" and "Maximum
    US $6,000". Other currencies print the currency code in place of
    "US $".
  - Dates print as month name, day, and year, for example
    "September 6, 2026".
  - Percentages print with "%". Durations print as digits with their unit
    word, for example "24 hours", "30 days", "five business days" becomes
    "5 business days".
  - Quantities in the unit of acceptance print with the full unit label
    where the source uses the full phrase, and with the label's final word
    where the source uses the bare word, singular for 1 and plural
    otherwise, for example "300 accepted usable hours" and "easy 1 hour".
  - Deliverables print as the lettered list "(a)" to "(n)" in one
    sentence. Pre-collection materials and site data fields print as a
    comma-separated series inside their sentences. Required manifest
    fields print as a comma-separated series in their table cell.
  - The Pricing, Delivery Schedule, Environment and Task Mix, Technical
    and Quality Standards, and Delivery / Manifest tables print with the
    column headings the source document uses.
- **FR-012**: Each signature block MUST show the two parties side by side
  with the Customer on the left and the Developer on the right, each with
  the legal name as the heading, then "Signature:", "Name:", "Title:",
  and "Date:" lines, with Name and Title filled from organization info and
  the counterparty step and Signature and Date left blank.
- **FR-013**: In invite-only counterparty mode, the Developer's legal
  name, entity and jurisdiction, and address in the preamble and the name
  and title in both signature blocks MUST render as bracketed blanks
  naming the field, for example "[Developer legal name]". Running text is
  unaffected because it refers to the counterparty as "Developer".
- **FR-014**: Every page MUST carry a footer with "Page N of M" and the SOW
  number, and the word "DRAFT" MUST appear on every page, since no
  document produced by this feature can be signed.
- **FR-015**: The document MUST be produced on US Letter pages with
  margins, fonts, and spacing that keep every table and paragraph inside
  the page, with table headings repeated when a table continues on the
  next page and no row split across pages.
- **FR-016**: The document MUST be a standard PDF that opens in common
  readers with its text selectable and searchable.

**Where the document is produced and kept**

- **FR-017**: The document MUST be produced on the sender's own device
  from the draft held there, with no copy kept anywhere after the sender
  leaves the document page. Nothing is sent to the service and nothing is
  stored. This is an interim preview, like the feature 002 stand-in for
  drafts. The canonical document that signatures cover is rendered and
  stored by the service in the sending feature, which reuses this
  feature's template and layout so the two renders match.
- **FR-018**: Generation MUST record the template version the document was
  produced from. A draft whose template version the renderer does not
  know MUST be refused with the message in Edge Cases.

**Access and privacy**

- **FR-019**: Only the sender who owns a draft MUST be able to generate,
  view, or download its document. Any other sender's request MUST be
  answered as not found.
- **FR-020**: Document contents, party names, emails, and addresses MUST
  never appear in logs, metrics, or error messages. Failures are reported
  by draft id and template version only.
- **FR-021**: A generated document MUST live no longer than the document
  page that shows it. Leaving the page discards it, and deleting a draft
  leaves no document behind.

### Key Entities

- **Document template**: one per template version. The fixed wording of
  the master agreement and the SOW, the position of every field, the
  conditional wording rules in FR-010, and the table layouts.
- **Rendered document**: produced from one draft at one moment and held
  only while the document page is open. The draft it came from, the
  template version, the time of generation, the page count, and the file
  itself. Replaced by the next generation of the same draft.
- **Agreement** (from feature 002): gains a "document available" state
  derived from step completion and organization info.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With the source document's values entered, every variable
  value from steps 1 to 7 and organization info appears in the generated
  document, verified field by field against the reference list with zero
  omissions.
- **SC-002**: Every fixed sentence of the generated document matches the
  source document's wording with zero differences beyond the listed
  corrections, verified by comparing the extracted text of the fixed
  passages against the source.
- **SC-003**: For a deal like the source document, the document page shows
  page 1 within 5 seconds of pressing "Generate PDF" in 95% of attempts on
  a current laptop browser.
- **SC-004**: Each of the nine conditional wording rules in FR-010 is
  verified with both settings, and 100% produce the correct presence or
  absence of the governed sentences.
- **SC-005**: A reader holding the source document finds every section of
  the generated document in the same order with the same number and
  heading, with zero reorderings.
- **SC-006**: A sender who changes a value and regenerates sees the new
  value in the document 100% of the time, and never sees a document
  produced from earlier values.
- **SC-007**: No sender can generate, view, or download another sender's
  document, verified by attempting every document address and action with
  a second account.
- **SC-008**: The downloaded file opens in at least three common PDF
  readers with the same page count and selectable text.
- **SC-009**: The document rendered from the source document's values is
  between 8 and 11 pages.

## Assumptions

- The document is a draft preview. It carries "DRAFT" on every page and
  blank signature lines, because sending and signing arrive in later
  features. The sending feature removes the marking when it freezes the
  canonical document.
- The template wording is transcribed from the saved source document into
  the template for version "content-development-1", the version feature
  002 records on every draft. The source document stays out of version
  control. The transcribed template wording is part of the product and is
  committed.
- The counterparty is referred to as "Developer" throughout, as the source
  document does, and the Customer is referred to by the short name from
  organization info, as the source uses "Sieve". The counterparty short
  name captured in feature 002 is not used by this template version. It
  stays in the form for a future template that needs it.
- Corrections to the source document applied in the template: the master
  agreement's deliverable description comes from the materials summary
  field, and the SOW carries its own effective date (both from feature
  002); the Customer's address is added to the preamble beside the
  Developer's, since notices need it; the one date written as "13
  September 2026" is printed in the same long form as every other date;
  the signature heading "Sieve, Inc:" is printed as the legal name from
  organization info; numbers written as words in the source ("two years",
  "two times", "five business days") print as digits so every
  standard-terms value renders the same way.
- Page size is US Letter, matching the source document. A4 is not offered
  in this feature.
- "Show it after form is filled" means the document is generated on
  demand from the Review step and from the list, and is not generated
  silently in the background as steps are saved.
- The document is view-only. Zoom and page navigation are the only
  controls beyond Download. Printing uses the reader's own print function
  on the downloaded file.
- The Review step's "Send agreement & signing links" action stays
  disabled with its "Sending arrives in a later release" note.
- Decided 2026-09-03 with the user: the document is produced in the
  browser from the local draft and is never stored. The three options
  weighed were a browser-only preview, moving drafts to the service so it
  renders and stores, and service rendering from browser-held drafts. The
  browser-only preview was chosen because feature 002 has no agreements
  service yet and the user asked only to generate and show the document.
- Retention follows the page: a document lives only while its document
  page is open, and deleting a draft leaves nothing behind.

## Reference: structure of the source document

Read from the saved copy on 2026-09-04. Values in braces name the form
field or organization field that fills that position. Everything else is
fixed wording.

### Master agreement

- **Heading**: "CONTENT DEVELOPMENT AGREEMENT".
- **Preamble**: effective as of {effective date}, between {organization
  legal name}, a {organization entity type} of {place of incorporation},
  with an address at {organization address} ("{organization short
  name}"), and {counterparty legal name}, a {entity type and
  jurisdiction}, with an address at {counterparty address}
  ("Developer"). Both are each a "Party" and together the "Parties".
- **1. Services**: 1.1 Content Development Services, with {materials
  summary} as the description of what Personnel create. 1.2 SOW.
- **2. Contributor Management**: 2.1 Developer Responsibilities. 2.2
  Consents and Site Permissions. 2.3 Subcontractors.
- **3. Submission and Acceptance**: 3.1 Submissions. 3.2 Acceptance
  Testing.
- **4. Work Product; Exclusive Rights**: 4.1 Ownership. 4.2 Exclusivity.
  4.3 Backup License.
- **5. Compliance with Laws and Data Protection**: 5.1 Developer
  Responsibility. 5.2 Lawful Collection and Delivery. 5.3 Prohibited
  Material. 5.4 Records, with {compliance records retention} years.
- **6. Confidentiality**: 6.1 Confidential Information. 6.2 Restrictions.
  6.3 Publicity.
- **7. Security**: 7.1 Safeguards, with {security incident notice window}
  hours.
- **8. Term and Termination**: 8.1 Term, with {breach cure period} days
  and {termination for convenience notice} days. 8.2 Effect of
  Termination. 8.3 Survival.
- **9. Representations and Warranties**: 9.1 Developer Warranties.
- **10. Indemnification**: 10.1 Mutual Indemnity.
- **11. Limitation of Liability**: 11.1 Limitations, with {liability
  look-back period} months, {excluded-claims cap multiplier} times fees,
  and {data and IP claims cap floor} as "US $50,000".
- **12. General**: 12.1 Independent Contractors. 12.2 Amendments; No
  Waiver. 12.3 Severability. 12.4 Governing Law; Venue, with {governing
  law jurisdiction} twice. 12.5 Entire Agreement.
- **Signature block**: "IN WITNESS WHEREOF, the parties have executed this
  Agreement as of the Effective Date." Left column {organization legal
  name} with Signature, Name {signer name}, Title {signer title}, Date.
  Right column {counterparty legal name} with Signature, Name {signatory
  name}, Title {signatory title}, Date.

The Customer's short name appears throughout the master agreement and the
SOW wherever the source says "Sieve", including possessives.

### Exhibit A, Statement of Work

- **Headings**: "EXHIBIT A", then "SOW {SOW number} — {SOW title}" in
  capitals, with "(EXCLUSIVE)" appended when exclusive.
- **Preamble**: entered into under the Content Development Agreement
  between {organization legal name} and {counterparty legal name}, dated
  {effective date}.
- **1. Overview**: Developer will collect, quality-control, and deliver
  {target volume} {unit, plural} of {"exclusive," when exclusive}
  {deliverable description} captured in {venue constraint} in {collection
  country or region}. The exclusivity sentence when exclusive. The Work
  Product sentence, with the Section 9 carveout when rejected material
  stays Developer-owned.
- **2. Deliverables**: "Each delivery will include:" followed by the
  lettered {deliverables} list. The ambient audio sentence.
- **3. Pricing**: table with columns Unit Fee, Target Volume, Total,
  Payment Basis. Unit Fee is {unit fee} "per" {unit, singular}. Target
  Volume is {target volume} {unit, plural}. Total is "Maximum" {maximum
  total}. Payment Basis is {unit, plural} "only;" {invoicing cadence}
  "invoicing in arrears; Net" {payment terms}. Then the pays-only-for
  accepted sentence and the deposit sentence.
- **4. Delivery Schedule**: table with columns Delivery Milestone, Target
  Volume, Deadline, one row per {milestone}. Then the paragraph: rolling
  uploads no later than the first milestone's deadline, the full {target
  volume} by end of day on the final milestone's deadline, and the three
  firm-deadline sentences when firm.
- **5. Environment and Task Mix**: table with columns Collection
  Verticals, Target Hours or Percentage, Excluded or Required Details,
  one row per {vertical}. Then the paragraph with {maximum from one
  category}, {minimum distinct sites}, the {difficulty mix} percentages,
  and {skilled share}. Then the paragraph with the {diversity caps} in
  the source's order and the two {repetitive motion limits}.
- **6. Collection Requirements**: 6.1 Collection Plan, with {change
  implementation window} hours, present when a plan is required. 6.2
  Natural Activity, with {additional excluded settings} appended to its
  list. 6.3 Privacy and Sensitive Content, with {additional prohibited
  content} appended to its list.
- **7. Technical and Quality Standards**: table with columns Requirement,
  Default Specification, Rejection Criteria, one row per {technical
  requirement}. Then the condition-of-payment sentence, the sentence
  listing {pre-collection materials}, and the non-conforming sentence.
- **8. Delivery Location, Manifest, and Metadata**: the delivery
  paragraph with {delivery cadence}. Then the table with columns Delivery
  / Manifest Field and Required Value, rows Storage / transfer location
  {storage and transfer location}, Manifest format {manifest format},
  Required fields {required manifest fields}, Site data {site data
  fields} "per site, which" {short name} "may assign.", Integrity and
  security {integrity and security requirements}. Then the missing
  metadata sentence.
- **9. Acceptance**: {acceptance review window} days, the definition of
  the unit of acceptance using {unit, singular}, {correction or
  replacement window} business days, the deemed-acceptance sentence when
  enabled, and the two rejected-material sentences with {deletion window}
  days when enabled.
- **10. Invoicing and Payment**: invoice {invoicing cadence} in arrears
  for {unit, plural}; due Net {payment terms} days after receipt; paid in
  {currency} by {payment methods}; invoices to {invoice email}.
- **Signature block**: identical to the master agreement's.
