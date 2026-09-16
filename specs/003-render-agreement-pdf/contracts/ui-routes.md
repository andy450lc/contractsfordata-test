# Contract: Document Page, Review Action, and List Action

**Date**: 2026-09-04 · **Feature**: 003-render-agreement-pdf

## New route

| Path | Guard | Element | Behavior |
|---|---|---|---|
| `/agreements/:id/document` | ProtectedLayout | AgreementDocumentPage (lazy) | Reads the draft and organization from the store. Unknown id → `<Navigate to="/agreements/not-found" replace />`. Not ready → `<Navigate to="/agreements/:id/edit?step=8" replace />`. Otherwise generates on mount and shows the states below. |

### Page states

| State | Visible elements |
|---|---|
| Generating | Heading "Your agreement", text "Generating your PDF…", a busy indicator with `role="status"`, "Back to review" and "Back to my agreements" links. No Download. |
| Ready | Heading "Your agreement", subheading "SOW {n} — {title}", text "{M} pages" and the indicator "1 of {M}" beside the frame, `<iframe title="Agreement PDF" src="blob:…">`, a Download link named "Download PDF" with `download="SOW {n} - {title}.pdf"`, "Back to review" (→ `/agreements/:id/edit?step=8`), "Back to my agreements" (→ `/agreements`). A line "Draft preview. Signing arrives in a later release." |
| Failed (unknown template) | The text "This draft uses a template this version cannot render." and the two Back links. |
| Failed (other) | The text "Couldn't generate the PDF. Try again." and a "Try again" button that generates again, plus the two Back links. |

The page header is the shared `AppHeader`. Leaving the page revokes the
object URL.

## Review step (step 8)

- A primary button "Generate PDF" appears in the footer beside the
  existing disabled "Send agreement & signing links" button, which keeps
  its note "Sending arrives in a later release".
- Enabled when `documentReadiness(draft, organization).ready`. Pressing
  it navigates to `/agreements/:id/document`.
- Disabled otherwise, with a note beneath: "Complete {items} to generate
  the PDF." where `{items}` lists each missing item as a link, joined
  with commas and "and", for example "Complete Developer and add your
  organization info to generate the PDF." The organization item reads
  "add your organization info" and links to `/organization`. Step items
  use the step name from `WIZARD_STEPS` and link to that step.

## Agreements list

- Each draft row gains a "View PDF" link (→ `/agreements/:id/document`)
  in the actions cell, before Delete, only when the draft is ready. Rows
  that are not ready show Delete alone.

## Messages

| Key | Text |
|---|---|
| generating | Generating your PDF… |
| unknownTemplate | This draft uses a template this version cannot render. |
| failed | Couldn't generate the PDF. Try again. |
| retry | Try again |
| download | Download PDF |
| draftNote | Draft preview. Signing arrives in a later release. |
| missing | Complete {items} to generate the PDF. |
