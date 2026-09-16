import { zodResolver } from '@hookform/resolvers/zod'
import { FormProvider, useForm, useWatch } from 'react-hook-form'

import { deliveryCadences, deliverySchema, type Delivery } from '../../schemas'
import { ChoiceField } from '../choice-field'
import { NumberField, TextField, ToggleField } from '../field'
import { RowsField } from '../rows-field'
import { WizardShell } from '../wizard-shell'
import type { StepProps } from './step-props'

export function StepDelivery({
  step,
  defaultValues,
  onNext,
  onBack,
}: StepProps<Delivery>) {
  const form = useForm<Delivery>({ resolver: zodResolver(deliverySchema), defaultValues })
  const { control } = form
  const rejectedStays = useWatch({ control, name: 'rejectedStaysDeveloperOwned' })

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onNext)} noValidate>
        <WizardShell
          step={step}
          heading="Set delivery and acceptance"
          subtitle="Where deliveries go, what the manifest holds, and how acceptance works."
          onBack={() => onBack(form.getValues())}
        >
          <div className="grid gap-5 sm:grid-cols-[3fr_2fr]">
            <TextField
              control={control}
              name="storageLocation"
              label="Delivery location"
              required
              help="Where the Developer uploads finished material. Transfers are always encrypted and accessible only to the two parties."
              placeholder="Customer-designated cloud bucket"
            />
            <ChoiceField
              control={control}
              name="deliveryCadence"
              label="Delivery cadence"
              required
              options={deliveryCadences}
              allowCustom
              help="How often the Developer uploads finished material instead of holding it for a larger batch."
            />
          </div>
          <TextField
            control={control}
            name="manifestFormat"
            label="Manifest format"
            required
            multiline
            rows={3}
            help="The manifest is the index that comes with each delivery and lists every file in it. Describe the file format you want it in."
          />
          <RowsField<Delivery>
            name="manifestFields"
            label="Required manifest fields"
            required
            help="The information every entry in the manifest must include. A delivery with missing or inconsistent entries can be rejected."
            columns={[{ key: 'text', label: 'Field' }]}
            emptyRow={{ text: '' }}
            addLabel="Add field"
            minRows={1}
          />
          <RowsField<Delivery>
            name="siteDataFields"
            label="Site data fields"
            help="Details the Developer records about each location where material is captured."
            columns={[{ key: 'text', label: 'Field' }]}
            emptyRow={{ text: '' }}
            addLabel="Add field"
          />
          <TextField
            control={control}
            name="integrityText"
            label="Integrity and security requirements"
            required
            multiline
            rows={2}
            help="How the Developer proves files arrive complete and untampered, and how they keep them secure."
          />

          <div className="grid gap-5 sm:grid-cols-2">
            <NumberField
              control={control}
              name="reviewWindowDays"
              label="Acceptance review window"
              required
              suffix="days"
              help="How many days you have to accept or reject a delivery once it arrives complete with its manifest."
            />
            <NumberField
              control={control}
              name="correctionDays"
              label="Correction or replacement window"
              required
              suffix="business days"
              help="How long the Developer has to fix or replace material you reject."
            />
            <ToggleField
              className="sm:col-span-2"
              control={control}
              name="deemedAcceptance"
              label="Deemed acceptance after the review window"
              description="Choose Yes so that a delivery you do not reject within the review window counts as accepted, as long as it arrived complete."
            />
            <ToggleField
              className="sm:col-span-2"
              control={control}
              name="rejectedStaysDeveloperOwned"
              label="Rejected material stays developer-owned"
              description="Choose Yes so that material you reject in writing and the Developer does not fix stays theirs, and you delete your copies."
            />
            {rejectedStays ? (
              <NumberField
                control={control}
                name="deletionWindowDays"
                label="Deletion window for rejected material"
                required
                suffix="days"
                help="How long after rejecting material you have to delete your copies."
              />
            ) : null}
          </div>
        </WizardShell>
      </form>
    </FormProvider>
  )
}
