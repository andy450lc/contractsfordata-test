import {
  defaultAgreementTerms,
  defaultCounterparty,
  defaultDelivery,
  defaultEnvironment,
  defaultPricing,
  defaultScope,
  defaultTechnical,
} from '../defaults'
import type { Organization } from '../schemas'
import { TEMPLATE_VERSION } from '../defaults'
import { useAgreementsStore, type Draft, type DraftSteps } from '../store'

const UNIT = 'accepted usable hours'

// sourceOrganization is the Customer of the source document.
export function sourceOrganization(): Organization {
  return {
    legalName: 'Sieve Inc.',
    entityType: 'corporation',
    incorporationPlace: 'Delaware',
    governingLaw: 'Delaware',
    address: '100 Market St, San Francisco, CA 94105',
    shortName: 'Sieve',
    signerName: 'Ishan Dhawan',
    signerTitle: 'Chief of Staff',
    signerEmail: 'ishan@sieve.example',
    contactPhone: '',
  }
}

// sourceSteps holds every step of the source document's SOW 1.
export function sourceSteps(): Required<DraftSteps> {
  const pricing = defaultPricing(300, UNIT)
  return {
    agreement: {
      ...defaultAgreementTerms(1),
      title: 'Pixels2 India Stereo Egocentric Pilot',
      effectiveDate: '2026-08-29',
    },
    scope: {
      ...defaultScope,
      deliverableDescription: 'stereo RGB, head-mounted first-person egocentric footage',
      regions: ['IN'],
      venueConstraint: 'real, non-residential operating businesses',
      targetVolume: 300,
      unit: UNIT,
    },
    pricing: {
      ...pricing,
      unitFee: 20,
      invoiceEmail: 'ap@sievedata.com',
      milestones: [
        {
          name: 'Rolling daily uploads',
          volume: 'Completed data as available',
          deadline: '2026-09-06',
          notes: 'Daily after collection begins',
          isFinal: false,
        },
        {
          name: 'Final delivery',
          volume: `300 ${UNIT}`,
          deadline: '2026-09-13',
          notes: '',
          isFinal: true,
        },
      ],
    },
    environment: defaultEnvironment(UNIT),
    technical: defaultTechnical,
    delivery: defaultDelivery,
    counterparty: {
      ...defaultCounterparty,
      email: 'akshaj@pixels2.example',
      legalName: 'Pixels Two Corporation',
      entityJurisdiction: 'Delaware corporation',
      address: '251 Little Falls Drive, Wilmington, New Castle County, Delaware 19808',
      shortName: 'Pixels',
      signatoryName: 'Akshaj Jain',
      signatoryTitle: 'President',
    },
  }
}

// sourceDraft builds a complete draft matching the source document.
// Overrides replace whole steps.
export function sourceDraft(overrides: Partial<DraftSteps> = {}): Draft {
  return {
    id: 'draft_source',
    createdAt: 1756425600000,
    templateVersion: TEMPLATE_VERSION,
    steps: { ...sourceSteps(), ...overrides },
  }
}

// seedSourceDocument writes the organization and the draft into the
// store and returns the draft id.
export function seedSourceDocument(): string {
  const store = useAgreementsStore.getState()
  store.setOrganization(sourceOrganization())
  const draft = sourceDraft()
  useAgreementsStore.setState({
    drafts: { ...useAgreementsStore.getState().drafts, [draft.id]: draft },
  })
  return draft.id
}
