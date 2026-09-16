import { zodResolver } from '@hookform/resolvers/zod'
import { FormProvider, useForm } from 'react-hook-form'

import { technicalSchema, type Technical } from '../../schemas'
import { ToggleField } from '../field'
import { RowsField } from '../rows-field'
import { WizardShell } from '../wizard-shell'
import type { StepProps } from './step-props'

export function StepTechnical({
  step,
  defaultValues,
  onNext,
  onBack,
}: StepProps<Technical>) {
  const form = useForm<Technical>({
    resolver: zodResolver(technicalSchema),
    defaultValues,
  })

  return (
    <FormProvider {...form}>
      <form onSubmit={form.handleSubmit(onNext)} noValidate>
        <WizardShell
          step={step}
          heading="Set the technical standards"
          subtitle="What every delivery must meet before you accept it."
          onBack={() => onBack(form.getValues())}
        >
          <RowsField<Technical>
            name="requirements"
            label="Technical requirements"
            required
            help="Each row names a requirement, the standard the Developer must meet, and what causes material to be rejected."
            columns={[
              { key: 'requirement', label: 'Requirement', width: 180 },
              { key: 'specification', label: 'Default specification', type: 'textarea' },
              { key: 'rejection', label: 'Rejection criteria', type: 'textarea' },
            ]}
            emptyRow={{ requirement: '', specification: '', rejection: '' }}
            addLabel="Add requirement"
            minRows={1}
          />
          <RowsField<Technical>
            name="preCollectionMaterials"
            label="Pre-collection materials"
            help="Anything the Developer must give you before collection starts, such as sample footage or equipment details."
            columns={[{ key: 'text', label: 'Provided before collection' }]}
            emptyRow={{ text: '' }}
            addLabel="Add material"
          />
          <ToggleField
            className="sm:col-span-2"
            control={form.control}
            name="conditionOfPayment"
            label="Every technical item is a condition of payment"
            description="Choose Yes so that material missing any requirement does not count and is not billable."
          />
        </WizardShell>
      </form>
    </FormProvider>
  )
}
