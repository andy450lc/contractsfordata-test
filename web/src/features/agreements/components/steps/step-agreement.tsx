import { zodResolver } from '@hookform/resolvers/zod'
import { ChevronDownIcon } from 'lucide-react'
import { useState } from 'react'
import { FormProvider, useForm } from 'react-hook-form'

import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { cn } from '@/lib/utils'

import { agreementTermsSchema, type AgreementTerms } from '../../schemas'
import { NumberField, TextField, ToggleField } from '../field'
import { WizardShell } from '../wizard-shell'
import type { StepProps } from './step-props'

export function StepAgreement({
  step,
  defaultValues,
  onNext,
  onBack,
  backTo,
}: StepProps<AgreementTerms>) {
  const form = useForm<AgreementTerms>({
    resolver: zodResolver(agreementTermsSchema),
    defaultValues,
  })
  const [termsOpen, setTermsOpen] = useState(false)
  const { control } = form

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onNext)} noValidate>
        <WizardShell
          step={step}
          heading="Name the agreement"
          subtitle="Give it a title, choose when it starts, and confirm the standard terms."
          onBack={() => onBack(form.getValues())}
          backTo={backTo}
        >
          <TextField
            control={control}
            name="title"
            label="SOW title"
            required
            help="A short name for this engagement. It appears at the top of the document and in your agreements list."
            placeholder="India Stereo Egocentric Pilot"
          />
          <TextField
            control={control}
            name="effectiveDate"
            label="Effective date"
            required
            type="date"
            help="The day the agreement comes into force. Every milestone deadline must be on or after this date."
            className="sm:w-1/2"
          />
          <ToggleField
            className="sm:col-span-2"
            control={control}
            name="exclusive"
            label="Exclusive engagement"
            description="The Developer may not reuse, resell, or license anything they produce for you, or anything substantially the same."
          />
          <TextField
            control={control}
            name="materialsSummary"
            label="Materials summary"
            required
            multiline
            rows={2}
            help="A short description of everything the Developer creates for you, in plain terms."
            placeholder="recordings, stereo video, IMU data, calibration data, metadata, and other materials"
          />

          <Collapsible
            open={termsOpen}
            onOpenChange={setTermsOpen}
            className="rounded-lg border"
          >
            <CollapsibleTrigger className="flex w-full items-center justify-between px-4 py-3 text-left text-sm font-medium">
              <span>
                Standard terms
                <span className="ml-2 font-normal text-muted-foreground">
                  Prefilled with our standard values. Change them only if you have agreed
                  different terms.
                </span>
              </span>
              <ChevronDownIcon
                className={cn('size-4 transition-transform', termsOpen && 'rotate-180')}
              />
            </CollapsibleTrigger>
            <CollapsibleContent className="grid gap-5 border-t px-4 py-4 sm:grid-cols-2">
              <NumberField
                control={control}
                name="incidentNoticeHours"
                label="Security incident notice"
                required
                suffix="hours"
                help="How soon the Developer must tell you about a security incident involving your data."
              />
              <NumberField
                control={control}
                name="cureDays"
                label="Breach cure period"
                required
                suffix="days"
                help="How long a party has to fix a serious breach after written notice, before the other party may end the agreement."
              />
              <NumberField
                control={control}
                name="convenienceNoticeDays"
                label="Termination for convenience notice"
                required
                suffix="days"
                help="How much written notice you must give to end the agreement when the Developer is not at fault."
              />
              <NumberField
                control={control}
                name="recordsRetentionYears"
                label="Compliance records retention"
                required
                suffix="years"
                help="How long after final delivery the Developer must keep consent forms, permissions, and delivery records."
              />
              <NumberField
                control={control}
                name="liabilityLookbackMonths"
                label="Liability look-back period"
                required
                suffix="months"
                help="Each party's liability is capped at the fees paid during this many months before a claim."
              />
              <NumberField
                control={control}
                name="excludedClaimsCapMultiplier"
                label="Excluded-claims cap multiplier"
                required
                suffix="× fees"
                help="For the most serious claims, the Developer's liability is capped at this multiple of the fees paid in the look-back period."
              />
              <NumberField
                control={control}
                name="dataClaimsCapFloor"
                label="Data and IP claims cap floor"
                required
                prefix="$"
                help="The lowest the cap can be for claims about your data, intellectual property, or privacy, even if the multiplier gives a smaller number."
              />
            </CollapsibleContent>
          </Collapsible>
        </WizardShell>
      </form>
    </FormProvider>
  )
}
