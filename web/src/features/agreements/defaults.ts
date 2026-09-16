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

// TEMPLATE_VERSION names the wording every draft's values belong to.
export const TEMPLATE_VERSION = 'content-development-1'

const rows = (items: string[]) => items.map((text) => ({ text }))

export const emptyOrganization: Organization = {
  legalName: '',
  entityType: '',
  incorporationPlace: '',
  governingLaw: '',
  address: '',
  shortName: '',
  signerName: '',
  signerTitle: '',
  signerEmail: '',
  contactPhone: '',
}

// defaultAgreementTerms carries the standard numbers from the master
// agreement. Deal-specific fields start empty.
export function defaultAgreementTerms(sowNumber: number): AgreementTerms {
  return {
    sowNumber,
    title: '',
    effectiveDate: '',
    exclusive: true,
    materialsSummary:
      'recordings, stereo video, IMU data, calibration data, metadata, and other materials',
    incidentNoticeHours: 24,
    cureDays: 30,
    convenienceNoticeDays: 30,
    recordsRetentionYears: 2,
    liabilityLookbackMonths: 12,
    excludedClaimsCapMultiplier: 2,
    dataClaimsCapFloor: 50_000,
  }
}

export const defaultScope: Scope = {
  deliverableDescription: '',
  regions: [],
  venueConstraint: '',
  targetVolume: Number.NaN,
  unit: 'accepted usable hours',
  deliverables: rows([
    'Original stereo video and associated camera, IMU, calibration, and metadata files',
    'The delivery manifest described in the Delivery section',
    'The collection and consent records sufficient to verify chain of title on reasonable request',
    'All other files specified in this SOW',
  ]),
  notes: '',
}

// defaultPricing fills the final milestone's volume from the scope step.
export function defaultPricing(targetVolume: number, unitPlural: string): Pricing {
  const volume = Number.isFinite(targetVolume) ? `${targetVolume} ${unitPlural}` : ''
  return {
    currency: 'USD',
    unitFee: Number.NaN,
    depositRequired: false,
    depositAmount: null,
    invoicingCadence: 'Weekly in arrears',
    paymentTermsDays: 30,
    paymentMethods: 'ACH or wire transfer',
    invoiceEmail: '',
    milestones: [
      {
        name: 'Rolling daily uploads',
        volume: 'Completed data as available',
        deadline: '',
        notes: 'Daily after collection begins',
        isFinal: false,
      },
      { name: 'Final delivery', volume, deadline: '', notes: '', isFinal: true },
    ],
    firmDeadline: true,
  }
}

// defaultEnvironment fills the limits with the source document's
// limits, counted in the scope's unit.
export function defaultEnvironment(unit: string): Environment {
  return {
    verticals: [
      {
        verticals:
          'Beauty/personal care; sports/leisure; commercial cleaning/facilities management',
        target:
          'Highest priority. Target meaningful coverage across all three where available.',
        details:
          'Genuine workplace tasks only. Prioritize skilled, multi-step work over routine cleaning, stocking or other repetitive actions.',
      },
      {
        verticals:
          'Apparel/textile; footwear manufacturing; warehousing/logistics/fulfilment',
        target: 'Second priority. Approx. balanced coverage across available categories.',
        details:
          'Real operating businesses and workers performing their normal jobs. Favor skilled and varied workflows.',
      },
      {
        verticals:
          'Electronics manufacturing/PCB/device assembly; glass/ceramics/porcelain; toy/small-appliance/consumer-goods assembly',
        target: 'Use to complete remaining volume after higher-priority categories.',
        details:
          "Factory-floor work only. Avoid restricted technical data and repetitive single-station work. Printing/packaging or jewellery/watch manufacturing may be substituted with the Customer's written approval.",
      },
    ],
    limits: [
      { limit: 'Maximum from any single category', value: '60', unit },
      { limit: 'Minimum distinct physical sites', value: '15', unit: 'sites' },
      {
        limit:
          'Minimum share performed by skilled workers doing their own occupation at their own workplace',
        value: '70',
        unit: '%',
      },
      { limit: 'Maximum per operator at one site across all tasks', value: '40', unit },
      { limit: 'Maximum repetitive motion per task and site', value: '1', unit },
      {
        limit:
          'Maximum repetitive motion per task and site where the worker moves through the space and the workflow genuinely varies',
        value: '2',
        unit,
      },
    ],
    difficulty: { easy: 20, medium: 30, hard: 50 },
    difficultyCaps: [
      { scope: 'Per operator, task, and site', easy: '1', medium: '2', hard: '10' },
      {
        scope: 'Per task and site across operators',
        easy: '10',
        medium: '20',
        hard: '40',
      },
      {
        scope: 'Per distinct task across the project',
        easy: '4.5',
        medium: '12',
        hard: '18',
      },
    ],
    planRequired: true,
    changeWindowHours: 24,
    extraProhibited: [],
    extraExcludedSettings: [],
  }
}

export const defaultTechnical: Technical = {
  requirements: [
    {
      requirement: 'Modality',
      specification: 'Stereo RGB; head-mounted/first-person egocentric',
      rejection: 'Any non-stereo or non-egocentric material.',
    },
    {
      requirement: 'Resolution / frame rate',
      specification: 'At least 1080p per eye; at least 30 fps',
      rejection: 'Below either floor.',
    },
    {
      requirement: 'Field of view / baseline',
      specification:
        'No ultra-wide lens, at least 110° horizontal and no more than 160° horizontal, 80° vertical after rectification; baseline measured per device',
      rejection: 'Missing measurement or below FoV floor.',
    },
    {
      requirement: 'Synchronization / IMU',
      specification:
        'Shared camera clock with under 10 ms offset; IMU at least 200 Hz, synchronized; camera-to-IMU offset reported',
      rejection: 'Out-of-spec or missing calibration data.',
    },
    {
      requirement: 'Codec / GOP',
      specification:
        'H.265 or H.264 at 10 Mbps or higher; no B-frames; GOP/keyframe interval no more than 30 frames at 30 fps',
      rejection: 'Any non-conforming encode.',
    },
    {
      requirement: 'Hands / clips',
      specification:
        'Hands in frame throughout active capture; 10+ consecutive seconds out of view may be rejected; clips 30 seconds to 30 minutes, approximately 10 minutes average',
      rejection: 'Does not meet stated requirement.',
    },
    {
      requirement: 'Calibration',
      specification:
        'Per-session intrinsics, distortion model/coefficient, stereo extrinsics/baseline, and camera-to-IMU transform',
      rejection: 'Missing or incomplete calibration.',
    },
  ],
  preCollectionMaterials: rows([
    'Device-specification sheet',
    'Representative sample footage',
    'Calibration files',
  ]),
  conditionOfPayment: true,
}

export const defaultDelivery: Delivery = {
  storageLocation: 'Customer-designated cloud bucket',
  deliveryCadence: 'Daily',
  manifestFormat:
    'CSV, JSON, or other format approved by the Customer. Dataset transport format will be MCAP + protobuf where applicable; the Customer may approve an alternate vendor-native delivery format in writing where the Customer will perform conversion to the required transport format.',
  manifestFields: rows([
    'Object path',
    'File size',
    'SHA-256 checksum',
    'Duration',
    'Task ID',
    'Task description',
    'Task difficulty',
    'Skill group',
    'Operator job',
    'Stable operator ID across sessions',
    'Recorded consent status',
    'Broad environment category',
    'Venue type',
    'Specific room or workstation',
    'Stable environment ID per physical site across sessions and deliveries',
    'Collection date',
    'Device ID',
    'Device frame rate',
    'Calibration date',
    'Calibration status',
    'Consent-record ID',
  ]),
  siteDataFields: rows([
    'Business name',
    'Registered address',
    'Government business identifier',
    'Six-digit NAICS industry code',
  ]),
  integrityText:
    'The Developer will provide manifests, checksums, malware-scan results, and other reasonable ingest-integrity information requested by the Customer.',
  reviewWindowDays: 30,
  correctionDays: 5,
  deemedAcceptance: true,
  rejectedStaysDeveloperOwned: true,
  deletionWindowDays: 30,
}

export const defaultCounterparty: Counterparty = {
  mode: 'details',
  email: '',
  legalName: '',
  entityJurisdiction: '',
  address: '',
  shortName: '',
  signatoryName: '',
  signatoryTitle: '',
  ccEmails: '',
}
