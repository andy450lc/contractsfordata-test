import { zodResolver } from '@hookform/resolvers/zod'
import { FormProvider, useForm, useWatch } from 'react-hook-form'

import { counterpartySchema, type Counterparty } from '../../schemas'
import { ChoiceCards, TextField } from '../field'
import { WizardShell } from '../wizard-shell'
import type { StepProps } from './step-props'

export function StepCounterparty({
  step,
  defaultValues,
  onNext,
  onBack,
}: StepProps<Counterparty>) {
  const form = useForm<Counterparty>({
    resolver: zodResolver(counterpartySchema),
    defaultValues,
  })
  const { control } = form
  const mode = useWatch({ control, name: 'mode' })

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onNext)} noValidate>
        <WizardShell
          step={step}
          heading="Developer information"
          subtitle="Who does the work, and where to send the signing request."
          onBack={() => onBack(form.getValues())}
        >
          <ChoiceCards
            control={control}
            name="mode"
            label="How do you want to add the Developer?"
            options={[
              { value: 'details', title: 'I have their details' },
              { value: 'invite', title: 'Invite by email only' },
            ]}
          />
          <TextField
            control={control}
            name="email"
            label="Developer email"
            required
            type="email"
            help="We send the signing link to this address."
            placeholder="jane@example.com"
          />
          {mode === 'details' ? (
            <>
              <TextField
                control={control}
                name="legalName"
                label="Developer legal name"
                required
                help="The Developer's full legal name, exactly as registered."
                placeholder="Acme Data Labs, Inc."
              />
              <div className="grid gap-5 sm:grid-cols-2">
                <TextField
                  control={control}
                  name="entityJurisdiction"
                  label="Entity type and jurisdiction"
                  required
                  help="The kind of entity and where it is registered, for example 'Delaware corporation' or 'private limited company in India'."
                  placeholder="Delaware corporation"
                />
                <TextField
                  control={control}
                  name="shortName"
                  label="Short name used in the document"
                  required
                  help="A short name to refer to the Developer throughout the document, usually the first word of their legal name."
                  placeholder="Acme"
                />
              </div>
              <TextField
                control={control}
                name="address"
                label="Developer address"
                required
                multiline
                rows={2}
                help="The Developer's registered or mailing address. Formal notices go here."
                placeholder="100 Main Street, Suite 200, Wilmington, Delaware 19801"
              />
              <div className="grid gap-5 sm:grid-cols-2">
                <TextField
                  control={control}
                  name="signatoryName"
                  label="Signatory's legal name"
                  required
                  help="The full name of the person who signs for the Developer."
                  placeholder="Jane Doe"
                />
                <TextField
                  control={control}
                  name="signatoryTitle"
                  label="Signatory's title"
                  help="That person's role at the company, for example 'President' or 'Director'."
                  placeholder="President"
                />
              </div>
            </>
          ) : (
            <p className="rounded-lg border px-3 py-2 text-sm text-muted-foreground">
              The developer will provide their name and details when they sign.
            </p>
          )}
          <TextField
            control={control}
            name="ccEmails"
            label="CC emails"
            help="These addresses receive a copy of the signed agreement once everyone has signed. Separate multiple addresses with commas."
            placeholder="cfo@example.com, counsel@example.com"
          />
        </WizardShell>
      </form>
    </FormProvider>
  )
}
