import { create } from 'zustand'
import { persist } from 'zustand/middleware'

import { TEMPLATE_VERSION } from './defaults'
import type {
  AgreementTerms,
  Counterparty,
  Delivery,
  Environment,
  Organization,
  Pricing,
  Scope,
  Technical,
} from './schemas'

export interface DraftSteps {
  agreement?: AgreementTerms
  scope?: Scope
  pricing?: Pricing
  environment?: Environment
  technical?: Technical
  delivery?: Delivery
  counterparty?: Counterparty
}

export type StepKey = keyof DraftSteps

export interface Draft {
  id: string
  createdAt: number
  templateVersion: string
  steps: DraftSteps
}

interface AgreementsState {
  organization: Organization | null
  drafts: Record<string, Draft>
  setOrganization: (organization: Organization) => void
  createDraft: (agreement: AgreementTerms) => string
  saveStep: <K extends StepKey>(
    id: string,
    key: K,
    values: NonNullable<DraftSteps[K]>,
  ) => void
  deleteDraft: (id: string) => void
}

// useAgreementsStore holds organization info and drafts in the browser
// until the API owns them. Values persist in localStorage.
export const useAgreementsStore = create<AgreementsState>()(
  persist(
    (set, get) => ({
      organization: null,
      drafts: {},
      setOrganization: (organization) => set({ organization }),
      createDraft: (agreement) => {
        const id = crypto.randomUUID()
        const draft: Draft = {
          id,
          createdAt: Date.now(),
          templateVersion: TEMPLATE_VERSION,
          steps: { agreement },
        }
        set({ drafts: { ...get().drafts, [id]: draft } })
        return id
      },
      saveStep: (id, key, values) => {
        const draft = get().drafts[id]
        if (draft === undefined) return
        set({
          drafts: {
            ...get().drafts,
            [id]: { ...draft, steps: { ...draft.steps, [key]: values } },
          },
        })
      },
      deleteDraft: (id) => {
        const drafts = { ...get().drafts }
        delete drafts[id]
        set({ drafts })
      },
    }),
    { name: 'sow.agreements.v1' },
  ),
)

// nextSowNumber returns one more than the highest SOW number in use.
export function nextSowNumber(drafts: Record<string, Draft>): number {
  const numbers = Object.values(drafts).map(
    (draft) => draft.steps.agreement?.sowNumber ?? 0,
  )
  return Math.max(0, ...numbers) + 1
}
