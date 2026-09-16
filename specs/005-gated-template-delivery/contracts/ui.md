# UI Contract: Disclaimer and Delivery Dialog

Every trigger is a button or guarded control. Opening starts a new attempt and
moves focus into a semantic modal titled `Before You Download`.

## Step 1

- Exact two-paragraph disclaimer in a bounded scroll container.
- Instruction: `Scroll to the bottom to enable the acknowledgment.`
- Genuine checkbox with the exact acknowledgment label.
- Checkbox unavailable until measured as read.
- Live announcement when the checkbox becomes available.
- Continue disabled until checked.
- Cancel closes without requesting a document.

## Step 2

- Primary action: `Download Word document`.
- Labeled `Work email` input.
- Primary action: `Email Word document`.
- Back returns to step 1 without bypassing acknowledgment.
- Cancel closes without requesting a document.
- Loading text names the active operation and buttons are disabled while it is
  pending.
- Errors use an alert and leave the relevant action available for retry.
- Email success is shown only after the API's provider-confirmed response.

The action area remains reachable on short screens while the disclaimer body
scrolls. Escape closes unless a request is in a non-cancelable response handoff.

