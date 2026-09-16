import { describe, expect, it } from 'vitest'

import { sourceDraft, sourceOrganization } from '@/test/factories/source-document'

import { contractConfiguration } from './configuration'

describe('contractConfiguration', () => {
  it('validates the complete draft and maps every field to the generated schema', () => {
    const draft = sourceDraft()

    expect(contractConfiguration(draft, sourceOrganization())).toEqual({
      template_version: 'content-development-1',
      organization: {
        legal_name: 'Sieve Inc.',
        entity_type: 'corporation',
        incorporation_place: 'Delaware',
        governing_law: 'Delaware',
        address: '100 Market St, San Francisco, CA 94105',
        short_name: 'Sieve',
        signer_name: 'Ishan Dhawan',
        signer_title: 'Chief of Staff',
        signer_email: 'ishan@sieve.example',
        contact_phone: '',
      },
      steps: {
        agreement: expect.objectContaining({
          sow_number: 1,
          effective_date: '2026-08-29',
          materials_summary: draft.steps.agreement?.materialsSummary,
        }),
        scope: expect.objectContaining({
          deliverable_description: draft.steps.scope?.deliverableDescription,
          target_volume: 300,
        }),
        pricing: expect.objectContaining({
          unit_fee: 20,
          deposit_required: false,
          deposit_amount: null,
          firm_deadline: true,
          milestones: expect.arrayContaining([
            expect.objectContaining({ is_final: true }),
          ]),
        }),
        environment: expect.objectContaining({
          difficulty_caps: draft.steps.environment?.difficultyCaps,
          plan_required: true,
        }),
        technical: expect.objectContaining({
          pre_collection_materials: draft.steps.technical?.preCollectionMaterials,
          condition_of_payment: true,
        }),
        delivery: expect.objectContaining({
          storage_location: draft.steps.delivery?.storageLocation,
          deemed_acceptance: true,
          rejected_stays_developer_owned: true,
        }),
        counterparty: expect.objectContaining({
          legal_name: draft.steps.counterparty?.legalName,
          signatory_name: draft.steps.counterparty?.signatoryName,
          cc_emails: '',
        }),
      },
    })
  })

  it('refuses missing and invalid configuration before making a request', () => {
    expect(() =>
      contractConfiguration(sourceDraft({ pricing: undefined }), sourceOrganization()),
    ).toThrow('Complete every required agreement field before delivery.')

    const draft = sourceDraft()
    expect(() =>
      contractConfiguration(
        {
          ...draft,
          steps: {
            ...draft.steps,
            environment: {
              ...draft.steps.environment!,
              difficulty: { easy: 20, medium: 30, hard: 40 },
            },
          },
        },
        sourceOrganization(),
      ),
    ).toThrow('Complete every required agreement field before delivery.')
  })

  it('preserves the optional ambient-audio choice and defaults legacy drafts', () => {
    const draft = sourceDraft()
    const withAudio = {
      ...draft,
      steps: {
        ...draft.steps,
        scope: { ...draft.steps.scope!, ambientAudio: true },
      },
    }

    expect(
      contractConfiguration(withAudio, sourceOrganization()).steps.scope,
    ).toHaveProperty('ambient_audio', true)
    expect(contractConfiguration(draft, sourceOrganization()).steps.scope).toHaveProperty(
      'ambient_audio',
      false,
    )
  })
})
