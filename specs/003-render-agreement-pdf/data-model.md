# Data Model: Render Agreement PDF

**Date**: 2026-09-04 · **Feature**: 003-render-agreement-pdf

No persisted data changes. The feature adds three in-memory shapes on
top of feature 002's `Draft` and `Organization`.

## DocumentReadiness

```ts
interface MissingItem {
  label: string        // "Scope", "Developer", "Organization info"
  to: string           // link target: wizard step path or /organization
}
interface DocumentReadiness {
  ready: boolean
  missing: MissingItem[]   // in step order 1 to 7, then organization info
}
```

Derived by `documentReadiness(draft, organization)`. `ready` is true when
every step key in `DraftSteps` is defined and `organization` is not null.

## DocumentModel

The template's only input. Every value is already formatted for the
page. Produced by `buildDocumentModel(draft, organization)`, which
throws `IncompleteDraftError` when readiness is false.

| Group | Field | Source | Formatting |
|---|---|---|---|
| meta | `sowNumber` | agreement.sowNumber | digits |
| meta | `templateVersion` | draft.templateVersion | as stored |
| customer | `legalName`, `entityType`, `incorporationPlace`, `address`, `shortName`, `signerName`, `signerTitle`, `governingLaw` | organization | as typed |
| developer | `known` | counterparty.mode === 'details' | boolean |
| developer | `legalName`, `entityJurisdiction`, `address`, `signatoryName`, `signatoryTitle` | counterparty | as typed, or bracketed blank when `known` is false, for example "[Developer legal name]" |
| terms | `effectiveDate` | agreement.effectiveDate | long date |
| terms | `title`, `exclusive`, `materialsSummary` | agreement | as typed |
| terms | `incidentNoticeHours`, `cureDays`, `convenienceNoticeDays`, `recordsRetentionYears`, `liabilityLookbackMonths`, `excludedClaimsCapMultiplier` | agreement | digits |
| terms | `dataClaimsCapFloor` | agreement | money, USD, no decimals when whole |
| scope | `description`, `venueConstraint` | scope | as typed |
| scope | `regions` | scope.regions | country names joined with commas and "and" |
| scope | `targetVolume` | scope.targetVolume | digits with separators |
| scope | `unitPlural`, `unitSingular`, `unitShortPlural`, `unitShortSingular` | scope.unit | R10 |
| scope | `deliverables` | scope.deliverables | text rows |
| scope | `ambientAudio` | scope.ambientAudio if present, else false | boolean |
| scope | `notes` | scope.notes | as typed |
| pricing | `currency` | pricing.currency | code |
| pricing | `unitFee`, `maximumTotal`, `depositAmount` | pricing, scope | money |
| pricing | `depositRequired`, `firmDeadline` | pricing | boolean |
| pricing | `invoicingCadence` | pricing.invoicingCadence | label as stored; the template maps it to phrases (R6) |
| pricing | `paymentTermsDays` | pricing | digits |
| pricing | `paymentMethods`, `invoiceEmail` | pricing | as typed |
| pricing | `currencyName` | pricing.currency | "U.S. dollars" or the code |
| pricing | `milestones` | pricing.milestones | rows: `name`, `volume`, `deadlineText` (notes + ", no later than " + long date, or the long date), `isFinal` |
| pricing | `firstDeadline`, `finalDeadline` | milestones | long dates: earliest non-final deadline (or the final one when none), and the final row's deadline |
| environment | `verticals` | environment.verticals | rows as typed |
| environment | `limits` | environment.limits | rows as stored; the template renders sentences (R6) |
| environment | `difficulty` | environment.difficulty | `{ easy, medium, hard }` digits with "%" |
| environment | `difficultyCaps` | environment.difficultyCaps | rows as stored; the template renders the caps sentence (R6) |
| environment | `planRequired` | environment.planRequired | boolean |
| environment | `changeWindowHours` | environment | digits |
| environment | `extraProhibited`, `extraExcludedSettings` | environment | text rows |
| technical | `requirements` | technical.requirements | rows as typed |
| technical | `preCollectionMaterials` | technical | text rows, joined as a series |
| technical | `conditionOfPayment` | technical | boolean |
| delivery | `cadence` | delivery.deliveryCadence | first letter lowercased |
| delivery | `storageLocation`, `manifestFormat`, `integrityText` | delivery | as typed |
| delivery | `manifestFields`, `siteDataFields` | delivery | text rows, joined as a series |
| delivery | `reviewWindowDays`, `correctionDays`, `deletionWindowDays` | delivery | digits |
| delivery | `deemedAcceptance`, `rejectedStaysDeveloperOwned` | delivery | boolean |

Note on `ambientAudio`: feature 002's `Scope` schema has no ambient audio
field. The spec's FR-010 lists it as conditional. The model reads
`scope.ambientAudio` when a future form version adds it and defaults to
false, which renders the source's no-audio sentence.

## RenderedDocument

```ts
interface RenderedDocument {
  blob: Blob            // application/pdf
  pageCount: number     // captured from the footer callback
  templateVersion: string
  generatedAt: number   // Unix ms
}
```

Produced by `renderDocument(model)`. Held in the document page's
component state together with the object URL created from `blob`. The
URL is revoked in the effect cleanup. Nothing is written to the store.

## Template registry

```ts
type Template = (model: DocumentModel) => Content[]
const templates: Record<string, Template> = { 'content-development-1': contentDevelopment1 }
function templateFor(version: string): Template   // throws UnknownTemplateError
```

## Errors

| Error | Raised by | Page behavior |
|---|---|---|
| `IncompleteDraftError` | `buildDocumentModel` | Not reached from the UI because both entry points check readiness. Shown as the generic failure if it ever is. |
| `UnknownTemplateError` | `templateFor` | "This draft uses a template this version cannot render." No retry. |
| any other | `renderDocument` | "Couldn't generate the PDF. Try again." with a Try again button. Console gets the draft id and template version only. |
