import { useState } from 'react'
import { Navigate, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { z } from 'zod'

import { AppHeader } from '../components/app-header'
import { StepAgreement } from '../components/steps/step-agreement'
import { StepCounterparty } from '../components/steps/step-counterparty'
import { StepDelivery } from '../components/steps/step-delivery'
import { StepEnvironment } from '../components/steps/step-environment'
import { StepPricing } from '../components/steps/step-pricing'
import { StepReview } from '../components/steps/step-review'
import { StepScope } from '../components/steps/step-scope'
import { StepTechnical } from '../components/steps/step-technical'
import { STEP_COUNT } from '../components/wizard-shell'
import {
  defaultAgreementTerms,
  defaultCounterparty,
  defaultDelivery,
  defaultEnvironment,
  defaultPricing,
  defaultScope,
  defaultTechnical,
} from '../defaults'
import { legacyInvoicingCadences, type Pricing, type Scope } from '../schemas'
import { nextSowNumber, useAgreementsStore, type DraftSteps } from '../store'
import { singularize } from '../units'

const stepParam = z.coerce.number().int().min(1).max(STEP_COUNT).catch(1)

// AgreementWizardPage drives the eight steps. A new agreement has no id
// until step 1 saves. Values typed on a step and abandoned with Back
// stay in page state for the rest of the visit.
export function AgreementWizardPage() {
  const { id } = useParams()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const drafts = useAgreementsStore((state) => state.drafts)
  const organization = useAgreementsStore((state) => state.organization)
  const createDraft = useAgreementsStore((state) => state.createDraft)
  const saveStep = useAgreementsStore((state) => state.saveStep)
  const [pending, setPending] = useState<Partial<DraftSteps>>({})

  const draft = id === undefined ? undefined : drafts[id]
  const step = id === undefined ? 1 : stepParam.parse(searchParams.get('step'))

  if (id !== undefined && draft === undefined) {
    return <Navigate to="/agreements/not-found" replace />
  }

  const steps: DraftSteps = { ...draft?.steps, ...pending }
  const stepPath = (target: number, draftId = id) =>
    draftId === undefined
      ? '/agreements/new'
      : `/agreements/${draftId}/edit?step=${target}`

  function goTo(target: number, draftId = id) {
    void navigate(stepPath(target, draftId))
  }

  function keep<K extends keyof DraftSteps>(key: K, values: DraftSteps[K]) {
    setPending((previous) => ({ ...previous, [key]: values }))
  }

  const scope: Scope = { ...defaultScope, ...steps.scope }
  const agreement = steps.agreement ?? defaultAgreementTerms(nextSowNumber(drafts))

  const content = (() => {
    switch (step) {
      case 1:
        return (
          <StepAgreement
            step={1}
            defaultValues={agreement}
            backTo="/agreements"
            onBack={() => undefined}
            onNext={(values) => {
              if (id === undefined) {
                const created = createDraft(values)
                goTo(2, created)
                return
              }
              saveStep(id, 'agreement', values)
              goTo(2)
            }}
          />
        )
      case 2:
        return (
          <StepScope
            step={2}
            defaultValues={scope}
            onBack={(values) => {
              keep('scope', values)
              goTo(1)
            }}
            onNext={(values) => {
              saveStep(id!, 'scope', values)
              goTo(3)
            }}
          />
        )
      case 3:
        return (
          <StepPricing
            step={3}
            defaultValues={pricingDefaults(scope, steps.pricing)}
            effectiveDate={agreement.effectiveDate}
            targetVolume={scope.targetVolume}
            unitSingular={singularize(scope.unit)}
            unitPlural={scope.unit}
            onBack={(values) => {
              keep('pricing', values)
              goTo(2)
            }}
            onNext={(values) => {
              saveStep(id!, 'pricing', values)
              goTo(4)
            }}
          />
        )
      case 4:
        return (
          <StepEnvironment
            step={4}
            defaultValues={{ ...defaultEnvironment(scope.unit), ...steps.environment }}
            unitPlural={scope.unit}
            onBack={(values) => {
              keep('environment', values)
              goTo(3)
            }}
            onNext={(values) => {
              saveStep(id!, 'environment', values)
              goTo(5)
            }}
          />
        )
      case 5:
        return (
          <StepTechnical
            step={5}
            defaultValues={{ ...defaultTechnical, ...steps.technical }}
            onBack={(values) => {
              keep('technical', values)
              goTo(4)
            }}
            onNext={(values) => {
              saveStep(id!, 'technical', values)
              goTo(6)
            }}
          />
        )
      case 6:
        return (
          <StepDelivery
            step={6}
            defaultValues={{ ...defaultDelivery, ...steps.delivery }}
            onBack={(values) => {
              keep('delivery', values)
              goTo(5)
            }}
            onNext={(values) => {
              saveStep(id!, 'delivery', values)
              goTo(7)
            }}
          />
        )
      case 7:
        return (
          <StepCounterparty
            step={7}
            defaultValues={{ ...defaultCounterparty, ...steps.counterparty }}
            onBack={(values) => {
              keep('counterparty', values)
              goTo(6)
            }}
            onNext={(values) => {
              saveStep(id!, 'counterparty', values)
              goTo(8)
            }}
          />
        )
      default:
        return (
          <StepReview
            step={8}
            draft={draft!}
            organization={organization}
            onBack={() => goTo(7)}
            stepPath={(target) => stepPath(target)}
          />
        )
    }
  })()

  return (
    <div className="min-h-svh bg-muted/40">
      <AppHeader />
      <main className="px-4 py-8">{content}</main>
    </div>
  )
}

// pricingDefaults merges the saved pricing over the defaults and maps
// cadence keys from older drafts to their labels.
function pricingDefaults(scope: Scope, saved: Pricing | undefined): Pricing {
  const merged = { ...defaultPricing(scope.targetVolume, scope.unit), ...saved }
  return {
    ...merged,
    invoicingCadence:
      legacyInvoicingCadences[merged.invoicingCadence] ?? merged.invoicingCadence,
  }
}
