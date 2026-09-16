# SoW — Project Context

SoW is a product for creating, sending, and signing Statements of Work
online. It does for the SoW what the YC SAFE does for seed investment:
one standard document, filled in from a structured template, signed by
every party, and stored with the evidence of signing.

The product name "SoW" is a placeholder until the user picks one.

## Flow

1. A **sender** logs in and creates an agreement by filling a structured
   template (parties, scope, deliverables, milestones, payment terms,
   dates).
2. The app renders a canonical PDF from those fields and stores it.
3. The sender sends the agreement. Each **signer** receives an email with
   a unique magic link.
4. A signer opens the link, reviews the document, and signs. No account
   is needed.
5. When every required signature is in, the agreement is complete. The
   signed PDF and its signature evidence are stored and available to all
   parties.

## Actors

| Role | Created via | Capabilities |
|---|---|---|
| **Sender** | Self sign-up (WorkOS AuthKit) | Creates, sends, voids, and views their own agreements. |
| **Signer** | Magic link, no account | Views and signs the one agreement the link grants. |
| **Admin** | Backend-created only | Operational access. **Deferred — not in initial scope.** |

- A sender may also be a signer of their own agreement (countersigning).
- Nothing about an agreement is visible without either a sender session
  or a valid signer link.

## The signed artifact

- The agreement is authored as **structured fields**, stored in Postgres.
- The **canonical PDF** is rendered server-side from those fields. Its
  hash is what a signature covers. Once the first signature lands, the
  PDF and the fields it was rendered from are frozen. Changes after that
  point mean a new agreement or a voided one.
- Each **signature record** stores: signer email, display name as typed,
  timestamp, IP address, user agent, the document hash, and the consent
  acknowledgement. Records are append-only.
- PDFs live in Cloudflare R2 and are served through short-lived presigned
  URLs.

## Out of scope for the initial release

- Payments, invoicing, or escrow tied to milestones.
- Free-form clause editing. The template is fixed per version.
- Signer accounts, dashboards, or notifications beyond the emails in the
  signing flow.

## Notes for specs

- Feature numbering starts at `feature/001-…` (constitution, Development
  Workflow). The governing rules live in `.specify/memory/constitution.md`
  and the binding convention docs in `docs/`.
- Security posture is elevated for this domain: agreement contents,
  party PII, and signature evidence (constitution, Principle III).
- Decisions still open and owned by feature specs: the PDF renderer, the
  transactional email provider, e-signature consent wording, and
  retention periods.
