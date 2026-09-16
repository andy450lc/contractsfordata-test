import { WIZARD_STEPS } from './components/wizard-shell'
import type { Organization } from './schemas'
import type { Draft, StepKey } from './store'

export interface MissingItem {
  label: string
  to: string
}

export interface DocumentReadiness {
  ready: boolean
  missing: MissingItem[]
}

const stepOrder: { key: StepKey; step: number }[] = [
  { key: 'agreement', step: 1 },
  { key: 'scope', step: 2 },
  { key: 'pricing', step: 3 },
  { key: 'environment', step: 4 },
  { key: 'technical', step: 5 },
  { key: 'delivery', step: 6 },
  { key: 'counterparty', step: 7 },
]

// documentReadiness reports whether a draft can render a document and
// lists what is missing in step order, then organization info.
export function documentReadiness(
  draft: Draft,
  organization: Organization | null,
): DocumentReadiness {
  const missing: MissingItem[] = stepOrder
    .filter(({ key }) => draft.steps[key] === undefined)
    .map(({ step }) => ({
      label: WIZARD_STEPS[step - 1] ?? `Step ${step}`,
      to: `/agreements/${draft.id}/edit?step=${step}`,
    }))
  if (organization === null) {
    missing.push({ label: 'Organization info', to: '/organization' })
  }
  return { ready: missing.length === 0, missing }
}
