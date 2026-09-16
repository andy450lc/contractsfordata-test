import { zodResolver } from '@hookform/resolvers/zod'
import { FormProvider, useForm, useWatch } from 'react-hook-form'

import { currencySymbol, formatMoney } from '../../currency'
import { formatNumber } from '../../number-format'
import {
  invoicingCadences,
  paymentMethods,
  pricingSchema,
  type Pricing,
} from '../../schemas'
import { ChoiceField } from '../choice-field'
import { FieldLabel, NumberField, TextField, ToggleField } from '../field'
import { MoneyField } from '../money-field'
import { RowsField } from '../rows-field'
import { WizardShell } from '../wizard-shell'
import type { StepProps } from './step-props'

interface StepPricingProps extends StepProps<Pricing> {
  effectiveDate: string
  targetVolume: number
  unitSingular: string
  unitPlural: string
}

export function StepPricing({
  step,
  defaultValues,
  onNext,
  onBack,
  effectiveDate,
  targetVolume,
  unitSingular,
  unitPlural,
}: StepPricingProps) {
  const form = useForm<Pricing>({
    resolver: zodResolver(pricingSchema(effectiveDate)),
    defaultValues,
  })
  const { control } = form
  const unitFee = useWatch({ control, name: 'unitFee' })
  const currency = useWatch({ control, name: 'currency' })
  const depositRequired = useWatch({ control, name: 'depositRequired' })
  const code = currency || 'USD'
  const hasTotal = Number.isFinite(unitFee) && Number.isFinite(targetVolume)
  const unit = unitSingular || 'unit'
  const units = unitPlural || 'units'

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onNext)} noValidate>
        <WizardShell
          step={step}
          heading="Set the price and the schedule"
          subtitle="Unit fee, payment terms, and delivery milestones."
          onBack={() => onBack(form.getValues())}
        >
          <MoneyField
            control={control}
            currencyName="currency"
            amountName="unitFee"
            label={`Fee per ${unit}`}
            required
            help="What you pay for each accepted unit. Pick the currency first, then enter the amount."
            placeholder="20.00"
          />

          <div className="flex flex-col gap-1 rounded-lg bg-muted/50 px-4 py-3">
            <div className="flex items-baseline justify-between gap-4">
              <FieldLabel help="The most you can be charged: the fee per unit times the target volume. You pay only for units you accept.">
                Maximum total
              </FieldLabel>
              <span className="text-base font-semibold">
                {hasTotal ? formatMoney(unitFee * targetVolume, code) : '—'}
              </span>
            </div>
            <p className="text-xs text-muted-foreground">
              {formatNumber(targetVolume) || '—'} {units} from the Scope step
              {hasTotal ? ` × ${formatMoney(unitFee, code)}` : ''}
            </p>
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <ToggleField
              className="sm:col-span-2"
              control={control}
              name="depositRequired"
              label="Deposit or advance"
              description="Whether you pay anything before work starts. Choose No to state that there is no deposit, advance, or equipment contribution."
            />
            {depositRequired ? (
              <NumberField
                control={control}
                name="depositAmount"
                label="Deposit amount"
                required
                prefix={currencySymbol(code)}
                help="Paid before collection begins and deducted from what you owe for accepted units."
              />
            ) : null}
          </div>

          <div className="grid gap-5 sm:grid-cols-2">
            <ChoiceField
              control={control}
              name="invoicingCadence"
              label="Invoicing cadence"
              required
              options={invoicingCadences}
              allowCustom
              help="How often the Developer may send you an invoice for accepted units."
            />
            <NumberField
              control={control}
              name="paymentTermsDays"
              label="Payment terms"
              required
              suffix="days after receipt"
              help="How many days you have to pay after receiving a correct invoice. Enter 30 for net 30."
            />
            <ChoiceField
              control={control}
              name="paymentMethods"
              label="Payment method"
              required
              options={paymentMethods}
              allowCustom
              help="How you will pay the Developer's invoices."
            />
            <TextField
              control={control}
              name="invoiceEmail"
              label="Invoice email"
              required
              type="email"
              help="The address the Developer sends invoices to."
              placeholder="billing@example.com"
            />
          </div>

          <RowsField<Pricing>
            name="milestones"
            label="Delivery milestones"
            required
            help="The delivery dates you both commit to. Mark one row as the final delivery. Its deadline is the end date of the engagement."
            columns={[
              {
                key: 'name',
                label: 'Milestone',
                type: 'textarea',
                placeholder: 'Final delivery',
                width: 150,
              },
              {
                key: 'volume',
                label: 'Target volume',
                type: 'textarea',
                placeholder: `300 ${units}`,
                width: 150,
              },
              { key: 'deadline', label: 'Deadline', type: 'date', width: 140 },
              {
                key: 'notes',
                label: 'Notes',
                type: 'textarea',
                placeholder: 'Daily after collection begins',
              },
              { key: 'isFinal', label: 'Final', type: 'radio', width: 56 },
            ]}
            emptyRow={{ name: '', volume: '', deadline: '', notes: '', isFinal: false }}
            addLabel="Add milestone"
            minRows={1}
          />
          <ToggleField
            className="sm:col-span-2"
            control={control}
            name="firmDeadline"
            label="Final deadline is firm"
            description="Choose Yes to make the final delivery date binding. If the Developer misses it you may reduce the scope or end the agreement, and any extension needs your written approval."
          />
        </WizardShell>
      </form>
    </FormProvider>
  )
}
