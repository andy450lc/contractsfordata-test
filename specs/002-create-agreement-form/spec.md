# Feature Specification: Create Agreement Form

**Feature Branch**: `feature/002-create-agreement-form`

**Created**: 2026-09-03

**Status**: Draft

**Input**: User description: "just extract everything. make a multi page
form. open chrome, drive https://www.ycombinator.com/tools/safe/create and
see how they have modelled it. follow that exactly. get the form right
first. we will iterate"

**Source document**: the executed "Sieve – Pixels2 India Content Development
Agreement + SOW 1 – Stereo Pilot" (master Content Development Agreement plus
Exhibit A, SOW 1). Every value in that document that changes from deal to
deal is a field in this form. Every sentence that does not change is fixed
template text and is not editable.

**Reference model**: the Y Combinator "Send a SAFE" creator. Its structure
is reproduced here: organization details saved once in settings and pulled
into every agreement, a numbered multi-step wizard with a progress bar, a
draft persisted as soon as the first step completes, a counterparty step
with a "details now" or "invite by email only" choice, a read-only review
step, and a list page with status counts. See the section "Reference:
how the YC SAFE creator is modelled" at the end.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sender saves organization info once (Priority: P1)

A signed-in sender opens Organization info from the agreements list, enters
their organization's legal name, entity type, place of incorporation,
governing law jurisdiction, address, the name, title, and email of the
person who signs for them, a contact phone, and the short name the document
uses for them. They save. Every agreement they create from then on shows
these details on its review step without re-entry. Agreements already sent
keep the organization details they were sent with.

**Why this priority**: the sender's own party block, signature block, and
governing law appear in every agreement. The YC model keeps these out of
the wizard entirely, and the wizard cannot render a complete review step
without them.

**Independent Test**: sign in, open Organization info, fill every required
field, save, reload, and confirm every value is shown again. Start a new
agreement and confirm the review step displays the saved values with an
"Edit organization info" link.

**Acceptance Scenarios**:

1. **Given** a signed-in sender with no saved organization info, **When**
   they open Organization info, **Then** they see an empty form whose
   labels, required markers, help tooltips, and example placeholders match
   the field list in FR-002.
2. **Given** the form with a required field empty, **When** they save,
   **Then** the empty field shows the message "Required" beneath it and
   nothing is saved.
3. **Given** a fully valid form, **When** they save, **Then** the values
   persist and the sender returns to the agreements list.
4. **Given** saved organization info and a draft agreement, **When** the
   sender changes the organization's signer name and saves, **Then** the
   draft's review step shows the new name.
5. **Given** the sender chooses Cancel, **When** they had unsaved edits,
   **Then** they return to the agreements list and the stored values are
   unchanged.

---

### User Story 2 - Sender fills the multi-step form and reaches review (Priority: P1)

A sender with saved organization info starts a new agreement. A wizard
titled "Create my agreement" walks them through eight named steps with a
progress bar: Agreement, Scope, Pricing & schedule, Environment & task mix,
Technical standards, Delivery & acceptance, Counterparty, Review. Every
field that varies in the source document has a home in one of these steps,
with the source document's values as the worked example in placeholders and
the document's standard values as defaults. Completing the first step
creates a draft. Each later Next saves that step. The Review step shows
everything entered as read-only grouped tables, with the sender's
organization pulled from settings.

**Why this priority**: this is the product's core object. The user's
instruction is to get the form right first and iterate on everything else.

**Independent Test**: recreate the Sieve and Pixels2 SOW from the source
document using only the form. Confirm every value in the source document
was entered somewhere, that no step accepted invalid input, and that the
review step shows every entered value.

**Acceptance Scenarios**:

1. **Given** the agreements list, **When** the sender chooses "New
   agreement", **Then** they see step 1 of 8, "Agreement", with the
   progress bar, the step name, the counter "1/8", a "Back to my
   agreements" link, and a Next button.
2. **Given** step 1 with a required field empty, **When** the sender
   presses Next, **Then** the field shows "Required" beneath it and the
   step does not advance.
3. **Given** step 1 fully valid, **When** the sender presses Next, **Then**
   a draft is created, the address bar changes to the draft's edit address
   with step 2, and the draft appears in the agreements list as Draft.
4. **Given** any step from 2 to 7 fully valid, **When** the sender presses
   Next, **Then** the step's values are saved and the next step opens.
5. **Given** any step from 2 to 8, **When** the sender presses Back or the
   previous-step arrow, **Then** the previous step opens with its saved
   values, and values typed on the current step but not yet saved remain
   available if the sender returns to it in the same session.
6. **Given** a numeric money field, **When** the sender types digits,
   **Then** the value displays with thousands separators and a currency
   prefix, and a value at or above the field's limit shows a message
   naming the limit.
7. **Given** the Environment & task mix step, **When** the easy, medium,
   and hard percentages do not sum to 100, **Then** the step shows the
   message "Must add up to 100%" and does not advance.
8. **Given** the Pricing & schedule step, **When** a milestone deadline is
   earlier than the effective date, or the final-delivery deadline is
   earlier than another milestone's deadline, **Then** the row shows a
   message and the step does not advance.
9. **Given** a repeating table (deliverables, milestones, verticals,
   technical requirements, manifest fields, site data fields), **When**
   the sender adds, edits, reorders, or removes a row, **Then** the change
   is reflected on the review step, and a table below its minimum row
   count shows a message and blocks Next.
10. **Given** the Review step, **When** the sender reads it, **Then** every
    field from steps 1 to 7 is shown under its step heading, the
    organization block is shown with the note "Organization details come
    from your saved organization info" and an "Edit organization info"
    link, and the computed maximum total equals unit fee times target
    volume.
11. **Given** the Review step, **When** the sender looks for the send
    action, **Then** the primary action reads "Send agreement & signing
    links" and is disabled with the note "Sending arrives in a later
    release". Back and "Back to my agreements" work.

---

### User Story 3 - Sender chooses how the developer is identified (Priority: P2)

The sender's organization is always the Customer, and the counterparty is
always the Developer. On the Counterparty step the sender picks between
"I have counterparty details" and "Invite by email only". With details, they enter the
counterparty's email, legal name, entity type and jurisdiction, address,
short name, signatory's legal name, signatory's title, and CC emails. With
invite only, they enter just the email and CC emails, and the form states
that the counterparty will provide their details when they sign.

**Why this priority**: the source document names both parties in full, but
the YC model shows senders often know only an email at drafting time. Both
paths must exist before sending can be built.

**Independent Test**: complete the Counterparty step in each mode and
confirm the review step shows the details entered, or the email with the
note that details will be collected at signing.

**Acceptance Scenarios**:

1. **Given** the Counterparty step, **When** it opens, **Then** "I have
   counterparty details" is preselected and the full field list is shown.
2. **Given** "Invite by email only" selected, **When** the sender looks at
   the step, **Then** only the counterparty email, CC emails, and the note
   "The counterparty will provide their name and details when they sign"
   are shown, and previously typed detail values are kept but hidden.
3. **Given** an email field containing text that is not an email address,
   **When** the sender presses Next, **Then** the field shows "Enter a
   valid email address" and the step does not advance.
4. **Given** the CC emails field containing several comma-separated
   addresses with one invalid, **When** the sender presses Next, **Then**
   the field names the invalid entry and the step does not advance.
---

### User Story 4 - Sender resumes, lists, and deletes drafts (Priority: P3)

The agreements list shows count tiles for Signed, Awaiting signature, and
Drafts, filter chips (All, Signed, Awaiting, Drafts), and a table with
Counterparty, Title, Status, and Date created columns. Opening a draft
returns the sender to the wizard at the first step that is not yet
complete. A draft can be deleted from the list.

**Why this priority**: drafts are useless if they cannot be reopened, and
the constitution requires a stated deletion path before party details are
stored. The list is the YC model's home page and the launch point for
"New agreement".

**Independent Test**: create two drafts stopping at different steps, sign
out and in, open each from the list, confirm each resumes at its next
incomplete step with values intact, then delete one and confirm it is gone.

**Acceptance Scenarios**:

1. **Given** a sender with drafts, **When** they open the agreements list,
   **Then** the Drafts tile shows the draft count, Signed and Awaiting
   show 0, and each draft row shows the counterparty (legal name over
   email, or the email alone), the SOW title, the status "Draft", and the
   creation date.
2. **Given** a draft row, **When** the sender opens it, **Then** the wizard
   opens at the first incomplete step with every saved value in place.
3. **Given** the list, **When** the sender sorts by a column header or
   filters by chip, **Then** the rows reorder or narrow accordingly.
4. **Given** a draft, **When** the sender chooses Delete and confirms,
   **Then** the draft and all its field values are removed and no longer
   listed.
5. **Given** another sender's draft address, **When** a signed-in sender
   opens it, **Then** they see a not-found page and none of its contents.

---

### Edge Cases

- The sender opens the wizard with no saved organization info: steps 1 to
  7 work, and the review step shows the organization block empty with the
  message "Add your organization info before sending" and the edit link.
- The sender reloads the browser mid-step before pressing Next: unsaved
  values on that step are lost, saved steps are intact, and the wizard
  reopens at the step in the address bar.
- Unit fee times target volume overflows the money limit: the maximum
  total shows the same limit message as a typed field.
- A repeating table row is left completely blank: it is dropped on Next.
- A row is partly filled: the empty required cells show "Required".
- Effective date in the past: allowed. The source document was dated four
  days before the final signature.
- The sender changes the unit label after milestones were entered: the
  milestone volume column relabels, values are kept.
- Two browser tabs edit the same draft: the last Next wins per step, and
  the review step shows what is stored.
- The counterparty email equals the sender's own signer email: allowed,
  since a sender may countersign their own agreement.

## Requirements *(mandatory)*

### Functional Requirements

**Organization info**

- **FR-001**: The system MUST provide an Organization info page reachable
  from the agreements list and from the review step's "Edit organization
  info" link, with the intro "Used for draft agreements when you review
  and send them. Agreements already sent keep their existing organization
  info."
- **FR-002**: Organization info MUST hold these fields, each with a label,
  a required marker where noted, a help tooltip, and an example
  placeholder: Organization legal name (required, e.g. "Pixels Two
  Corporation"); Entity type (required, e.g. "corporation", "LLC");
  Place of incorporation (required, a U.S. state or a country); Governing
  law jurisdiction (required, defaults to the place of incorporation);
  Organization address (required, multi-line); Short name used in the
  document (required, e.g. "Sieve"; defaults to the first word of the
  legal name); Signer name (required); Signer title (required); Signer
  email (required, valid email); Contact phone (optional).
- **FR-003**: Saving organization info MUST update the review step of
  every draft. Agreements that have been sent MUST keep a frozen copy of
  the organization info they were sent with. This feature ships no sent
  agreements, and the frozen copy is taken at send time by the sending
  feature.

**Wizard shell**

- **FR-004**: "New agreement" MUST open a wizard titled "Create my
  agreement" with eight steps in this order: Agreement, Scope, Pricing &
  schedule, Environment & task mix, Technical standards, Delivery &
  acceptance, Counterparty, Review.
- **FR-005**: Every step MUST show a progress bar with one segment per
  step, the current step's name, the counter "N/8", the text "Step N of
  8", a section heading, a one-line subtitle, a Back control, and a Next
  control. Step 1's Back and the previous-step arrow MUST lead to the
  agreements list.
- **FR-006**: Pressing Next MUST validate the current step. When any field
  fails, the step MUST stay open, each failing field MUST show its message
  beneath it, and the first failing field MUST be scrolled into view.
- **FR-007**: The first successful Next MUST create a draft owned by the
  signed-in sender and change the address to the draft's edit address
  with the step number. Every later successful Next MUST save that step's
  values. Opening a draft's edit address with a step number MUST open that
  step with the saved values.
- **FR-008**: Back MUST open the previous step without validating the
  current one. Values typed but not saved on the current step MUST remain
  available while the wizard stays open in the same session.
- **FR-009**: Every field MUST show a label, a required asterisk where the
  field is required, an information icon whose tooltip explains where the
  value appears in the document, and a placeholder holding the source
  document's value as an example. Fields with a standard value in the
  source document MUST be prefilled with that value as a default.
- **FR-010**: Money fields MUST show a currency prefix and format typed
  digits with thousands separators. Percentage fields MUST show a "%"
  suffix. Duration fields MUST show their unit ("days", "hours",
  "business days") as a suffix. Values above 10 trillion in money fields
  MUST show "Must be less than 10 trillion". Percentages MUST be between
  0 and 100.
- **FR-011**: Repeating tables MUST support adding a row, removing a row,
  reordering rows, and editing cells in place, with a minimum row count
  per table stated below.

**Step 1: Agreement**

- **FR-012**: Step 1 MUST hold: SOW number (required, integer, defaults to
  one more than the sender's highest existing SOW number, starting at 1);
  SOW title (required, e.g. "Pixels2 India Stereo Egocentric Pilot");
  Effective date (required, date); Exclusive engagement (yes/no, default
  yes; when yes the title suffix "(Exclusive)" and the exclusivity
  sentences render; when no they are omitted); Materials summary for the
  master agreement (required, default "recordings, stereo video, IMU
  data, calibration data, metadata, and other materials").
- **FR-013**: Step 1 MUST hold a collapsed group "Standard terms", opened
  on demand, holding the master agreement's negotiable numbers with the
  source document's values as defaults: Security incident notice window
  (24 hours); Breach cure period (30 days); Termination for convenience
  notice (30 days); Compliance records retention (2 years); Liability
  look-back period (12 months); Excluded-claims cap multiplier (2 times
  fees); Data and IP claims cap floor (US $50,000). All are required.

**Step 2: Scope**

- **FR-014**: Step 2 MUST hold: Deliverable description (required, e.g.
  "stereo RGB, head-mounted first-person egocentric footage");
  Collection country or region (required, e.g. "India"); Venue
  constraint (required, e.g. "real, non-residential operating
  businesses"); Unit of acceptance, singular and plural (required,
  default "accepted usable hour" / "accepted usable hours"); Target
  volume (required, positive number, e.g. 300); Deliverables (repeating
  list, minimum 1, default rows: original stereo video with camera, IMU,
  calibration, and metadata files; the delivery manifest; collection and
  consent records sufficient to verify chain of title; all other files
  specified in the SOW); Ambient audio permitted (yes/no, default no).

**Step 3: Pricing & schedule**

- **FR-015**: Step 3 MUST hold: Currency (required, default USD); Unit fee
  (required, money, e.g. 20.00); Target volume (read-only, from step 2);
  Maximum total (read-only, unit fee times target volume); Deposit or
  advance (yes/no, default no; when yes, an amount); Invoicing cadence
  (required, one of weekly, biweekly, monthly, per milestone; default
  weekly); Payment terms in days after receipt (required, default 30);
  Payment methods (required, default "ACH or wire transfer"); Invoice
  email (required, valid email, e.g. "ap@sievedata.com").
- **FR-016**: Step 3 MUST hold a Milestones table (minimum 1 row) with
  columns Milestone (text), Target volume (text or number in the unit),
  Deadline (date), and Notes (optional text), defaulting to two rows:
  "Rolling daily uploads / Completed data as available / [date]" and
  "Final delivery / [target volume] / [date]". One row MUST be marked as
  the final delivery, and its deadline MUST be the latest. A deadline
  before the effective date MUST be rejected with "Must be on or after the
  effective date".
- **FR-017**: Step 3 MUST hold Final deadline is firm (yes/no, default
  yes). When yes, the document renders the firm-obligation sentence, the
  reduce-or-terminate remedy, and the written-extension sentence.

**Step 4: Environment & task mix**

- **FR-018**: Step 4 MUST hold a Collection verticals table (minimum 1
  row) with columns Verticals (text), Priority and target (text), and
  Required or excluded details (text), defaulting to the source
  document's three rows.
- **FR-019**: Step 4 MUST hold: Maximum accepted units from one category
  (number, 60); Minimum distinct physical sites (integer, 15); Difficulty
  mix easy, medium, hard (three percentages that sum to 100; 20, 30, 50);
  Minimum share by skilled workers at their own workplace (percentage,
  70).
- **FR-020**: Step 4 MUST hold a Diversity caps group: per operator, task,
  and site (easy 1, medium 2, hard 10); per task and site across
  operators (easy 10, medium 20, hard 40); per distinct task across the
  project (easy 4.5, medium 12, hard 18); per operator at one site across
  all tasks (40); Repetitive motion limit per task and site (1); Repetitive
  motion limit where the workflow varies (2). All are numbers in the unit
  of acceptance.
- **FR-021**: Step 4 MUST hold: Collection plan required before filming
  (yes/no, default yes); Change implementation window (hours, default 24);
  Additional prohibited content (optional repeating list appended to the
  fixed prohibited list); Additional excluded settings (optional repeating
  list appended to the fixed excluded settings).

**Step 5: Technical standards**

- **FR-022**: Step 5 MUST hold a Technical requirements table (minimum 1
  row) with columns Requirement, Default specification, and Rejection
  criteria, defaulting to the source document's seven rows: Modality;
  Resolution and frame rate; Field of view and baseline; Synchronization
  and IMU; Codec and GOP; Hands and clips; Calibration.
- **FR-023**: Step 5 MUST hold Pre-collection materials (repeating list,
  minimum 0, default: device-specification sheet; representative sample
  footage; calibration files) and Every item is a condition of payment
  (yes/no, default yes).

**Step 6: Delivery & acceptance**

- **FR-024**: Step 6 MUST hold: Storage and transfer location (required
  text, default "Customer-designated cloud bucket; encrypted in transit
  and at rest; private access only."); Delivery cadence (required, default
  "daily"); Manifest format (required text, default the source document's
  manifest-format cell); Required manifest fields (repeating list, minimum
  1, default the source document's 21 fields); Site data fields (repeating
  list, minimum 0, default: business name; registered address; government
  business identifier; six-digit NAICS industry code); Integrity and
  security requirements (required text, default the source document's
  cell).
- **FR-025**: Step 6 MUST hold: Acceptance review window (days, 30);
  Correction or replacement window (business days, 5); Deemed acceptance
  when the review window lapses with a complete manifest (yes/no, default
  yes); Rejected material stays Developer-owned (yes/no, default yes);
  Deletion window for rejected material (days, 30, shown only when the
  previous answer is yes).

**Step 7: Counterparty**

- **FR-026**: Step 7 MUST hold Counterparty mode (radio cards, "I have
  counterparty details" or "Invite by email only", default details). The
  sender's organization is always the Customer and the counterparty is
  always the Developer.
- **FR-027**: In details mode, step 7 MUST hold: Counterparty email
  (required, valid email); Counterparty legal name (required); Entity type
  and jurisdiction (required, e.g. "Delaware corporation"); Counterparty
  address (required, multi-line); Short name used in the document
  (required, defaults to the first word of the legal name); Signatory's
  legal name (required); Signatory's title (optional); CC emails
  (optional, comma-separated valid emails).
- **FR-028**: In invite-only mode, step 7 MUST hold only Counterparty
  email (required) and CC emails, with the note "The counterparty will
  provide their name and details when they sign." Detail values entered
  earlier MUST be kept and hidden.

**Step 8: Review**

- **FR-029**: The Review step MUST show, under the subtitle "Confirm
  everything looks right before sending the signing links.", read-only
  grouped tables for every field from steps 1 to 7, an organization block
  drawn from saved organization info with the note "Organization details
  come from your saved organization info." and an "Edit organization
  info" link, and a counterparty block, labelled Customer and Developer.
- **FR-030**: The Review step MUST show a primary action "Send agreement &
  signing links" that is disabled in this feature with the note "Sending
  arrives in a later release". Each review group MUST link back to its
  step.

**Agreements list**

- **FR-031**: The agreements list MUST show tiles for Signed ("Fully
  executed"), Awaiting signature ("In-progress"), and Drafts ("Not yet
  sent") with counts, filter chips All, Signed, Awaiting, Drafts, a "New
  agreement" button, and a table with sortable columns Counterparty, Title,
  Status, and Date created. Counterparty shows the legal name over the
  email, or the email alone in invite-only mode, or "No counterparty yet"
  before step 7 is saved.
- **FR-032**: Opening a draft row MUST open the wizard at the first step
  whose values have not been saved, or the Review step when all seven are
  saved.
- **FR-033**: A draft MUST be deletable from the list after a confirmation
  that names the draft. Deletion removes the draft and every stored value
  permanently.

**Access, retention, and privacy**

- **FR-034**: Every draft and organization record MUST be visible and
  editable only to the sender who owns it. Any other sender's request for
  it MUST be answered as not found.
- **FR-035**: Drafts and organization info MUST be retained until the
  sender deletes them. Counterparty emails, names, and addresses MUST
  never appear in logs, metrics, or error messages.
- **FR-036**: The document template is fixed per version. This feature
  MUST record the template version on every draft so the renderer built
  later knows which wording the values belong to.

### Key Entities

- **Organization info**: one per sender. Legal name, entity type, place
  of incorporation, governing law jurisdiction, address, short name,
  signer name, signer title, signer email, contact phone.
- **Agreement**: owned by a sender. Status (Draft in this feature),
  template version, creation date, the step-completion record, and one
  value set per step.
- **Agreement terms**: SOW number, title, effective date, exclusivity,
  materials summary, and the seven standard-terms numbers.
- **Scope**: deliverable description, country or region, venue
  constraint, unit labels, target volume, ambient-audio flag, and the
  Deliverable items.
- **Pricing and schedule**: currency, unit fee, deposit, invoicing
  cadence, payment terms, payment methods, invoice email, firm-deadline
  flag, and the Milestones (name, volume, deadline, notes, final flag).
- **Environment and task mix**: the Vertical rows, category and site
  minimums, difficulty mix, skilled share, diversity caps, repetitive
  motion limits, plan and change-window settings, extra prohibited
  content, extra excluded settings.
- **Technical standards**: the Technical requirement rows and the
  pre-collection materials.
- **Delivery and acceptance**: storage location, cadence, manifest format,
  manifest fields, site data fields, integrity text, review and correction
  windows, deemed-acceptance flag, rejected-material flags.
- **Counterparty**: mode, email, legal name, entity type and
  jurisdiction, address, short name, signatory name, signatory title, CC
  emails.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A tester holding the source document can enter every value
  from it into the form without a single value lacking a field, verified
  by a checklist that enumerates every variable value in the document.
- **SC-002**: With defaults accepted and only deal-specific values typed,
  a sender reaches the review step for a deal like the source document in
  under 15 minutes.
- **SC-003**: Every value entered on steps 1 to 7 appears on the review
  step, verified field by field with zero omissions.
- **SC-004**: 100% of invalid inputs in the validation list (missing
  required fields, malformed emails, percentages not summing to 100,
  deadlines before the effective date, money above the limit) are blocked
  with a message beneath the field and never advance the step.
- **SC-005**: A draft interrupted at any step resumes with 100% of saved
  values intact after sign-out and sign-in.
- **SC-006**: A sender who saves organization info once never re-enters
  any of its ten fields on any later agreement.
- **SC-007**: No sender can read or change another sender's draft or
  organization info, verified by attempting every draft address and
  action with a second account.

## Assumptions

- The sender's organization is always the Customer and the counterparty
  is always the Developer, matching the YC model where the company always
  sends to the investor. The template's terms are written from the
  customer's side, so a developer would never send it as drafted. A
  vendor-authored SOW would be a separate template version.
- "Extract everything" includes the master agreement's negotiable numbers.
  They sit in a collapsed group with the source document's values as
  defaults, so the default path stays short.
- The form is the deliverable. Rendering the PDF, sending, and signing are
  later features. The review step keeps the YC layout with the send action
  disabled so the next feature enables it without redesign.
- The counterparty signature block is Name and Title only, as in the
  source document. The YC "signature block" free-text field is not
  reproduced.
- Organization info supports a U.S. state or a country for place of
  incorporation because the source document's parties are Delaware
  corporations while collection happens in India.
- Fixed template text includes every sentence of the master agreement
  other than the values in FR-012 and FR-013, and every SOW sentence other
  than the values in FR-014 to FR-028. Free-form clause editing stays out
  of scope per the project context.
- Delete is offered for drafts only. Sent and signed agreements are
  immutable and are handled by later features.
- The agreements list replaces the placeholder dashboard from feature 001
  as the signed-in home page.
- The source document's two defects are corrected in the template: the
  master agreement's deliverable description comes from the materials
  summary field rather than hard-coded video terms, and the SOW carries
  its own effective date rather than borrowing the master's.

## Reference: how the YC SAFE creator is modelled

Observed on 2026-09-03 in the sender's signed-in account.

- **Home** ("My SAFEs"): subtitle "Every SAFE you've sent and its current
  signature status." Three count tiles (Signed / Fully executed, Awaiting
  signature / In-progress, Drafts / Not yet sent). Filter chips All,
  Signed, Awaiting, Drafts. Buttons "Use with an agent" and "New SAFE".
  Table columns Investor (legal name over email), Type, Status (Draft,
  Awaiting investor signature), Date created, each sortable. A draft row
  links to its edit address. A sent row links to a view address.
- **Company info** (settings): intro "Used for draft SAFEs when you review
  and send them. SAFEs already sent keep their existing company info."
  Fields: Company legal name*, State of incorporation* (dropdown),
  Governing law jurisdiction* (dropdown), Company address* (multi-line),
  Signer name*, Signer title*, Signer email*, Contact phone. Cancel and
  Save changes. Every label has an info tooltip and an example
  placeholder.
- **Wizard** ("Create my SAFE"): three steps, Terms, Investor, Review.
  Header shows a previous-step arrow, the title, the step name, "1/3",
  "Step 1 of 3", and a three-segment progress bar. Footer shows Back and
  Next. Step 1's Back is "Back to My SAFEs".
- **Step 1, Terms** ("Choose your SAFE", "Pick a SAFE type and set the
  financial terms."): three radio cards with a title and one-line
  description. Choosing one collapses the others into a dropdown on the
  selected card. Conditional fields per type: Valuation Cap shows
  Investment Amount* and Valuation Cap* (money inputs with "$" prefix and
  live thousands separators) and "Include Pro Rata Side Letter"
  (checkbox); Discount shows Investment Amount* and Discount* ("%"
  suffix); Uncapped MFN shows Investment Amount* and the side-letter
  checkbox. Pressing Next with an empty required field shows "Required"
  beneath it. An oversized value shows "Must be less than $10 trillion".
  The first successful Next creates the draft and moves the address to
  /tools/safe/{id}/edit?step=2.
- **Step 2, Investor** ("Investor information", "Where to send the signing
  request."): two radio cards, "I have investor details" and "Invite by
  email only". Details mode fields: Investor email* ("We'll send the
  signing link to this address."), Investor or fund legal name*,
  Investor's signature block (multi-line, for entity signers), Signatory's
  legal name*, Signatory's title, Investor's address (multi-line, "used in
  the notice section"), CC emails ("These addresses receive a copy of the
  signed SAFE once it's complete. Separate multiple with commas."). Invite
  mode shows only Investor email*, the note "The investor will provide
  their name and details when they sign.", and CC emails.
- **Step 3, Review** ("Review & send", "Confirm everything looks right
  before sending the signing links."): read-only two-column tables
  grouped as SAFE Terms, Company, Company Signer, Investor. Below the
  company group: "Company details come from your saved company info. Edit
  company info". Footer: Back and "Send SAFE & signing links".
- **Sent view**: SAFE Terms table, then Signing Status listing each signer
  with role, email, status Pending, and a Copy Link button. "View my
  SAFEs" button.
