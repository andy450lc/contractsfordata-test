import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'

import { countryName } from '../../countries'
import { formatMoney } from '../../currency'
import { formatNumber } from '../../number-format'
import { documentReadiness, type MissingItem } from '../../readiness'
import type { Organization } from '../../schemas'
import type { Draft } from '../../store'
import { singularize } from '../../units'
import { TemplateDeliveryDialog } from '../template-delivery-dialog'
import { WizardShell } from '../wizard-shell'

interface StepReviewProps {
  step: number
  draft: Draft
  organization: Organization | null
  onBack: () => void
  stepPath: (step: number) => string
}

function yesNo(value: boolean | undefined): string {
  if (value === undefined) return ''
  return value ? 'Yes' : 'No'
}

function joinRows(rows: { text: string }[] | undefined): string {
  return (rows ?? []).map((row) => row.text).join('; ')
}

interface Row {
  label: string
  value: ReactNode
}

function ReviewGroup({
  title,
  editTo,
  rows,
  note,
}: {
  title: string
  editTo?: string
  rows: Row[]
  note?: ReactNode
}) {
  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-center justify-between">
        <h3 className="font-semibold">{title}</h3>
        {editTo ? (
          <Link to={editTo} className="text-sm underline underline-offset-4">
            Edit
          </Link>
        ) : null}
      </div>
      <dl className="divide-y rounded-lg border text-sm">
        {rows.map((row) => (
          <div key={row.label} className="grid gap-1 px-3 py-2 sm:grid-cols-[14rem_1fr]">
            <dt className="text-muted-foreground">{row.label}</dt>
            <dd className="whitespace-pre-wrap sm:text-right">{row.value}</dd>
          </div>
        ))}
      </dl>
      {note ? <p className="text-xs text-muted-foreground">{note}</p> : null}
    </section>
  )
}

// joinNodes separates nodes with commas and "and" the way a sentence
// lists items.
function joinNodes(nodes: ReactNode[]): ReactNode[] {
  if (nodes.length <= 1) return nodes
  if (nodes.length === 2) return [nodes[0], ' and ', nodes[1]]
  return nodes.flatMap((node, index) => {
    if (index === 0) return [node]
    const separator = index === nodes.length - 1 ? ', and ' : ', '
    return [separator, node]
  })
}

// MissingNote names what the sender must finish before a document can be generated.
function MissingNote({ items }: { items: MissingItem[] }) {
  const links = items.map((item) => (
    <Link key={item.to} to={item.to} className="underline underline-offset-4">
      {item.label === 'Organization info' ? 'add your organization info' : item.label}
    </Link>
  ))
  return (
    <p className="max-w-sm text-right text-xs text-muted-foreground">
      Complete {joinNodes(links)} to get the Word document.
    </p>
  )
}

// StepReview shows every saved value as read-only tables. Sending is
// disabled until the sending feature ships.
export function StepReview({
  step,
  draft,
  organization,
  onBack,
  stepPath,
}: StepReviewProps) {
  const { agreement, scope, pricing, environment, technical, delivery, counterparty } =
    draft.steps
  const readiness = documentReadiness(draft, organization)
  const currency = pricing?.currency ?? 'USD'
  const maxTotal =
    pricing &&
    scope &&
    Number.isFinite(pricing.unitFee) &&
    Number.isFinite(scope.targetVolume)
      ? formatMoney(pricing.unitFee * scope.targetVolume, currency)
      : ''

  return (
    <form onSubmit={(event) => event.preventDefault()}>
      <WizardShell
        step={step}
        heading="Review & send"
        subtitle="Confirm everything looks right before sending the signing links."
        onBack={onBack}
        nextLabel="Send agreement & signing links"
        nextDisabled
        nextNote="Sending arrives in a later release"
        extraAction={
          readiness.ready ? (
            <TemplateDeliveryDialog
              draft={draft}
              organization={organization}
              trigger={
                <Button type="button" size="lg" variant="outline">
                  Get Word document
                </Button>
              }
            />
          ) : (
            <Button type="button" size="lg" variant="outline" disabled>
              Get Word document
            </Button>
          )
        }
        note={readiness.ready ? null : <MissingNote items={readiness.missing} />}
      >
        <ReviewGroup
          title="Agreement"
          editTo={stepPath(1)}
          rows={[
            { label: 'SOW number', value: agreement?.sowNumber ?? '' },
            {
              label: 'Title',
              value: agreement
                ? `${agreement.title}${agreement.exclusive ? ' (Exclusive)' : ''}`
                : '',
            },
            { label: 'Effective date', value: agreement?.effectiveDate ?? '' },
            { label: 'Exclusive engagement', value: yesNo(agreement?.exclusive) },
            { label: 'Materials summary', value: agreement?.materialsSummary ?? '' },
            {
              label: 'Standard terms',
              value: agreement
                ? `Incident notice ${agreement.incidentNoticeHours} hours; cure ${agreement.cureDays} days; convenience notice ${agreement.convenienceNoticeDays} days; records ${agreement.recordsRetentionYears} years; look-back ${agreement.liabilityLookbackMonths} months; excluded-claims cap ${agreement.excludedClaimsCapMultiplier}× fees; data claims floor ${formatMoney(agreement.dataClaimsCapFloor, 'USD')}`
                : '',
            },
          ]}
        />
        <ReviewGroup
          title="Scope"
          editTo={stepPath(2)}
          rows={[
            {
              label: 'Deliverable description',
              value: scope?.deliverableDescription ?? '',
            },
            {
              label: 'Collection countries',
              value: (scope?.regions ?? []).map((code) => countryName(code)).join(', '),
            },
            { label: 'Allowed venues', value: scope?.venueConstraint ?? '' },
            {
              label: 'Target volume',
              value: scope ? `${formatNumber(scope.targetVolume)} ${scope.unit}` : '',
            },
            { label: 'Deliverables', value: joinRows(scope?.deliverables) },
            { label: 'Additional notes', value: scope?.notes ?? '' },
          ]}
        />
        <ReviewGroup
          title="Pricing & schedule"
          editTo={stepPath(3)}
          rows={[
            {
              label: `Fee per ${scope ? singularize(scope.unit) : 'unit'}`,
              value: pricing ? formatMoney(pricing.unitFee, currency) : '',
            },
            { label: 'Maximum total', value: maxTotal },
            {
              label: 'Deposit or advance',
              value: pricing
                ? pricing.depositRequired
                  ? formatMoney(pricing.depositAmount, currency)
                  : 'None'
                : '',
            },
            {
              label: 'Invoicing',
              value: pricing
                ? `${pricing.invoicingCadence}; Net ${pricing.paymentTermsDays}; ${pricing.paymentMethods}`
                : '',
            },
            { label: 'Invoice email', value: pricing?.invoiceEmail ?? '' },
            {
              label: 'Milestones',
              value: (pricing?.milestones ?? [])
                .map(
                  (row) =>
                    `${row.name}: ${row.volume} by ${row.deadline}${row.isFinal ? ' (final)' : ''}`,
                )
                .join('\n'),
            },
            { label: 'Final deadline is firm', value: yesNo(pricing?.firmDeadline) },
          ]}
        />
        <ReviewGroup
          title="Environment & task mix"
          editTo={stepPath(4)}
          rows={[
            {
              label: 'Collection verticals',
              value: (environment?.verticals ?? [])
                .map((row) => row.verticals)
                .join('\n'),
            },
            {
              label: 'Limits',
              value: (environment?.limits ?? [])
                .map((row) => `${row.limit}: ${row.value} ${row.unit}`.trim())
                .join('\n'),
            },
            {
              label: 'Difficulty mix',
              value: environment
                ? `${environment.difficulty.easy}% easy, ${environment.difficulty.medium}% medium, ${environment.difficulty.hard}% hard`
                : '',
            },
            {
              label: 'Caps by difficulty',
              value: (environment?.difficultyCaps ?? [])
                .map((row) => `${row.scope}: ${row.easy} / ${row.medium} / ${row.hard}`)
                .join('\n'),
            },
            {
              label: 'Collection plan',
              value: environment
                ? `${environment.planRequired ? 'Required' : 'Not required'}; changes within ${environment.changeWindowHours} hours`
                : '',
            },
            {
              label: 'Additional prohibited content',
              value: joinRows(environment?.extraProhibited),
            },
            {
              label: 'Additional excluded settings',
              value: joinRows(environment?.extraExcludedSettings),
            },
          ]}
        />
        <ReviewGroup
          title="Technical standards"
          editTo={stepPath(5)}
          rows={[
            {
              label: 'Requirements',
              value: (technical?.requirements ?? [])
                .map((row) => `${row.requirement}: ${row.specification}`)
                .join('\n'),
            },
            {
              label: 'Pre-collection materials',
              value: joinRows(technical?.preCollectionMaterials),
            },
            {
              label: 'Condition of payment',
              value: yesNo(technical?.conditionOfPayment),
            },
          ]}
        />
        <ReviewGroup
          title="Delivery & acceptance"
          editTo={stepPath(6)}
          rows={[
            { label: 'Delivery location', value: delivery?.storageLocation ?? '' },
            { label: 'Delivery cadence', value: delivery?.deliveryCadence ?? '' },
            { label: 'Manifest format', value: delivery?.manifestFormat ?? '' },
            {
              label: 'Required manifest fields',
              value: joinRows(delivery?.manifestFields),
            },
            { label: 'Site data fields', value: joinRows(delivery?.siteDataFields) },
            { label: 'Integrity and security', value: delivery?.integrityText ?? '' },
            {
              label: 'Acceptance',
              value: delivery
                ? `${delivery.reviewWindowDays}-day review; ${delivery.correctionDays} business days to correct; deemed acceptance ${delivery.deemedAcceptance ? 'yes' : 'no'}; rejected material ${delivery.rejectedStaysDeveloperOwned ? `developer-owned, deleted within ${delivery.deletionWindowDays} days` : 'remains work product'}`
                : '',
            },
          ]}
        />
        <ReviewGroup
          title="Customer (us)"
          rows={[
            { label: 'Legal name', value: organization?.legalName ?? '' },
            {
              label: 'Entity',
              value: organization
                ? `${organization.entityType}, ${organization.incorporationPlace}`
                : '',
            },
            { label: 'Governing law', value: organization?.governingLaw ?? '' },
            { label: 'Address', value: organization?.address ?? '' },
            { label: 'Short name', value: organization?.shortName ?? '' },
            {
              label: 'Signer',
              value: organization
                ? `${organization.signerName}, ${organization.signerTitle} · ${organization.signerEmail}`
                : '',
            },
          ]}
          note={
            organization ? (
              <>
                Organization details come from your saved organization info.{' '}
                <Link to="/organization" className="underline underline-offset-4">
                  Edit organization info
                </Link>
              </>
            ) : (
              <>
                Add your organization info before sending.{' '}
                <Link to="/organization" className="underline underline-offset-4">
                  Edit organization info
                </Link>
              </>
            )
          }
        />
        <ReviewGroup
          title="Developer"
          editTo={stepPath(7)}
          rows={
            counterparty?.mode === 'invite'
              ? [
                  { label: 'Email', value: counterparty.email },
                  {
                    label: 'Details',
                    value: 'Provided by the Developer when they sign',
                  },
                  { label: 'CC emails', value: counterparty.ccEmails },
                ]
              : [
                  { label: 'Legal name', value: counterparty?.legalName ?? '' },
                  { label: 'Entity', value: counterparty?.entityJurisdiction ?? '' },
                  { label: 'Address', value: counterparty?.address ?? '' },
                  { label: 'Short name', value: counterparty?.shortName ?? '' },
                  {
                    label: 'Signatory',
                    value: counterparty
                      ? [counterparty.signatoryName, counterparty.signatoryTitle]
                          .filter((part) => part !== '')
                          .join(', ')
                      : '',
                  },
                  { label: 'Email', value: counterparty?.email ?? '' },
                  { label: 'CC emails', value: counterparty?.ccEmails ?? '' },
                ]
          }
        />
      </WizardShell>
    </form>
  )
}
