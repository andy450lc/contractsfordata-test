import type { components } from '@/lib/api/schema'

import {
  agreementTermsSchema,
  counterpartySchema,
  deliverySchema,
  environmentSchema,
  organizationSchema,
  pricingSchema,
  scopeSchema,
  technicalSchema,
} from '../schemas'
import type { Draft } from '../store'
import { TEMPLATE_VERSION } from '../defaults'
import type { Organization } from '../schemas'

type ContractConfiguration = components['schemas']['ContractConfiguration']

const incompleteMessage = 'Complete every required agreement field before delivery.'

// contractConfiguration validates a complete draft and maps it to the API schema.
export function contractConfiguration(
  draft: Draft,
  organization: Organization | null,
): ContractConfiguration {
  const { agreement, scope, pricing, environment, technical, delivery, counterparty } =
    draft.steps
  if (
    draft.templateVersion !== TEMPLATE_VERSION ||
    agreement === undefined ||
    scope === undefined ||
    pricing === undefined ||
    environment === undefined ||
    technical === undefined ||
    delivery === undefined ||
    counterparty === undefined ||
    organization === null
  ) {
    throw new Error(incompleteMessage)
  }

  const parsedAgreement = agreementTermsSchema.safeParse(agreement)
  const parsedScope = scopeSchema.safeParse(scope)
  const parsedPricing = pricingSchema(agreement.effectiveDate).safeParse(pricing)
  const parsedEnvironment = environmentSchema.safeParse(environment)
  const parsedTechnical = technicalSchema.safeParse(technical)
  const parsedDelivery = deliverySchema.safeParse(delivery)
  const parsedCounterparty = counterpartySchema.safeParse(counterparty)
  const parsedOrganization = organizationSchema.safeParse(organization)
  const results = [
    parsedAgreement,
    parsedScope,
    parsedPricing,
    parsedEnvironment,
    parsedTechnical,
    parsedDelivery,
    parsedCounterparty,
    parsedOrganization,
  ]
  if (results.some((result) => !result.success)) {
    throw new Error(incompleteMessage)
  }

  return {
    template_version: TEMPLATE_VERSION,
    organization: {
      legal_name: organization.legalName,
      entity_type: organization.entityType,
      incorporation_place: organization.incorporationPlace,
      governing_law: organization.governingLaw,
      address: organization.address,
      short_name: organization.shortName,
      signer_name: organization.signerName,
      signer_title: organization.signerTitle,
      signer_email: organization.signerEmail,
      contact_phone: organization.contactPhone,
    },
    steps: {
      agreement: {
        sow_number: agreement.sowNumber,
        title: agreement.title,
        effective_date: agreement.effectiveDate,
        exclusive: agreement.exclusive,
        materials_summary: agreement.materialsSummary,
        incident_notice_hours: agreement.incidentNoticeHours,
        cure_days: agreement.cureDays,
        convenience_notice_days: agreement.convenienceNoticeDays,
        records_retention_years: agreement.recordsRetentionYears,
        liability_lookback_months: agreement.liabilityLookbackMonths,
        excluded_claims_cap_multiplier: agreement.excludedClaimsCapMultiplier,
        data_claims_cap_floor: agreement.dataClaimsCapFloor,
      },
      scope: {
        deliverable_description: scope.deliverableDescription,
        regions: scope.regions,
        venue_constraint: scope.venueConstraint,
        target_volume: scope.targetVolume,
        unit: scope.unit,
        deliverables: scope.deliverables,
        ambient_audio: scope.ambientAudio ?? false,
        notes: scope.notes,
      },
      pricing: {
        currency: pricing.currency,
        unit_fee: pricing.unitFee,
        deposit_required: pricing.depositRequired,
        deposit_amount: pricing.depositAmount,
        invoicing_cadence: pricing.invoicingCadence,
        payment_terms_days: pricing.paymentTermsDays,
        payment_methods: pricing.paymentMethods,
        invoice_email: pricing.invoiceEmail,
        milestones: pricing.milestones.map((milestone) => ({
          name: milestone.name,
          volume: milestone.volume,
          deadline: milestone.deadline,
          notes: milestone.notes,
          is_final: milestone.isFinal,
        })),
        firm_deadline: pricing.firmDeadline,
      },
      environment: {
        verticals: environment.verticals,
        limits: environment.limits,
        difficulty: environment.difficulty,
        difficulty_caps: environment.difficultyCaps,
        plan_required: environment.planRequired,
        change_window_hours: environment.changeWindowHours,
        extra_prohibited: environment.extraProhibited,
        extra_excluded_settings: environment.extraExcludedSettings,
      },
      technical: {
        requirements: technical.requirements,
        pre_collection_materials: technical.preCollectionMaterials,
        condition_of_payment: technical.conditionOfPayment,
      },
      delivery: {
        storage_location: delivery.storageLocation,
        delivery_cadence: delivery.deliveryCadence,
        manifest_format: delivery.manifestFormat,
        manifest_fields: delivery.manifestFields,
        site_data_fields: delivery.siteDataFields,
        integrity_text: delivery.integrityText,
        review_window_days: delivery.reviewWindowDays,
        correction_days: delivery.correctionDays,
        deemed_acceptance: delivery.deemedAcceptance,
        rejected_stays_developer_owned: delivery.rejectedStaysDeveloperOwned,
        deletion_window_days: delivery.deletionWindowDays,
      },
      counterparty: {
        mode: counterparty.mode,
        email: counterparty.email,
        legal_name: counterparty.legalName,
        entity_jurisdiction: counterparty.entityJurisdiction,
        address: counterparty.address,
        short_name: counterparty.shortName,
        signatory_name: counterparty.signatoryName,
        signatory_title: counterparty.signatoryTitle,
        cc_emails: counterparty.ccEmails,
      },
    },
  }
}
