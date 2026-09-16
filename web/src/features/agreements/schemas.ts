import { z } from 'zod'

import { isCountryCode } from './countries'
import { isCurrencyCode } from './currency'

const MONEY_LIMIT = 10_000_000_000_000

// requiredText accepts a non-empty string after trimming.
const requiredText = z.string().trim().min(1, 'Required')

const optionalText = z.string().trim()

// requiredNumber accepts a finite number. An empty input arrives as null.
const requiredNumber = z
  .number({ error: 'Required' })
  .refine((value) => Number.isFinite(value), 'Required')

const nonNegative = requiredNumber.min(0, 'Must be 0 or more')

const money = nonNegative.lt(MONEY_LIMIT, 'Must be less than 10 trillion')

const percent = nonNegative.max(100, 'Must be between 0 and 100')

const wholeNumber = nonNegative.int('Must be a whole number')

const email = requiredText.pipe(z.email('Enter a valid email address'))

// ccEmails accepts a comma-separated list. The message names the first
// invalid entry.
const ccEmails = optionalText.superRefine((value, ctx) => {
  if (value === '') return
  const entries = value.split(',').map((entry) => entry.trim())
  const invalid = entries.find((entry) => !z.email().safeParse(entry).success)
  if (invalid !== undefined) {
    ctx.addIssue({ code: 'custom', message: `"${invalid}" is not a valid email address` })
  }
})

const dateText = requiredText.regex(/^\d{4}-\d{2}-\d{2}$/, 'Enter a date')

const textRow = z.object({ text: requiredText })

export const organizationSchema = z.object({
  legalName: requiredText,
  entityType: requiredText,
  incorporationPlace: requiredText,
  governingLaw: requiredText,
  address: requiredText,
  shortName: requiredText,
  signerName: requiredText,
  signerTitle: requiredText,
  signerEmail: email,
  contactPhone: optionalText,
})

export const agreementTermsSchema = z.object({
  sowNumber: wholeNumber.min(1, 'Must be 1 or more'),
  title: requiredText,
  effectiveDate: dateText,
  exclusive: z.boolean(),
  materialsSummary: requiredText,
  incidentNoticeHours: wholeNumber,
  cureDays: wholeNumber,
  convenienceNoticeDays: wholeNumber,
  recordsRetentionYears: wholeNumber,
  liabilityLookbackMonths: wholeNumber,
  excludedClaimsCapMultiplier: nonNegative,
  dataClaimsCapFloor: money,
})

export const scopeSchema = z.object({
  deliverableDescription: requiredText,
  regions: z
    .array(z.string().refine((code): boolean => isCountryCode(code), 'Pick a country'))
    .min(1, 'Pick at least one country'),
  venueConstraint: requiredText,
  targetVolume: requiredNumber.positive('Must be more than 0'),
  unit: requiredText,
  deliverables: z.array(textRow).min(1, 'Add at least one deliverable'),
  ambientAudio: z.boolean().optional(),
  notes: optionalText,
})

export const invoicingCadences = [
  'Weekly in arrears',
  'Every two weeks in arrears',
  'Monthly in arrears',
  'Per milestone',
  'On acceptance of each batch',
] as const

// legacyInvoicingCadences maps the keys older drafts stored to labels.
export const legacyInvoicingCadences: Record<string, string> = {
  weekly: 'Weekly in arrears',
  biweekly: 'Every two weeks in arrears',
  monthly: 'Monthly in arrears',
  per_milestone: 'Per milestone',
}

export const paymentMethods = [
  'ACH or wire transfer',
  'Wire transfer (SWIFT)',
  'ACH',
  'SEPA transfer',
  'Bank transfer (NEFT, RTGS, or IMPS)',
  'UPI',
  'Wise',
  'PayPal',
  'Stripe',
  'Credit card',
  'Cheque',
] as const

const milestoneSchema = z.object({
  name: requiredText,
  volume: requiredText,
  deadline: dateText,
  notes: optionalText,
  isFinal: z.boolean(),
})

// pricingSchema builds the step 3 schema. Deadlines are checked against
// the effective date saved on step 1.
export function pricingSchema(effectiveDate: string) {
  return z
    .object({
      currency: requiredText.refine(
        (code): boolean => isCurrencyCode(code),
        'Pick a currency',
      ),
      unitFee: money,
      depositRequired: z.boolean(),
      depositAmount: z.number().nullable(),
      invoicingCadence: requiredText,
      paymentTermsDays: wholeNumber,
      paymentMethods: requiredText,
      invoiceEmail: email,
      milestones: z.array(milestoneSchema).min(1, 'Add at least one milestone'),
      firmDeadline: z.boolean(),
    })
    .superRefine((values, ctx) => {
      if (
        values.depositRequired &&
        (values.depositAmount === null || values.depositAmount <= 0)
      ) {
        ctx.addIssue({ code: 'custom', path: ['depositAmount'], message: 'Required' })
      }
      const finalRows = values.milestones.filter((row) => row.isFinal)
      if (finalRows.length !== 1) {
        ctx.addIssue({
          code: 'custom',
          path: ['milestones'],
          message: 'Mark exactly one milestone as the final delivery',
        })
      }
      const finalDeadline = finalRows[0]?.deadline ?? ''
      values.milestones.forEach((row, index) => {
        if (effectiveDate !== '' && row.deadline < effectiveDate) {
          ctx.addIssue({
            code: 'custom',
            path: ['milestones', index, 'deadline'],
            message: 'Must be on or after the effective date',
          })
        }
        if (!row.isFinal && finalDeadline !== '' && row.deadline > finalDeadline) {
          ctx.addIssue({
            code: 'custom',
            path: ['milestones', index, 'deadline'],
            message: 'Must be on or before the final delivery',
          })
        }
      })
    })
}

const verticalSchema = z.object({
  verticals: requiredText,
  target: requiredText,
  details: requiredText,
})

const limitSchema = z.object({
  limit: requiredText,
  value: requiredText.regex(/^\d+(\.\d+)?$/, 'Enter a number'),
  unit: optionalText,
})

const numberText = requiredText.regex(/^\d+(\.\d+)?$/, 'Enter a number')

const difficultyCapSchema = z.object({
  scope: requiredText,
  easy: numberText,
  medium: numberText,
  hard: numberText,
})

export const environmentSchema = z
  .object({
    verticals: z.array(verticalSchema).min(1, 'Add at least one row'),
    limits: z.array(limitSchema).min(1, 'Add at least one limit'),
    difficulty: z.object({ easy: percent, medium: percent, hard: percent }),
    difficultyCaps: z.array(difficultyCapSchema),
    planRequired: z.boolean(),
    changeWindowHours: wholeNumber,
    extraProhibited: z.array(textRow),
    extraExcludedSettings: z.array(textRow),
  })
  .superRefine((values, ctx) => {
    const { easy, medium, hard } = values.difficulty
    if (Math.round((easy + medium + hard) * 100) / 100 !== 100) {
      ctx.addIssue({
        code: 'custom',
        path: ['difficulty'],
        message: 'Must add up to 100%',
      })
    }
  })

const requirementSchema = z.object({
  requirement: requiredText,
  specification: requiredText,
  rejection: requiredText,
})

export const technicalSchema = z.object({
  requirements: z.array(requirementSchema).min(1, 'Add at least one requirement'),
  preCollectionMaterials: z.array(textRow),
  conditionOfPayment: z.boolean(),
})

export const deliveryCadences = [
  'Daily',
  'Twice a week',
  'Weekly',
  'Every two weeks',
  'Per milestone',
  'On completion',
] as const

export const deliverySchema = z.object({
  storageLocation: requiredText,
  deliveryCadence: requiredText,
  manifestFormat: requiredText,
  manifestFields: z.array(textRow).min(1, 'Add at least one field'),
  siteDataFields: z.array(textRow),
  integrityText: requiredText,
  reviewWindowDays: wholeNumber,
  correctionDays: wholeNumber,
  deemedAcceptance: z.boolean(),
  rejectedStaysDeveloperOwned: z.boolean(),
  deletionWindowDays: wholeNumber,
})

export const counterpartyModes = ['details', 'invite'] as const

const counterpartyBase = z.object({
  mode: z.enum(counterpartyModes),
  email: email,
  legalName: z.string().trim(),
  entityJurisdiction: z.string().trim(),
  address: z.string().trim(),
  shortName: z.string().trim(),
  signatoryName: z.string().trim(),
  signatoryTitle: z.string().trim(),
  ccEmails: ccEmails,
})

const detailFields = [
  'legalName',
  'entityJurisdiction',
  'address',
  'shortName',
  'signatoryName',
] as const

// counterpartySchema requires the detail fields only in details mode.
export const counterpartySchema = counterpartyBase.superRefine((values, ctx) => {
  if (values.mode !== 'details') return
  for (const field of detailFields) {
    if (values[field] === '') {
      ctx.addIssue({ code: 'custom', path: [field], message: 'Required' })
    }
  }
})

export type Organization = z.infer<typeof organizationSchema>
export type AgreementTerms = z.infer<typeof agreementTermsSchema>
export type Scope = z.infer<typeof scopeSchema>
export type Pricing = z.infer<ReturnType<typeof pricingSchema>>
export type Milestone = z.infer<typeof milestoneSchema>
export type Environment = z.infer<typeof environmentSchema>
export type Technical = z.infer<typeof technicalSchema>
export type Delivery = z.infer<typeof deliverySchema>
export type Counterparty = z.infer<typeof counterpartySchema>
