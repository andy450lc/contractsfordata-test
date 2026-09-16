import { ChevronDownIcon } from 'lucide-react'
import { useState } from 'react'
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
} from 'react-hook-form'

import { cn } from '@/lib/utils'

import { formatNumber, parseNumberInput } from '../number-format'
import { unitPresets } from '../units'
import { FieldError, FieldLabel } from './field'
import { SearchPicker, type PickerOption } from './search-picker'

const presetOptions: PickerOption[] = unitPresets.map((unit) => ({
  value: unit,
  label: unit,
}))

interface VolumeFieldProps<T extends FieldValues> {
  control: Control<T>
  amountName: FieldPath<T>
  unitName: FieldPath<T>
  label: string
  required?: boolean
  help?: string
  placeholder?: string
}

// VolumeField is one input holding a quantity and, on its right, the
// unit it is counted in. The unit comes from a searchable list or is
// typed in.
export function VolumeField<T extends FieldValues>({
  control,
  amountName,
  unitName,
  label,
  required,
  help,
  placeholder,
}: VolumeFieldProps<T>) {
  const id = `field-${amountName}`
  const errorId = `${id}-error`

  return (
    <Controller
      control={control}
      name={unitName}
      render={({ field: unitField, fieldState: unitState }) => (
        <Controller
          control={control}
          name={amountName}
          render={({ field: amountField, fieldState }) => {
            const unit = typeof unitField.value === 'string' ? unitField.value : ''
            const error = fieldState.error?.message ?? unitState.error?.message
            const options = presetOptions.some((option) => option.value === unit)
              ? presetOptions
              : [{ value: unit, label: unit }, ...presetOptions]
            return (
              <div className="flex flex-col gap-1.5">
                <FieldLabel htmlFor={id} required={required} help={help}>
                  {label}
                </FieldLabel>
                <div
                  className={cn(
                    'flex h-8 w-full items-stretch rounded-lg border border-input transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50',
                    error && 'border-destructive ring-3 ring-destructive/20',
                  )}
                >
                  <QuantityInput
                    id={id}
                    placeholder={placeholder}
                    value={
                      typeof amountField.value === 'number' ? amountField.value : null
                    }
                    onChange={(value) => amountField.onChange(value ?? Number.NaN)}
                    onBlur={amountField.onBlur}
                    error={error}
                    errorId={errorId}
                  />
                  <SearchPicker
                    value={unit}
                    onChange={unitField.onChange}
                    options={options.filter((option) => option.value !== '')}
                    searchPlaceholder="Search or type a unit"
                    emptyText="Type a unit and press Enter."
                    customLabel={(search) => `Use "${search}"`}
                    align="end"
                    triggerLabel={`Unit, ${unit === '' ? 'none' : unit}`}
                    triggerClassName="flex max-w-[60%] items-center gap-1.5 rounded-r-lg border-l border-input px-2.5 text-sm outline-none hover:bg-muted"
                  >
                    <span
                      className={cn('truncate', unit === '' && 'text-muted-foreground')}
                    >
                      {unit === '' ? 'unit' : unit}
                    </span>
                    <ChevronDownIcon className="size-3.5 shrink-0 text-muted-foreground" />
                  </SearchPicker>
                </div>
                <FieldError id={errorId} message={error} />
              </div>
            )
          }}
        />
      )}
    />
  )
}

interface QuantityInputProps {
  id: string
  placeholder?: string
  value: number | null
  onChange: (value: number | null) => void
  onBlur: () => void
  error?: string
  errorId: string
}

function QuantityInput({
  id,
  placeholder,
  value,
  onChange,
  onBlur,
  error,
  errorId,
}: QuantityInputProps) {
  const normalized = value !== null && Number.isFinite(value) ? value : null
  const [text, setText] = useState(() => formatNumber(normalized))
  const [lastValue, setLastValue] = useState(normalized)
  if (!Object.is(normalized, lastValue)) {
    setLastValue(normalized)
    if (parseNumberInput(text).value !== normalized) setText(formatNumber(normalized))
  }

  return (
    <input
      id={id}
      inputMode="decimal"
      value={text}
      placeholder={placeholder}
      onChange={(event) => {
        const parsed = parseNumberInput(event.target.value)
        setText(parsed.text)
        setLastValue(parsed.value)
        onChange(parsed.value)
      }}
      onBlur={onBlur}
      aria-invalid={error ? true : undefined}
      aria-describedby={error ? errorId : undefined}
      className="min-w-0 flex-1 rounded-l-lg bg-transparent px-2.5 text-sm outline-none placeholder:text-muted-foreground"
    />
  )
}
