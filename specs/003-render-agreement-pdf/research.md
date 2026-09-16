# Research: Render Agreement PDF

**Date**: 2026-09-04 · **Feature**: 003-render-agreement-pdf

No NEEDS CLARIFICATION markers remained in the spec. The decisions below
cover the choices the spec leaves to planning. Library behavior was
verified on 2026-09-04 by installing the candidates in a scratch folder
and rendering a probe document in node, both through pdfmake's node
entry and through its browser bundle.

## R1. Renderer: pdfmake 0.3.11, browser bundle

- **Decision**: `pdfmake` 0.3.11 with `@types/pdfmake` 0.3.3. The app
  imports `pdfmake/build/pdfmake` (the UMD browser bundle) through a
  dynamic import so it lands in its own chunk. The document is described
  as a pdfmake document definition, a plain object tree, built by pure
  functions.
- **Rationale**: pdfmake has a flowing layout engine with tables that
  repeat header rows across pages (`headerRows`), keep rows intact
  (`dontBreakRows`), a `footer` callback that receives the page number
  and page count, a `watermark` option, and explicit page breaks. Those
  are exactly FR-014 and FR-015. Because the output of the template is a
  data tree, wording and conditional rules are unit-testable without
  producing a PDF. The probe rendered a 4-page document with a 60-row
  table in 36 ms, with the header row repeated on every page and no row
  split.
- **Alternatives considered**: `@react-pdf/renderer` (tables are hand
  built from flex rows, repeated headers need `fixed` workarounds, and
  the output is React elements rather than inspectable data); `jsPDF` +
  `jspdf-autotable` (imperative drawing, paragraph flow across pages is
  manual); browser print to PDF (no file for Download, untestable);
  rendering on the service (rejected by the user's decision in the spec).

## R2. Fonts: the standard Helvetica family, no virtual file system

- **Decision**: register pdfmake's built-in Helvetica family
  (`pdfmake/build/standard-fonts/Helvetica`) with `addFonts` and set it
  as the default font. Do not ship `vfs_fonts.js` (836 KB of Roboto).
- **Rationale**: the standard 14 PDF fonts need no font file, so the
  bundle stays at the pdfmake engine alone. The probe confirmed that em
  dashes, curly quotes, the degree sign, and "US $6,000" render and
  extract correctly, which covers every character in the source
  document.
- **Known limitation**: standard fonts carry the WinAnsi character set.
  A value typed with characters outside it (for example an address in
  Devanagari) renders as substitution glyphs. This is recorded in
  quickstart.md as a known limitation. A Unicode font is a one-line
  registration when a deal needs one.
- **Alternatives considered**: bundling Roboto through `vfs_fonts.js`
  (836 KB for no gain on this document); bundling Inter (already a web
  dependency as a variable font, which pdfkit does not embed).

## R3. Viewer: an iframe on a blob URL, the browser's own PDF viewer

- **Decision**: the document page turns the Blob into an object URL and
  shows it in an `<iframe title="Agreement PDF">`. Page navigation and
  zoom come from the browser's built-in viewer. The app adds the page
  indicator "N of M" from the render result, a Download link (`<a
  download>` on the same URL), "Back to review", and "Back to my
  agreements". The object URL is revoked when the page unmounts or a new
  document replaces it.
- **Rationale**: every target browser ships a PDF viewer with paging and
  zoom. Adding pdf.js would double the chunk, draw to canvas, and be
  untestable in jsdom. The spec's page controls are satisfied by the
  viewer, and the app-level indicator and Download keep the spec's
  visible elements in the page.
- **CSP**: `web/public/_headers` gains `frame-src blob:`. `object-src
  'none'` stays. The browser walk in quickstart.md confirms Chrome and
  Safari render the frame under the deployed policy.
- **Alternatives considered**: `react-pdf` on pdf.js (heavier, canvas
  only, no built-in text selection without an extra layer); opening the
  PDF in a new tab (leaves the app, no Back links).

## R4. Where generation runs: on the document page, at mount

- **Decision**: "Generate PDF" on the Review step and "View PDF" on the
  list both navigate to `/agreements/:id/document`. The page reads the
  draft and organization from the store, builds the model, renders, and
  shows a busy state until the Blob exists. Pressing either action again
  while the page is generating does nothing new because the navigation
  target is the page that is already generating. Regeneration is a new
  mount: leaving the page revokes the URL, and returning renders again
  from current values (FR-006, FR-021).
- **Rationale**: one place generates, so the list action and the review
  action cannot drift. No Blob needs to cross routes, so no store is
  needed for it.
- **Alternatives considered**: generating on the Review step and
  carrying the Blob in a Zustand store to the page (a store entry for a
  value that must not outlive the page); generating in the background
  as steps save (rejected by the spec's Assumptions).

## R5. Two pure stages: document model, then content tree

- **Decision**: `buildDocumentModel(draft, organization)` produces a
  `DocumentModel` of already-formatted strings and boolean flags. The
  template turns the model into pdfmake `Content`. Formatting lives in
  `format.ts`. Wording lives in the template files. A `textOf(content)`
  helper flattens `Content` to a string for tests.
- **Rationale**: formatting rules (money, dates, units, digits) are
  tested once on the model. Wording rules are tested on the content tree
  by substring, without rendering. The render test then proves the whole
  pipeline on the real PDF.
- **Alternatives considered**: one function from draft to PDF (every
  test would need pdf-parse and would be slow and brittle).

## R6. Generic lists that the source wrote as prose

Feature 002 stored some source paragraphs as generic lists. The
template renders them with a lookup for the known default labels and a
generic form for anything else.

- **Environment limits** (`environment.limits`, rows of label, value,
  unit). The source's paragraph is six specific sentences. Known default
  labels map to the source sentence with the value and unit substituted
  (for example "Maximum from any single category" renders "No more than
  60 accepted usable hours may come from a single category."). An
  unknown label renders "{label}: {value} {unit}." The sentences render
  in row order. The difficulty mix sentence sits after the first two
  limits, matching the source order, and the two repetitive motion
  limits render after the diversity caps.
- **Difficulty caps** (`environment.difficultyCaps`, rows of scope and
  three numbers) render as "Binding diversity caps: {scope, first
  letter lowercased}: easy {e} {unit}, medium {m} {unit}, hard {h}
  {unit}; ..." joined with semicolons and "and" before the last item,
  the unit in its short form (R10).
- **Invoicing cadence** labels map to the two phrasings the source uses:
  Payment Basis "weekly invoicing in arrears" and section 10 "invoice
  weekly in arrears". Known labels have both forms in `phrases.ts`. An
  unknown label renders "{label, lowercased} invoicing" and "invoice
  {label, lowercased}".
- **Delivery cadence** renders the label with its first letter
  lowercased inside "deliver completed Deliverables {cadence} to".
- **Currency name** in section 10: USD renders "U.S. dollars" as the
  source does. Any other code renders the code itself, matching the spec
  edge case.
- **Rationale**: the default path reproduces the source word for word,
  and edited lists still produce correct sentences.
- **Alternatives considered**: changing the feature 002 form to named
  fields (out of scope, and the user asked to iterate on the form
  separately).

## R7. Filename

- **Decision**: `documentFilename(sowNumber, title)` returns
  `SOW {n} - {title}.pdf` with `/ \ : * ? " < > |` and control
  characters replaced by a space, runs of spaces collapsed, and the title
  trimmed to 120 characters.
- **Rationale**: FR-005 names the pattern. The replacements keep the
  name valid on macOS, Windows, and Linux.

## R8. Numbers and dates

- **Decision**: every number renders as digits (the source's "two
  years", "two times", and "five business days" become "2 years", "2
  times", and "5 business days"). Dates are ISO strings from the form,
  parsed as UTC and rendered as "September 6, 2026" through a new
  `formatLongDate` in `lib/dates.ts`. Money renders as "US $" plus an
  en-US grouped number, two decimals for unit fees and deposits, no
  decimals when a total is whole. Other currencies use the code and a
  space in place of "US $".
- **Rationale**: spec FR-011 and its Assumptions. Parsing as UTC avoids
  the off-by-one-day shift a local-time parse causes west of UTC.

## R9. Testing the real PDF

- **Decision**: `pdf-parse` 2.4.5 as a dev dependency. `render.test.ts`
  runs under `// @vitest-environment node`, renders the source-document
  fixture through the same `renderDocument` the page uses, and asserts:
  page count between 8 and 11, "Page 1 of N" present, "DRAFT" present,
  "CONTENT DEVELOPMENT AGREEMENT" on page 1, "EXHIBIT A" on a later
  page, a sample of fixed sentences present verbatim, and the
  conditional sentences present or absent for a yes and a no fixture.
- **Rationale**: the browser bundle of pdfmake runs under node (verified
  in the probe: `getBuffer` and `getBlob` both work), so the test
  exercises the production code path. pdf-parse needs node APIs that
  jsdom lacks, hence the per-file environment.
- **Alternatives considered**: `pdfjs-dist` directly (more setup for the
  same text).

## R10. Unit labels

- **Decision**: the form's unit label (for example "accepted usable
  hours") renders in full where the source uses the full phrase and in
  its short form, the label's last word, where the source uses "hours"
  or "hour" alone. `singularize` from feature 002 supplies the singular.
  A quantity of exactly 1 uses the singular.
- **Rationale**: spec FR-011. The source itself mixes "accepted usable
  hours", "accepted hours", and "hours". The rule keeps the sentences
  readable for any unit.

## R11. Fields with no fixed home in the source

- **Decision**: `scope.notes`, when non-empty, renders as a final
  paragraph of the SOW Overview. `scope.regions` renders as country
  names joined with commas and "and". The counterparty short name is
  not rendered (spec Assumptions).
- **Rationale**: the form captures notes as free text about the work,
  and the Overview is where the work is described.

## R12. Readiness and not-found

- **Decision**: `documentReadiness(draft, organization)` returns the
  list of missing items in step order plus organization info. The Review
  action is disabled and the note lists each missing item as a link when
  the list is non-empty. The list shows "View PDF" only when the list is
  empty. The document page for a draft id that is not in the store
  navigates to `/agreements/not-found`, the same path the wizard uses.
  A draft whose `templateVersion` has no registry entry shows the spec's
  template message and no retry.
- **Rationale**: FR-001, FR-002, FR-018, FR-019. A draft from another
  browser is by construction absent from this store.
