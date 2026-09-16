import { zodResolver } from '@hookform/resolvers/zod'
import { Controller, FormProvider, useForm } from 'react-hook-form'

import { environmentSchema, type Environment } from '../../schemas'
import { DifficultyMixField } from '../difficulty-mix-field'
import { FieldGroup, NumberField, ToggleField, errorAt } from '../field'
import { RowsField } from '../rows-field'
import { WizardShell } from '../wizard-shell'
import type { StepProps } from './step-props'

interface StepEnvironmentProps extends StepProps<Environment> {
  unitPlural: string
}

export function StepEnvironment({
  step,
  defaultValues,
  onNext,
  onBack,
  unitPlural,
}: StepEnvironmentProps) {
  const form = useForm<Environment>({
    resolver: zodResolver(environmentSchema),
    defaultValues,
  })
  const { control, formState } = form
  const unit = unitPlural || 'units'

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onNext)} noValidate>
        <WizardShell
          step={step}
          heading="Set the environment and task mix"
          subtitle="Which workplaces, how varied, and how hard."
          onBack={() => onBack(form.getValues())}
        >
          <RowsField<Environment>
            name="verticals"
            label="Collection verticals"
            required
            help="The kinds of businesses to capture, grouped by priority. For each group, give the industries, how much you want from them, and anything that must be included or left out."
            columns={[
              { key: 'verticals', label: 'Verticals', type: 'textarea' },
              { key: 'target', label: 'Priority and target', type: 'textarea' },
              { key: 'details', label: 'Required or excluded details', type: 'textarea' },
            ]}
            emptyRow={{ verticals: '', target: '', details: '' }}
            addLabel="Add tier"
            minRows={1}
          />

          <RowsField<Environment>
            name="limits"
            label="Limits"
            required
            help="Ceilings and floors on what you will accept, such as the most from a single industry or the fewest distinct sites. Give each a number and what it is counted in."
            columns={[
              {
                key: 'limit',
                label: 'Limit',
                type: 'textarea',
                placeholder: 'Maximum from any single category',
              },
              { key: 'value', label: 'Value', placeholder: '60', width: 110 },
              { key: 'unit', label: 'Unit', placeholder: unit, width: 190 },
            ]}
            emptyRow={{ limit: '', value: '', unit: '' }}
            addLabel="Add limit"
            minRows={1}
          />

          <FieldGroup
            title="Difficulty mix"
            description="The share of accepted units you want at each difficulty. Drag the handles or type a share. The three always add up to 100%."
          >
            <Controller
              control={control}
              name="difficulty"
              render={({ field }) => (
                <DifficultyMixField
                  value={field.value}
                  onChange={field.onChange}
                  error={errorAt(formState.errors, 'difficulty')}
                />
              )}
            />
          </FieldGroup>

          <RowsField<Environment>
            name="difficultyCaps"
            label="Caps by difficulty"
            help={`The most ${unit} you will accept of the same thing at each difficulty, so the collection stays varied. Say what each cap applies to, such as one operator, task, and site.`}
            columns={[
              {
                key: 'scope',
                label: 'Cap applies per',
                type: 'textarea',
                placeholder: 'Per operator, task, and site',
              },
              { key: 'easy', label: 'Easy', placeholder: '1', width: 100 },
              { key: 'medium', label: 'Medium', placeholder: '2', width: 100 },
              { key: 'hard', label: 'Hard', placeholder: '10', width: 100 },
            ]}
            emptyRow={{ scope: '', easy: '', medium: '', hard: '' }}
            addLabel="Add cap"
          />

          <div className="grid gap-5 sm:grid-cols-2">
            <ToggleField
              className="sm:col-span-2"
              control={control}
              name="planRequired"
              label="Collection plan required before filming"
              description="Choose Yes to require a written collection plan that you approve before any filming starts."
            />
            <NumberField
              control={control}
              name="changeWindowHours"
              label="Change implementation window"
              required
              suffix="hours"
              help="How quickly the Developer must carry out changes you ask for to the plan."
            />
          </div>

          <RowsField<Environment>
            name="extraProhibited"
            label="Additional prohibited content"
            help="Things that must never be captured or delivered, on top of the standard list: minors, bystanders who have not consented, passwords, financial or health data, other companies' confidential information, classified or controlled data, restricted technical data, precise locations, and restricted facilities."
            columns={[{ key: 'text', label: 'Must not be captured or delivered' }]}
            emptyRow={{ text: '' }}
            addLabel="Add item"
          />
          <RowsField<Environment>
            name="extraExcludedSettings"
            label="Additional excluded settings"
            help="Places and situations that are out of scope, on top of the standard list: homes, studios or staged scenes, long stretches of one repeated motion, and material already delivered to someone else."
            columns={[{ key: 'text', label: 'Setting' }]}
            emptyRow={{ text: '' }}
            addLabel="Add setting"
          />
        </WizardShell>
      </form>
    </FormProvider>
  )
}
