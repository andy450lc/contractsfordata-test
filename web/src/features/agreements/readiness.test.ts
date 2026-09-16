import { describe, expect, it } from 'vitest'

import { sourceDraft, sourceOrganization } from '@/test/factories/source-document'

import { defaultAgreementTerms } from './defaults'
import { documentReadiness } from './readiness'
import type { Draft } from './store'

describe('documentReadiness', () => {
  it('lists every unsaved step in order, then organization info', () => {
    const draft: Draft = {
      id: 'd1',
      createdAt: 0,
      templateVersion: 'content-development-1',
      steps: { agreement: defaultAgreementTerms(1) },
    }

    const result = documentReadiness(draft, null)

    expect(result.ready).toBe(false)
    expect(result.missing).toEqual([
      { label: 'Scope', to: '/agreements/d1/edit?step=2' },
      { label: 'Pricing & schedule', to: '/agreements/d1/edit?step=3' },
      { label: 'Environment & task mix', to: '/agreements/d1/edit?step=4' },
      { label: 'Technical standards', to: '/agreements/d1/edit?step=5' },
      { label: 'Delivery & acceptance', to: '/agreements/d1/edit?step=6' },
      { label: 'Developer', to: '/agreements/d1/edit?step=7' },
      { label: 'Organization info', to: '/organization' },
    ])
  })

  it('is ready when every step and the organization are saved', () => {
    const result = documentReadiness(sourceDraft(), sourceOrganization())
    expect(result).toEqual({ ready: true, missing: [] })
  })

  it('names only the organization when the steps are complete', () => {
    const result = documentReadiness(sourceDraft(), null)
    expect(result.ready).toBe(false)
    expect(result.missing).toEqual([{ label: 'Organization info', to: '/organization' }])
  })
})
