# Tasks: Render Agreement PDF

**Input**: Design documents from `/specs/003-render-agreement-pdf/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Included. The constitution mandates TDD: every test task is written and failing before the implementation task that follows it. Integration tests mount the real routes and store with no module mocks. Browser APIs that jsdom lacks (`URL.createObjectURL`, `URL.revokeObjectURL`) are stubbed in the test setup, which is not application code.

**Organization**: Phase 1 adds dependencies and the fixture. Phase 2 builds the pure helpers every story uses. Phases 3 to 5 map to the spec's user stories in priority order. Phase 6 is the gate and the browser walk. No GitHub issues are created for this feature (user instruction, 2026-09-04).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1 to US3)
- Paths are repository-relative. The source wording is in `specs/003-render-agreement-pdf/source/source-document.txt` (gitignored, present in this worktree).

## Path Conventions

- Web: `web/` (Vite + React SPA). Feature code under `web/src/features/agreements/`.
- Document code under `web/src/features/agreements/document/`.
- Template under `web/src/features/agreements/document/templates/content-development-1/`.

---

## Phase 1: Setup

**Purpose**: Dependencies, the CSP change, and the fixture every test uses.

- [X] T001 Add `pdfmake@0.3.11` and `@types/pdfmake@0.3.3` to `dependencies` and `pdf-parse@2.4.5` to `devDependencies` in `web/package.json`; run `npm install` in `web/` and commit the lockfile change
- [X] T002 [P] Add `frame-src blob:` to the Content-Security-Policy line in `web/public/_headers` (keep `object-src 'none'`); note the addition in `docs/deployment.md` under the Web section's security-headers sentence
- [X] T003 [P] Create `web/src/test/factories/source-document.ts` exporting `sourceOrganization(): Organization` (Sieve Inc., corporation, Delaware, Delaware, "251 Little Falls Drive, Wilmington, New Castle County, Delaware 19808" is the Developer's address so use "100 Market St, San Francisco, CA 94105" for the Customer, short name Sieve, Ishan Dhawan, Chief of Staff, ishan@sieve.example) and `sourceDraft(overrides?: Partial<DraftSteps>): Draft` with every step matching the source document: agreement (SOW 1, "Pixels2 India Stereo Egocentric Pilot", 2026-08-29, exclusive, defaults), scope (`defaultScope` with description "stereo RGB, head-mounted first-person egocentric footage", regions `['IN']`, venue "real, non-residential operating businesses", volume 300), pricing (`defaultPricing(300, unit)` with unitFee 20, invoiceEmail ap@sievedata.com, milestones dated 2026-09-06 with notes "Daily after collection begins" and 2026-09-13 final), environment `defaultEnvironment(unit)`, technical `defaultTechnical`, delivery `defaultDelivery`, counterparty details mode (Pixels Two Corporation, "Delaware corporation", the Wilmington address above, akshaj@pixels2.example, Akshaj Jain, President); also export `seedSourceDocument()` that writes both into `useAgreementsStore` and returns the draft id
- [X] T004 [P] Extend `web/src/test/setup.ts` with `URL.createObjectURL` and `URL.revokeObjectURL` stubs (counting calls on `globalThis.__objectUrls` so tests can assert creation and revocation) installed in `beforeAll` and reset in `afterEach`

---

## Phase 2: Foundational

**Purpose**: Pure helpers used by every story. Each test file is written first and fails before its implementation.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [X] T005 [P] Add failing tests to `web/src/lib/dates.test.ts` for `formatLongDate`: `'2026-09-06'` → `'September 6, 2026'`, `'2026-01-01'` → `'January 1, 2026'`, an empty string → `''`, and a value parsed as UTC (no day shift when `TZ` is west of UTC)
- [X] T006 Implement `formatLongDate(isoDate: string): string` in `web/src/lib/dates.ts` using `Intl.DateTimeFormat('en-US', { dateStyle: 'long', timeZone: 'UTC' })` on `new Date(\`${isoDate}T00:00:00Z\`)`
- [X] T007 [P] Create failing `web/src/features/agreements/document/format.test.ts` covering `formatDocumentMoney(20, 'USD')` → `'US $20.00'`, `formatDocumentMoney(6000, 'USD', { wholeAsInteger: true })` → `'US $6,000'`, `formatDocumentMoney(1234.5, 'EUR')` → `'EUR 1,234.50'`, `digits(4.5)` → `'4.5'`, `digits(300)` → `'300'`, `unitForms('accepted usable hours')` → `{ plural: 'accepted usable hours', singular: 'accepted usable hour', shortPlural: 'hours', shortSingular: 'hour' }`, `quantity(1, forms)` → `'1 accepted usable hour'`, `quantity(300, forms, 'short')` → `'300 hours'`, `series(['a'])`, `series(['a','b'])` → `'a and b'`, `series(['a','b','c'])` → `'a, b, and c'`, `lettered(['x','y','z'])` → `'(a) x; (b) y; and (c) z'`, `lowerFirst('Daily')` → `'daily'`, `countryList(['IN'])` → `'India'`, `countryList(['IN','NP'])` → `'India and Nepal'`
- [X] T008 Implement `web/src/features/agreements/document/format.ts` with those functions (`quantity` accepts the count as number or numeric string and falls back to the text as typed when it is not numeric); reuse `singularize` from `../units` and `countryName` from `../countries`
- [X] T009 [P] Create failing `web/src/features/agreements/readiness.test.ts`: a draft with only step 1 and no organization yields `ready: false` and `missing` in order Scope, Pricing & schedule, Environment & task mix, Technical standards, Delivery & acceptance, Developer, Organization info with `to` values `/agreements/{id}/edit?step=N` and `/organization`; the fixture with organization yields `ready: true` and an empty list; the fixture without organization yields only Organization info
- [X] T010 Implement `documentReadiness(draft: Draft, organization: Organization | null): DocumentReadiness` in `web/src/features/agreements/readiness.ts` using `WIZARD_STEPS` for labels and the `DraftSteps` key order
- [X] T011 [P] Create failing `web/src/features/agreements/document/filename.test.ts`: `documentFilename(1, 'Pixels2 India Stereo Egocentric Pilot')` → `'SOW 1 - Pixels2 India Stereo Egocentric Pilot.pdf'`; reserved characters `/ \ : * ? " < > |` and control characters become spaces and runs collapse; a 200-character title is cut to 120
- [X] T012 Implement `web/src/features/agreements/document/filename.ts`
- [X] T013 [P] Create failing `web/src/features/agreements/document/text-of.test.ts`: `textOf` flattens a string, `{ text }`, nested `text` arrays, `{ stack }`, `{ columns }`, `{ ul }`, `{ ol }`, and `{ table: { body } }` cells into one string with single spaces between leaves and a newline between table rows; `headingsOf(content)` returns the `text` of every node with `style === 'h1'` or `style === 'h2'` in order
- [X] T014 Implement `web/src/features/agreements/document/text-of.ts` (typed against `Content` from `pdfmake/interfaces`)
- [X] T015 [P] Create failing `web/src/features/agreements/document/model.test.ts`: from `sourceDraft()` and `sourceOrganization()` the model carries `effectiveDate 'August 29, 2026'`, `unitFee 'US $20.00'`, `maximumTotal 'US $6,000'`, `targetVolume '300'`, unit forms, `regions 'India'`, `firstDeadline 'September 6, 2026'`, `finalDeadline 'September 13, 2026'`, milestone `deadlineText` `'Daily after collection begins, no later than September 6, 2026'` and `'September 13, 2026'`, `dataClaimsCapFloor 'US $50,000'`, `developer.known true`; in invite mode `developer.known false` and each detail is its bracketed blank (`'[Developer legal name]'`, `'[Developer entity type and jurisdiction]'`, `'[Developer address]'`, `'[Developer signatory name]'`, `'[Developer signatory title]'`); `delivery.cadence 'daily'`; `currencyName 'U.S. dollars'` for USD and `'EUR'` for EUR; `depositAmount` formatted when required; an incomplete draft throws `IncompleteDraftError`; a null organization throws `IncompleteDraftError`
- [X] T016 Implement `web/src/features/agreements/document/model.ts` exporting `DocumentModel`, `IncompleteDraftError`, and `buildDocumentModel(draft, organization)` per data-model.md (the model carries `environment.limits` rows, `environment.difficultyCaps` rows, and the invoicing cadence label as stored; wording for those lists is applied by the template)

**Checkpoint**: `npm run test -- format dates readiness filename text-of model` green.

---

## Phase 3: User Story 1 - Sender generates and reads the document (Priority: P1) 🎯 MVP

**Goal**: A ready draft renders the full agreement in the browser and shows it on the document page with page count, download, and back links.

**Independent Test**: seed the fixture, open the Review step, press "Generate PDF", and compare the rendered PDF to `source/source-document.pdf` page by page. Every value in its place, every fixed sentence verbatim, sections in the same order.

### Tests for User Story 1

- [X] T017 [P] [US1] Create failing `web/src/features/agreements/document/templates/content-development-1/master-agreement.test.ts`: `headingsOf(masterAgreement(model))` equals the 12 section titles in order ("1. Services." to "12. General."), the flattened text starts with "CONTENT DEVELOPMENT AGREEMENT", the preamble contains the fixture's effective date, both legal names, "a corporation of Delaware", both addresses, "(“Sieve”)" and "(“Developer”)"; the text contains "Sieve and Developer are each a “Party”", the materials summary in 1.1, "at least 2 years after final delivery" in 5.4, "within 24 hours" in 7.1, "30 days’ written notice" twice in 8.1, "12 months preceding" and "2 times the fees" and "US $50,000" in 11.1, "governed by Delaware law" and "courts located in Delaware" in 12.4; the text contains no "Sieve Inc.:" and the signature block text contains "IN WITNESS WHEREOF", "Sieve Inc.", "Pixels Two Corporation", "Name: Ishan Dhawan", "Title: Chief of Staff", "Name: Akshaj Jain", "Title: President", and two "Signature:" and two "Date:" labels; with `customer.shortName` "Acme" the text contains "Acme's" wherever the source has "Sieve's" and never contains "Sieve" outside the legal name
- [X] T018 [P] [US1] Create failing `web/src/features/agreements/document/templates/content-development-1/statement-of-work.test.ts` (US1 cases): `headingsOf` yields "EXHIBIT A", the title "SOW 1 — PIXELS2 INDIA STEREO EGOCENTRIC PILOT (EXCLUSIVE)", then the 10 section titles in order; the Overview text contains "deliver to Sieve 300 accepted usable hours of exclusive, stereo RGB, head-mounted first-person egocentric footage captured in real, non-residential operating businesses in India"; Deliverables contains "(a) Original stereo video" and "(d) All other files specified in this SOW"; the Pricing table body has the header row Unit Fee, Target Volume, Total, Payment Basis and a data row "US $20.00 per accepted usable hour", "300 accepted usable hours", "Maximum US $6,000", "Accepted usable hours only; weekly invoicing in arrears; Net 30"; the Delivery Schedule table has header Delivery Milestone, Target Volume, Deadline and rows for both milestones with the `deadlineText` values; the Environment table has header Collection Verticals, Target Hours or Percentage, Excluded or Required Details and three rows; the limits paragraph contains "No more than 60 accepted usable hours may come from a single category." and "At least 15 distinct physical sites are required." and "The target task-difficulty mix is 20% easy, 30% medium, and 50% hard; at least 70% of accepted usable hours must be performed by skilled workers"; the caps paragraph contains "Binding diversity caps: per operator, task, and site: easy 1 hour, medium 2 hours, hard 10 hours;" and "and per operator at one site across all tasks: 40 hours." and "Repetitive motion is limited to 1 hour per task/site, or 2 hours only where"; the Technical table has header Requirement, Default Specification, Rejection Criteria and seven rows starting with Modality; the Delivery / Manifest table has header Delivery / Manifest Field, Required Value and rows Storage / transfer location, Manifest format, Required fields (a comma series of the 21 fields), Site data ("… per site, which Sieve may assign."), Integrity and security; Acceptance contains "30 days after receipt", "within 5 business days", and "deemed accepted only if"; Invoicing contains "invoice weekly in arrears for accepted usable hours", "Net 30 days after receipt", "in U.S. dollars by ACH or wire transfer", "ap@sievedata.com"; the SOW ends with a second signature block
- [X] T019 [P] [US1] Create failing `web/src/features/agreements/document/doc-definition.test.ts`: `buildDocDefinition(model)` has `pageSize 'LETTER'`, `pageMargins [54, 54, 54, 60]`, `defaultStyle.font 'Helvetica'`, `watermark.text 'DRAFT'`, `footer(2, 9)` text `'SOW 1 · DRAFT · Page 2 of 9'`, the first content node text "CONTENT DEVELOPMENT AGREEMENT", and an "EXHIBIT A" node with `pageBreak 'before'`; every `table` node has `headerRows 1` and `dontBreakRows true`; `templateFor('content-development-1')` returns a function and `templateFor('nope')` throws `UnknownTemplateError`
- [X] T020 [P] [US1] Create failing `web/src/features/agreements/document/render.test.ts` with `// @vitest-environment node` at the top: `renderDocument(buildDocumentModel(sourceDraft(), sourceOrganization()))` resolves to a `Blob` of type `application/pdf`, `pageCount` between 8 and 11, and the same number of pages reported by `pdf-parse` (`new PDFParse({ data: Buffer.from(await blob.arrayBuffer()) }).getText()`); page 1 text contains "CONTENT DEVELOPMENT AGREEMENT", "August 29, 2026", "Page 1 of", and "DRAFT"; some later page starts with "EXHIBIT A"; the joined text contains these fixed sentences verbatim: "Developer is responsible for all Personnel and subcontractors as if it performed the Services itself.", "Only Deliverables accepted by Sieve are billable.", "No footage is deemed accepted merely because Sieve has not completed review.", "Sieve pays only for Deliverables that are received, accepted, and usable under this SOW.", and "Missing, inaccurate, or inconsistent required metadata is non-conforming and may be rejected."; the "Default Specification" header appears at least once and each "Requirement" row name appears once
- [X] T021 [US1] Create failing `web/src/features/agreements/document.integration.test.tsx` (US1 cases) using `renderApp` and `seedSourceDocument()`: the Review step shows an enabled "Generate PDF" button beside the disabled send button; a draft with steps 1 and 2 only and no organization shows "Generate PDF" disabled with the note "Complete Pricing & schedule, Environment & task mix, Technical standards, Delivery & acceptance, Developer, and add your organization info to generate the PDF." whose links point at `?step=3` … `?step=7` and `/organization`; pressing "Generate PDF" on the ready draft navigates to `/agreements/{id}/document`, shows `role="status"` with "Generating your PDF…", then the heading "Your agreement", "SOW 1 — Pixels2 India Stereo Egocentric Pilot", an `iframe` titled "Agreement PDF" whose `src` starts with `blob:`, text matching `/\d+ pages/` and `/1 of \d+/`, a link "Download PDF" with `download="SOW 1 - Pixels2 India Stereo Egocentric Pilot.pdf"` and the same href as the frame, "Draft preview. Signing arrives in a later release.", "Back to review" → `/agreements/{id}/edit?step=8`, "Back to my agreements" → `/agreements`; `/agreements/unknown/document` shows the not-found page; a ready draft whose `templateVersion` is `'other-1'` shows "This draft uses a template this version cannot render." with no "Try again"; when `URL.createObjectURL` throws, the page shows "Couldn't generate the PDF. Try again." and pressing "Try again" after the stub is restored shows the frame

### Implementation for User Story 1

- [X] T022 [P] [US1] Create `web/src/features/agreements/document/templates/content-development-1/phrases.ts` with the limit-label, invoicing-cadence, and currency-name lookups from `contracts/document-template.md`, each with the documented fallback, plus `limitSentences(rows, unitPlural)`, `capsSentence(rows, shortForms, operatorLimitRow)`, and `repetitiveMotionSentence(rows)` helpers
- [X] T023 [P] [US1] Create `web/src/features/agreements/document/templates/content-development-1/signature-block.ts` exporting `signatureBlock(model): Content` as the witness sentence plus a two-column table with no borders
- [X] T024 [P] [US1] Create `web/src/features/agreements/document/templates/content-development-1/styles.ts` with the named styles `title`, `h1`, `h2`, `body`, `tableHeader`, `footer` and the shared table layout (light grid, 4 pt padding)
- [X] T025 [US1] Create `web/src/features/agreements/document/templates/content-development-1/master-agreement.ts` exporting `masterAgreement(model): Content[]`: transcribe the preamble and sections 1 to 12 from `source/source-document.txt` verbatim, replacing "Sieve" and "Sieve's" with the customer short name, the deliverable phrase in 1.1 with `materialsSummary`, "two years" / "24 hours" / "30 days'" / "12 months" / "two times" / "US $50,000" / "Delaware" with the model values, and ending with `signatureBlock(model)`; one function per section, each returning `Content[]`
- [X] T026 [US1] Create `web/src/features/agreements/document/templates/content-development-1/statement-of-work.ts` exporting `statementOfWork(model): Content[]`: "EXHIBIT A" with `pageBreak: 'before'`, the title line, the preamble, and sections 1 to 10 transcribed verbatim with the model values and the five tables (widths per the contract), the Deliverables lettered list, the limits, mix, and caps paragraphs from `phrases.ts`, and `signatureBlock(model)` last; one function per section; the yes path of every conditional rule renders in this task and the no path is added in US2
- [X] T027 [US1] Create `web/src/features/agreements/document/templates/content-development-1/index.ts` exporting `contentDevelopment1(model): Content[]` as master then SOW, and `web/src/features/agreements/document/registry.ts` exporting `templateFor(version)` and `UnknownTemplateError`
- [X] T028 [US1] Create `web/src/features/agreements/document/doc-definition.ts` exporting `buildDocDefinition(model): TDocumentDefinitions` with the page setup, styles, footer, and watermark from the contract; the footer callback also reports the page count through an optional `onPageCount` argument
- [X] T029 [US1] Create `web/src/features/agreements/document/render.ts` exporting `renderDocument(model): Promise<RenderedDocument>`: dynamic `import('pdfmake/build/pdfmake')` and `import('pdfmake/build/standard-fonts/Helvetica')`, `addFonts` once, `createPdf(definition).getBlob()`, page count captured from the footer callback; add `web/src/types/pdfmake-standard-fonts.d.ts` declaring the `pdfmake/build/standard-fonts/Helvetica` module as `TFontDictionary`
- [X] T030 [US1] Create `web/src/features/agreements/pages/agreement-document-page.tsx` per `contracts/ui-routes.md`: reads the draft and organization from the store, redirects unknown ids to `/agreements/not-found` and unready drafts to step 8, generates in an effect on mount, holds `{ status, rendered, url, error }` in component state, revokes the object URL in cleanup, and renders the four states with the exact messages
- [X] T031 [US1] Register the lazy route `/agreements/:id/document` under `ProtectedLayout` in `web/src/app/router.tsx` (wrap in `Suspense` with `FullPageLoader`) and export `AgreementDocumentPage` from `web/src/features/agreements/index.ts`
- [X] T032 [US1] Add the "Generate PDF" action and the missing-items note to `web/src/features/agreements/components/steps/step-review.tsx` (a new `extraAction` slot in `web/src/features/agreements/components/wizard-shell.tsx` rendered beside the submit button); the note joins items with `series` and links each; the button navigates to the document route

**Checkpoint**: `npm run test -- document format model master statement doc-definition render` green. US1 is demonstrable: seed, review, generate, read.

---

## Phase 4: User Story 2 - Document wording follows the form's choices (Priority: P1)

**Goal**: Every conditional rule renders correctly in both settings, extra list items append, the plan subsection renumbers, and invite-only mode renders bracketed blanks.

**Independent Test**: for each of the nine rules, build the model with yes and with no and assert the governed sentences present or absent on the content tree and on the real PDF.

### Tests for User Story 2

- [X] T033 [P] [US2] Add failing cases to `web/src/features/agreements/document/templates/content-development-1/statement-of-work.test.ts`: exclusive false → title has no "(EXCLUSIVE)", Overview has "hours of stereo RGB" and no "The engagement and all Deliverables are exclusive."; firmDeadline false → no "firm obligation", no "reduce or reallocate", no "Any extension requires"; depositRequired true with 1000 → "Sieve will pay a deposit of US $1,000.00 against the fees above." and no "There is no deposit"; ambientAudio true → "Ambient audio is permitted." and no "No ambient audio"; planRequired false → no "Collection Plan", headings "6.1. Natural Activity." and "6.2. Privacy and Sensitive Content."; conditionOfPayment false → "Every technical item is a condition of acceptance." and no "and payment"; deemedAcceptance false → no "deemed accepted only if"; rejectedStaysDeveloperOwned false → Overview ends "subject to Section 4 of the Agreement." with no "carveout", and Acceptance has no "Notwithstanding Section 4" and no "delete its copies"; rejectedStaysDeveloperOwned true with deletionWindowDays 45 → "within 45 days after rejection"; extraProhibited `['weapons']` → the 6.3 list ends "…restricted/sensitive facilities, or weapons may be captured or delivered."; extraExcludedSettings `['outdoor markets']` → the 6.2 list contains "outdoor markets, or material previously captured"
- [X] T034 [P] [US2] Add failing cases to `master-agreement.test.ts`: invite mode → preamble contains "[Developer legal name], a [Developer entity type and jurisdiction], with an address at [Developer address]" and the signature block contains "Name: [Developer signatory name]" and "Title: [Developer signatory title]"; running text still says "Developer" and never contains a bracket outside those positions
- [X] T035 [P] [US2] Add failing cases to `web/src/features/agreements/document/render.test.ts`: a fixture with exclusive false, firmDeadline false, planRequired false, deemedAcceptance false, and rejectedStaysDeveloperOwned false renders a PDF whose text lacks "(EXCLUSIVE)", "firm obligation", "Collection Plan", "deemed accepted", and "Notwithstanding Section 4", and whose page count is still between 8 and 11

### Implementation for User Story 2

- [X] T036 [US2] Implement the no path of every rule and the list appends in `statement-of-work.ts` (each rule in its section function; the Collection Requirements section numbers its subsections from a running counter)
- [X] T037 [US2] Implement the invite-mode blanks in `model.ts` (already specified in T016) and confirm `master-agreement.ts` and `signature-block.ts` read only model fields, then run the three test files green

**Checkpoint**: US1 and US2 tests green. Both settings of every rule verified on the tree and on the PDF.

---

## Phase 5: User Story 3 - Sender regenerates and returns to the document (Priority: P2)

**Goal**: Regeneration reflects current values with no stale copy, the list offers "View PDF" only for ready drafts, and Download saves the named file.

**Independent Test**: generate, change the unit fee in the store, open the document again, and confirm a new object URL was created and the old one revoked; open the list and confirm "View PDF" appears only on the ready row.

### Tests for User Story 3

- [X] T038 [P] [US3] Add failing cases to `web/src/features/agreements/document.integration.test.tsx`: the list shows "View PDF" (→ `/agreements/{id}/document`) on the fixture row and not on a step-1-only row; opening the document page, navigating to "Back to review", changing the unit fee via `saveStep`, and pressing "Generate PDF" again creates a second object URL and revokes the first (assert on the setup counters); leaving the page revokes its URL; the "Download PDF" link carries the `download` attribute and the frame's href

### Implementation for User Story 3

- [X] T039 [US3] Add the "View PDF" link to the actions cell in `web/src/features/agreements/pages/agreements-list-page.tsx`, shown only when `documentReadiness(draft, organization).ready`
- [X] T040 [US3] Confirm `agreement-document-page.tsx` regenerates on every mount and revokes on cleanup and on replacement (adjust if T030 left either out); run the integration suite green
- [X] T041 [P] [US3] Create `web/src/app/dev-seed.ts` that, when `import.meta.env.DEV`, assigns `window.__sowSeedSourceDocument = seedSourceDocument` (importing from the test factory is not allowed in app code, so move the fixture values to `web/src/features/agreements/fixtures/source-document.ts` and re-export them from the test factory); call it from `web/src/main.tsx`

**Checkpoint**: All three stories green.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [X] T042 Run the full gate from `web/`: `npm run typecheck && npm run lint && npm run format:check && npm run test:coverage && npm run build`; confirm coverage stays at or above 85 percent on the new files and that the pdfmake chunk is separate in the build output
- [ ] T043 (blocked on a sign-in to the preview browser; the exported PDF was reviewed page by page instead, see plan.md Implementation Notes) Browser walk per `specs/003-render-agreement-pdf/quickstart.md` against the dev server with a real sign-in: seed, review, generate, read page 1 and the Exhibit A page, download, regenerate non-exclusive, regenerate invite-only; take screenshots of steps 4 and 5; confirm no party data in the console
- [X] T044 [P] Record any deviation from the plan in an "Implementation Notes" section at the end of `specs/003-render-agreement-pdf/plan.md`, and update `specs/003-render-agreement-pdf/data-model.md` if the model's fields changed
- [X] T045 Update the memory note for the project with the feature 003 state (branch, what shipped, the browser-only decision)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: T001 first (installs the packages the types need); T002 to T004 in parallel.
- **Foundational (Phase 2)**: after Setup. Each test task precedes its implementation task. The five pairs are independent of each other.
- **US1 (Phase 3)**: after Foundational. Tests T017 to T021 in parallel, then T022 to T024 in parallel, then T025 → T026 → T027 → T028 → T029 → T030 → T031 → T032.
- **US2 (Phase 4)**: after US1 (edits the same template files). T033 to T035 in parallel, then T036, T037.
- **US3 (Phase 5)**: after US1. T038 then T039, T040; T041 in parallel with them.
- **Polish (Phase 6)**: after all stories. T042 before T043.

### Parallel Opportunities

- Phase 2: T005, T007, T009, T011, T013, T015 together; then their implementations together.
- Phase 3: T017, T018, T019, T020, T021 together; T022, T023, T024 together.
- Phase 4: T033, T034, T035 together.

---

## Parallel Example: User Story 1

```bash
# Tests first, all at once:
Task: "master-agreement.test.ts"
Task: "statement-of-work.test.ts"
Task: "doc-definition.test.ts"
Task: "render.test.ts"
Task: "document.integration.test.tsx"

# Then the leaf modules:
Task: "phrases.ts"
Task: "signature-block.ts"
Task: "styles.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Phases 1 and 2.
2. Phase 3. Stop and validate: the fixture renders and reads like the source document.

### Incremental Delivery

1. Add US2: both settings of every rule.
2. Add US3: list action, regeneration, download, dev seed.
3. Gate and browser walk.

---

## Notes

- Transcribe the fixed wording from `source/source-document.txt`. Keep the curly quotes, the em dash in the SOW title, and the degree signs as they are.
- Comments follow the constitution's comment voice. No references to spec ids in code.
- Complexity per function stays at or below 10. The template files are long but each section is its own small function.
