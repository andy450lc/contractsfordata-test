import { describe, expect, it } from 'vitest'

import { defaultCounterparty, defaultEnvironment, defaultPricing } from './defaults'
import { counterpartySchema, environmentSchema, pricingSchema } from './schemas'

function messagesOf(result: {
  success: boolean
  error?: { issues: { path: PropertyKey[]; message: string }[] }
}) {
  return (result.error?.issues ?? []).map(
    (issue) => `${issue.path.join('.')}: ${issue.message}`,
  )
}

describe('pricingSchema', () => {
  const base = {
    ...defaultPricing(300, 'accepted usable hours'),
    unitFee: 20,
    invoiceEmail: 'ap@example.com',
  }

  it('accepts the source document values', () => {
    const values = {
      ...base,
      milestones: [
        { ...base.milestones[0]!, deadline: '2026-09-06' },
        { ...base.milestones[1]!, deadline: '2026-09-13' },
      ],
    }
    expect(pricingSchema('2026-08-29').safeParse(values).success).toBe(true)
  })

  it('rejects a deadline before the effective date', () => {
    const values = {
      ...base,
      milestones: [
        { ...base.milestones[0]!, deadline: '2026-08-01' },
        { ...base.milestones[1]!, deadline: '2026-09-13' },
      ],
    }
    expect(messagesOf(pricingSchema('2026-08-29').safeParse(values))).toContain(
      'milestones.0.deadline: Must be on or after the effective date',
    )
  })

  it('requires exactly one final milestone with the latest deadline', () => {
    const values = {
      ...base,
      milestones: [
        { ...base.milestones[0]!, deadline: '2026-09-20' },
        { ...base.milestones[1]!, deadline: '2026-09-13' },
      ],
    }
    expect(messagesOf(pricingSchema('2026-08-29').safeParse(values))).toContain(
      'milestones.0.deadline: Must be on or before the final delivery',
    )
    const noFinal = {
      ...values,
      milestones: values.milestones.map((row) => ({ ...row, isFinal: false })),
    }
    expect(messagesOf(pricingSchema('2026-08-29').safeParse(noFinal))).toContain(
      'milestones: Mark exactly one milestone as the final delivery',
    )
  })

  it('caps money at ten trillion and requires a deposit amount when a deposit is required', () => {
    const tooMuch = { ...base, unitFee: 10_000_000_000_000 }
    expect(messagesOf(pricingSchema('').safeParse(tooMuch))).toContain(
      'unitFee: Must be less than 10 trillion',
    )
    const deposit = { ...base, depositRequired: true, depositAmount: null }
    expect(messagesOf(pricingSchema('').safeParse(deposit))).toContain(
      'depositAmount: Required',
    )
  })
})

describe('environmentSchema', () => {
  it('accepts the source document defaults', () => {
    expect(environmentSchema.safeParse(defaultEnvironment('hours')).success).toBe(true)
  })

  it('requires the difficulty mix to sum to 100', () => {
    const values = {
      ...defaultEnvironment('hours'),
      difficulty: { easy: 20, medium: 30, hard: 40 },
    }
    expect(messagesOf(environmentSchema.safeParse(values))).toContain(
      'difficulty: Must add up to 100%',
    )
  })
})

describe('counterpartySchema', () => {
  it('requires detail fields only in details mode', () => {
    const details = { ...defaultCounterparty, email: 'a@b.co' }
    expect(messagesOf(counterpartySchema.safeParse(details))).toEqual(
      expect.arrayContaining(['legalName: Required', 'signatoryName: Required']),
    )
    const invite = { ...details, mode: 'invite' as const }
    expect(counterpartySchema.safeParse(invite).success).toBe(true)
  })

  it('names the invalid CC entry', () => {
    const values = {
      ...defaultCounterparty,
      mode: 'invite' as const,
      email: 'a@b.co',
      ccEmails: 'cfo@example.com, not-an-email',
    }
    expect(messagesOf(counterpartySchema.safeParse(values))).toContain(
      'ccEmails: "not-an-email" is not a valid email address',
    )
  })
})
