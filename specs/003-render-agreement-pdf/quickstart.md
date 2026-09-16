# Quickstart: Render Agreement PDF

**Feature**: 003-render-agreement-pdf · Validation guide (run after
implementation; see [plan.md](plan.md) for design, [spec.md](spec.md)
for acceptance criteria).

## Prerequisites

- Node 22 and npm 10 (`web/.nvmrc`, `packageManager`).
- `web/.env.local` with `VITE_API_URL=http://localhost:8080` and the
  backend running for sign-in (feature 001), or the session seeded the
  way the integration tests do.
- Saved organization info and one draft with all seven steps saved. The
  fastest path is the source-document fixture: sign in, then in the
  browser console run the snippet in "Seeding" below.

## Automated validation (CI-equivalent)

```bash
cd web && npm ci && npm run typecheck && npm run lint && npm run format:check && npm run test:coverage && npm run build
```

Key suites and what they prove:

| Suite | Proves |
|---|---|
| `src/features/agreements/document/format.test.ts` | FR-011: money, long dates, digits, unit forms, series and lettered lists |
| `src/features/agreements/document/model.test.ts` | Model derivation from the fixture, bracketed blanks in invite mode, lookups with fallbacks (R6), incomplete draft rejected |
| `src/features/agreements/document/templates/content-development-1/*.test.ts` | FR-007 to FR-010, FR-012, FR-013: section order and headings, value positions, every conditional rule in both settings, renumbering when the plan subsection is absent |
| `src/features/agreements/document/render.test.ts` (node) | SC-002 sample, SC-009 page count 8 to 11, FR-014 footer and DRAFT on every page, EXHIBIT A on a new page |
| `src/features/agreements/document.integration.test.tsx` | FR-001 to FR-006, FR-018, FR-019: disabled action with linked missing items, enabled action navigates, busy state, iframe and page count, download name, View PDF only when ready, unknown id, unknown template message, blob URL revoked on leave |
| `src/features/agreements/readiness.test.ts` | Missing items in step order plus organization info |
| `src/lib/dates.test.ts` | `formatLongDate` parses as UTC |

## Browser walk (constitution: look at it before reporting done)

1. Start the web dev server (`web` entry in `.claude/launch.json`) and
   the backend, sign in, and seed the fixture (below).
2. Open the agreements list. The seeded row shows "View PDF" and Delete.
   A second draft with only step 1 saved shows Delete alone.
3. Open the seeded draft's Review step. "Generate PDF" is enabled beside
   the disabled send button. Press it.
4. The document page shows "Generating your PDF…" briefly, then the
   frame with page 1: the heading "CONTENT DEVELOPMENT AGREEMENT", the
   preamble with both parties, and the footer "SOW 1 · DRAFT · Page 1 of
   N". Expected N: 9 to 11.
5. Scroll to the Exhibit A page. Confirm the title "SOW 1 — PIXELS2
   INDIA STEREO EGOCENTRIC PILOT (EXCLUSIVE)", the five tables with
   their headings, and the Technical table's header repeated if it
   crosses a page.
6. Press "Download PDF". The saved file is named "SOW 1 - Pixels2 India
   Stereo Egocentric Pilot.pdf" and opens in Preview or Acrobat with the
   same page count.
7. Go back to review, open step 1, set Exclusive engagement to No, save
   through to review, and generate again. The title suffix and the
   exclusivity sentences are gone.
8. Open step 7, switch to "Invite by email only", save through, and
   generate. The preamble and both signature blocks show bracketed
   blanks for the Developer.
9. Check the browser console: no errors, and no party names or emails
   in any log line.
10. Deployed-policy check: `npm run build` then `npm run preview` with
    `web/dist/_headers` applied (or the deployed stage) and confirm the
    frame renders under `frame-src blob:` in Chrome and Safari.

Take a screenshot of step 4 and step 5 for the completion report.

## Seeding the fixture in a browser

Paste into the console on any signed-in page, then reload the list:

```js
// Copies web/src/test/factories/source-document.ts values into the store.
// The implementation exposes it in dev builds as window.__sowSeedSourceDocument().
window.__sowSeedSourceDocument()
```

## Known limitations

- Standard PDF fonts cover the WinAnsi character set. Text outside it
  (for example Devanagari or CJK) renders as substitution glyphs. A
  Unicode font is a one-line font registration when needed (research R2).
- The page indicator "1 of N" reports the total; the current page is
  tracked by the browser's own viewer.
