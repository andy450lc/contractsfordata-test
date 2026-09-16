# Contract: Template `content-development-1`

**Date**: 2026-09-04 · **Feature**: 003-render-agreement-pdf

The section order, headings, and value positions are specified in
[spec.md](../spec.md) under "Reference: structure of the source
document". This contract adds the rules the template code follows and
the lookups it carries. The fixed wording is transcribed from
`source/source-document.txt` (gitignored) during implementation.

## Page setup

- US Letter, margins 72 pt left and right, 64 pt top, 66 pt bottom.
- Default font Helvetica 10.5 pt, line height 1.2. Headings bold. Section
  titles bold with the number, for example "1. Services." Subsection
  numbers bold, for example "1.1. Content Development Services."
- Footer on every page, centered, 8 pt: "SOW {n} · DRAFT · Page {p} of
  {c}".
- Watermark "DRAFT", bold, opacity 0.08.
- "EXHIBIT A" starts a new page.
- Tables: header row bold, `headerRows: 1`, `dontBreakRows: true`,
  light grid lines, cell padding 4 pt. Column widths: Pricing
  `['*', '*', '*', '*']`; Delivery Schedule `['*', '*', '*']`;
  Environment `['*', '*', '*']`; Technical `['auto', '*', '*']`;
  Delivery / Manifest `['auto', '*']`.

## Party terms

- Customer running text uses `customer.shortName` wherever the source
  says "Sieve", including "Sieve's" → "{shortName}'s" and "Sieve-designated"
  → "{shortName}-designated".
- The counterparty is "Developer" throughout. Its details appear in the
  master preamble and both signature blocks only.
- Master preamble: `This Content Development Agreement (the “Agreement”)
  is effective as of {effectiveDate} and is entered into between
  {customer.legalName}, a {customer.entityType} of
  {customer.incorporationPlace}, with an address at {customer.address}
  (“{customer.shortName}”), and {developer.legalName}, a
  {developer.entityJurisdiction}, with an address at {developer.address}
  (“Developer”). {shortName} and Developer are each a “Party” and
  together the “Parties.”`
- SOW preamble: `This Statement of Work (“SOW”) is entered into under the
  Content Development Agreement between {customer.legalName} and
  {developer.legalName} dated {effectiveDate} (the “Agreement”).
  Capitalized terms not defined here have the meanings in the
  Agreement.`
- Signature block, both places: the sentence `IN WITNESS WHEREOF, the
  parties have executed this Agreement as of the Effective Date.` then
  two columns. Left: `{customer.legalName}`, `Signature:`, `Name:
  {signerName}`, `Title: {signerTitle}`, `Date:`. Right:
  `{developer.legalName}`, `Signature:`, `Name: {signatoryName}`,
  `Title: {signatoryTitle}`, `Date:`.

## Conditional rules (spec FR-010)

| Flag | When true | When false |
|---|---|---|
| `terms.exclusive` | Title suffix " (EXCLUSIVE)"; Overview "…{targetVolume} {unitPlural} of exclusive, {description}…"; sentence "The engagement and all Deliverables are exclusive." | Suffix absent; "of {description}"; sentence absent. |
| `pricing.firmDeadline` | Delivery Schedule paragraph ends with the three source sentences (firm obligation, reduce or terminate, written extension). | Paragraph ends after the "must upload the full … by EOD …" sentence. |
| `pricing.depositRequired` | "{shortName} will pay a deposit of {depositAmount} against the fees above. There is no other advance payment or hardware contribution." | "There is no deposit, advance payment, or hardware contribution." |
| `scope.ambientAudio` | "Ambient audio is permitted." | "No ambient audio is permitted unless {shortName} approves it in writing." |
| `environment.planRequired` | Subsection "Collection Plan" with `{changeWindowHours}` present as 6.1, then Natural Activity 6.2, Privacy 6.3. | Subsection absent; Natural Activity is 6.1, Privacy is 6.2. |
| `technical.conditionOfPayment` | "Every technical item is a condition of acceptance and payment." | "Every technical item is a condition of acceptance." |
| `delivery.deemedAcceptance` | Sentence "A batch not rejected within the {reviewWindowDays}-day review period is deemed accepted only if Developer has first delivered the complete manifest and all required supporting materials." | Sentence absent. |
| `delivery.rejectedStaysDeveloperOwned` | Overview ends "…subject to the rejected-material carveout in Section 9."; Acceptance carries the "Notwithstanding Section 4…" sentence and the deletion sentence with `{deletionWindowDays}`. | Overview ends "…Work Product subject to Section 4 of the Agreement."; both sentences absent. |
| `environment.extraProhibited` non-empty | Items appended to the 6.3 list before "may be captured or delivered". | List unchanged. |
| `environment.extraExcludedSettings` non-empty | Items appended to the 6.2 list before "or material previously captured…". | List unchanged. |

## Lookups (`phrases.ts`)

Limit labels (feature 002 defaults) to sentences. `{v}` is the value,
`{u}` the unit as stored on the row:

| Label | Sentence |
|---|---|
| Maximum from any single category | No more than {v} {u} may come from a single category. |
| Minimum distinct physical sites | At least {v} distinct physical sites are required. |
| Minimum share performed by skilled workers doing their own occupation at their own workplace | At least {v}% of {unitPlural} must be performed by skilled workers doing their own occupation at their own workplace. |
| Maximum per operator at one site across all tasks | rendered inside the caps sentence as "and per operator at one site across all tasks: {v} {u}" |
| Maximum repetitive motion per task and site | Repetitive motion is limited to {v} {u} per task/site |
| Maximum repetitive motion per task and site where the worker moves through the space and the workflow genuinely varies | , or {v} {u} only where the worker moves through the space and the workflow genuinely varies. |
| any other | {label}: {v} {u}. |

Invoicing cadence labels to phrases:

| Label | Payment Basis phrase | Section 10 phrase |
|---|---|---|
| Weekly in arrears | weekly invoicing in arrears | invoice weekly in arrears |
| Every two weeks in arrears | invoicing every two weeks in arrears | invoice every two weeks in arrears |
| Monthly in arrears | monthly invoicing in arrears | invoice monthly in arrears |
| Per milestone | invoicing per milestone | invoice per milestone |
| On acceptance of each batch | invoicing on acceptance of each batch | invoice on acceptance of each batch |
| any other | {label lowercased} invoicing | invoice {label lowercased} |

Currency names for section 10: `USD` → "U.S. dollars"; otherwise the
code.

## Text helpers

- `series(items)` joins with ", " and ", and " before the last item, or
  " and " for two items.
- `lettered(items)` renders "(a) …; (b) …; and (n) …".
- `lowerFirst(text)` lowercases the first character.
