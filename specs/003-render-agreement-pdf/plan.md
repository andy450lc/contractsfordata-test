# Implementation Plan: Render Agreement PDF

**Branch**: `feature/003-render-agreement-pdf` | **Date**: 2026-09-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/003-render-agreement-pdf/spec.md`

## Summary

Add a browser-side PDF renderer to the agreements feature. A draft that
has all seven steps saved, plus saved organization info, gains a
"Generate PDF" action on the Review step and a "View PDF" action on the
list. Both open a new document page at `/agreements/:id/document`, which
builds a pdfmake document definition from the local draft, renders it to
a Blob with pdfmake's browser bundle and the built-in Helvetica fonts,
shows it in an iframe through a blob URL, and offers a Download control.
The template for version `content-development-1` transcribes the source
document's fixed wording, places every form value, and applies the nine
conditional wording rules. Nothing is sent to the API and nothing is
stored. The blob URL is revoked when the page unmounts.

## Technical Context

**Language/Version**: TypeScript ~6.0, React 19, Vite 8, Node 22 (pinned
in `web/.nvmrc`). No backend change.

**Primary Dependencies**: existing web stack (react-router-dom 7,
zustand, zod 4, shadcn/ui, Tailwind 4). New: `pdfmake` 0.3.11 (browser
bundle, standard fonts, no virtual file system), `@types/pdfmake` 0.3.3.
New dev dependency: `pdf-parse` 2.4.5 for text extraction in the render
test.

**Storage**: none. Drafts and organization info stay in the persisted
Zustand store from feature 002. The rendered Blob lives in component
state for the life of the document page.

**Testing**: Vitest + Testing Library. Unit tests on the pure document
model and the template content tree. One node-environment render test
that produces a real PDF from the source document's values and extracts
its text with pdf-parse. Integration tests that mount the real routes,
seed the store, press "Generate PDF", and assert the document page, the
iframe source, the page count, the download name, the disabled states,
and the not-found path.

**Target Platform**: current desktop browsers with a built-in PDF viewer
(Chrome, Edge, Firefox, Safari). Same SPA deployment as feature 001.

**Project Type**: web application, frontend only for this feature.

**Performance Goals**: document page shows page 1 within 5 s of pressing
"Generate PDF" on a current laptop (SC-003). The probe rendered a
4-page document with a 60-row table in 36 ms in node. The pdfmake bundle
(about 1 MB minified) loads as a lazy chunk on first use.

**Constraints**: no document leaves the browser (spec FR-017). No party
names, emails, or addresses in console output or error messages
(FR-020). CSP must allow a blob URL in an iframe (`frame-src blob:`).
Cyclomatic complexity per function at or below 10. Standard PDF fonts
cover the WinAnsi character set only. See research R2.

**Scale/Scope**: one new route, one new page, one new action on two
existing screens, one template with about 60 fixed paragraphs and five
tables, and about 12 test files.

## Constitution Check

*GATE: evaluated pre-Phase-0 and re-checked post-design. PASS with one
justified deviation recorded in Complexity Tracking.*

| Principle | Status | Notes |
|---|---|---|
| I. Spec-Driven Development | PASS | spec.md governs. No API surface, so `contracts/openapi.yaml` is untouched. UI contracts live in `contracts/`. |
| II. Test-First & Meaningful Coverage | PASS | Every task pairs a failing test with its implementation. Integration tests mount the real router and store. The render test asserts extracted text against the source wording, page count, and conditional sentences. Coverage gate stays at 85 percent. |
| III. Security is Paramount | PASS with deviation | No document flows through the API because no document leaves the browser. The blob URL is revoked on unmount. Errors are reported by draft id and template version only. CSP gains `frame-src blob:` and nothing else. The deviation from "rendered PDFs live in R2" is the user's decision, recorded in spec Assumptions and below. |
| IV. Simplicity & Patterns | PASS | Two pure stages, model then content tree, keep formatting and wording testable without rendering. Template sections are one function each. No new store, no new global state. |
| V. Contract Integrity & RESTful API | PASS | No API change. |
| VI. Observability & Debuggability | PASS | Generation failures surface in the page with a retry control and reach the route error boundary when thrown outside the generation path. Console output carries ids only. |

**Post-Phase-1 re-check**: PASS. Design added the blob URL revocation
and the standard-font character limitation (research R2), both recorded.

## Project Structure

### Documentation (this feature)

```text
specs/003-render-agreement-pdf/
├── plan.md                        # This file
├── research.md                    # Phase 0 decisions R1 to R12
├── data-model.md                  # Phase 1 document model, readiness, rendered document
├── quickstart.md                  # Phase 1 run and validate
├── contracts/
│   ├── ui-routes.md               # New route, actions, states, messages
│   └── document-template.md       # Template content-development-1 rules and lookups
├── source/                        # gitignored: source-document.pdf and .txt
└── tasks.md                       # Phase 2 (/speckit-tasks)
```

### Source Code (repository root)

```text
web/
├── package.json                                   # + pdfmake, @types/pdfmake, pdf-parse (dev)
├── public/_headers                                # CSP: + frame-src blob:
└── src/
    ├── app/router.tsx                             # + /agreements/:id/document (lazy)
    ├── lib/dates.ts                               # + formatLongDate(isoDate)
    ├── features/agreements/
    │   ├── index.ts                               # + AgreementDocumentPage
    │   ├── readiness.ts                           # documentReadiness(draft, organization)
    │   ├── readiness.test.ts
    │   ├── document/
    │   │   ├── model.ts                           # buildDocumentModel(draft, organization)
    │   │   ├── model.test.ts
    │   │   ├── format.ts                          # money, dates, units, lists, digits
    │   │   ├── format.test.ts
    │   │   ├── text-of.ts                         # flattens pdfmake Content to text (tests + page count)
    │   │   ├── doc-definition.ts                  # page setup, footer, watermark, assembly
    │   │   ├── doc-definition.test.ts
    │   │   ├── render.ts                          # renderDocument(model): lazy pdfmake, Blob, page count
    │   │   ├── render.test.ts                     # @vitest-environment node, pdf-parse
    │   │   ├── registry.ts                        # templateFor(version) or UnknownTemplateError
    │   │   ├── filename.ts                        # documentFilename(sowNumber, title)
    │   │   ├── filename.test.ts
    │   │   └── templates/content-development-1/
    │   │       ├── index.ts                       # template entry: master + exhibit
    │   │       ├── master-agreement.ts            # sections 1 to 12, preamble, signature block
    │   │       ├── master-agreement.test.ts
    │   │       ├── statement-of-work.ts           # SOW sections 1 to 10, title, preamble
    │   │       ├── statement-of-work.test.ts
    │   │       ├── signature-block.ts             # shared two-column block
    │   │       └── phrases.ts                     # lookups: limits, cadences, currency names
    │   ├── components/steps/step-review.tsx       # + Generate PDF action and missing-items note
    │   ├── pages/agreements-list-page.tsx         # + View PDF action
    │   ├── pages/agreement-document-page.tsx      # new page
    │   └── document.integration.test.tsx          # new integration suite
    └── test/factories/source-document.ts          # draft + organization matching the source document
```

**Structure Decision**: everything lives inside the `agreements` feature
module per frontend conventions §1. The document code sits in its own
`document/` folder so the template files, which are long by nature, stay
apart from the wizard. The template version folder name matches
`TEMPLATE_VERSION` so a second version is a sibling folder and one
registry entry. `formatLongDate` joins `lib/dates.ts` because the
conventions route all date formatting through that module.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Rendered document is produced and held in the browser rather than rendered by the service and stored in R2 (constitution III, project context) | The user chose the browser-only preview on 2026-09-03 after weighing three options (spec Assumptions). Feature 002 has no agreements API, and the ask was to generate and show the document. | Moving drafts to the service first would turn a rendering feature into the agreements API feature. Rendering on the service from browser-held drafts would store documents with no agreement record behind them. The sending feature performs the canonical server render and reuses this template. |
| A lazy 1 MB chunk for pdfmake | The renderer needs a layout engine with tables, repeated headers, and page counts. | A browser print-to-PDF path produces no file for Download and cannot be tested. Lighter libraries lack flowing tables (research R1). |

## Implementation Notes (post-build reconciliation, 2026-09-04)

Deviations from the planned design, recorded as they happened:

- **Helvetica is a font container, not a font dictionary**: pdfmake's
  browser build ships `build/standard-fonts/Helvetica` as `{ vfs, fonts }`
  carrying the AFM metric files, so the renderer registers it with
  `addFontContainer`. The plain dictionary under `standard-fonts/` is for
  the node entry only.
- **Page setup matches the source**: one-inch side margins and 10.5 pt
  type (instead of 54 pt and 10 pt) bring the fixture to 8 pages, inside
  the 8 to 11 range of SC-009. The contract was updated to match.
- **List wording lives in the template stage**: the model carries the
  limit rows, the difficulty-cap rows, and the cadence label as stored,
  and `phrases.ts` turns them into sentences. data-model.md was updated.
- **Series conjunction**: the source joins its prohibited and excluded
  lists with "or", so `series` takes a conjunction argument.
- **Silent h**: "an accepted usable hour is an hour" needs an article
  rule for words that start with a silent h.
- **`nodesOf` helper** joined `text-of.ts` so tests can assert table
  properties and page breaks on the content tree.
- **Wizard shell gained `extraAction` and `note` slots** for the Generate
  PDF button and the missing-items note beside the disabled send button.
- **Retry resets state in the click handler**, not in the effect, to
  satisfy the React hooks lint rule on setState inside effects.
- **Dev seed** is exported through the feature index so `app/dev-seed.ts`
  never deep-imports the fixture.
- **Browser walk deferred to the credential hand-off**: the pages need a
  real WorkOS sign-in. The rendered PDF was looked at page by page from
  an export of the same render path, and the page states are covered by
  the integration suite. The in-app document page is reported as
  unverified in the browser until the user signs in to the preview.
