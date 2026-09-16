import { zodResolver } from '@hookform/resolvers/zod'
import { FormProvider, useForm } from 'react-hook-form'

import { scopeSchema, type Scope } from '../../schemas'
import { CountryField } from '../country-field'
import { TextField } from '../field'
import { RowsField } from '../rows-field'
import { VolumeField } from '../volume-field'
import { WizardShell } from '../wizard-shell'
import type { StepProps } from './step-props'

export function StepScope({ step, defaultValues, onNext, onBack }: StepProps<Scope>) {
  const form = useForm<Scope>({ resolver: zodResolver(scopeSchema), defaultValues })
  const { control } = form

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onNext)} noValidate>
        <WizardShell
          step={step}
          heading="Describe the work"
          subtitle="What is collected, where, and how much."
          onBack={() => onBack(form.getValues())}
        >
          <TextField
            control={control}
            name="deliverableDescription"
            label="Deliverable description"
            required
            help="What the Developer captures, in a few words. Include the format and point of view, for example 'stereo video from a head-mounted camera'."
            placeholder="stereo RGB, head-mounted first-person egocentric footage"
          />
          <VolumeField
            control={control}
            amountName="targetVolume"
            unitName="unit"
            label="Target volume"
            required
            help="How much you expect to receive over the engagement, and what you count it in. This unit is used for pricing and acceptance throughout."
            placeholder="300"
          />
          <div className="grid gap-5 sm:grid-cols-2">
            <CountryField
              control={control}
              name="regions"
              label="Collection countries"
              required
              help="The countries where the Developer may capture material. Anything captured elsewhere is out of scope."
            />
            <TextField
              control={control}
              name="venueConstraint"
              label="Allowed venues"
              required
              help="The kinds of places where capture is allowed, for example working businesses rather than homes or studios."
              placeholder="real, non-residential operating businesses"
            />
          </div>
          <RowsField<Scope>
            name="deliverables"
            label="Deliverables"
            required
            help="What the Developer hands over with every delivery. Add one item per row."
            columns={[{ key: 'text', label: 'Each delivery includes', type: 'textarea' }]}
            emptyRow={{ text: '' }}
            addLabel="Add deliverable"
            minRows={1}
          />
          <TextField
            control={control}
            name="notes"
            label="Additional notes"
            multiline
            help="Anything else the Developer should know about the scope, such as audio, language, or things to leave out."
            placeholder="No ambient audio unless approved in writing."
          />
        </WizardShell>
      </form>
    </FormProvider>
  )
}
