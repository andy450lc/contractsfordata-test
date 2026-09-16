import { ChevronDownIcon } from 'lucide-react'
import { useState } from 'react'
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
} from 'react-hook-form'

import { cn } from '@/lib/utils'

import { currencyCodes, currencyName, currencySymbol } from '../currency'
import { formatNumber, parseNumberInput } from '../number-format'
import { FieldError, FieldLabel } from './field'
import { SearchPicker, type PickerOption } from './search-picker'

interface MoneyFieldProps<T extends FieldValues> {
  control: Control<T>
  currencyName: FieldPath<T>
  amountName: FieldPath<T>
  label: string
  required?: boolean
  help?: string
  placeholder?: string
}

// MoneyField is one input with a searchable currency picker on the left
// and the amount on the right. The picker shows each currency's symbol.
export function MoneyField<T extends FieldValues>({
  control,
  currencyName: currencyPath,
  amountName,
  label,
  required,
  help,
  placeholder,
}: MoneyFieldProps<T>) {
  const id = `field-${amountName}`
  const errorId = `${id}-error`

  return (
    <Controller
      control={control}
      name={currencyPath}
      render={({ field: currencyField }) => (
        <Controller
          control={control}
          name={amountName}
          render={({ field: amountField, fieldState }) => {
            const code =
              typeof currencyField.value === 'string' ? currencyField.value : 'USD'
            const error = fieldState.error?.message
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
                  <CurrencyPicker value={code} onChange={currencyField.onChange} />
                  <AmountInput
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

const currencyOptions: PickerOption[] = currencyCodes.map((code) => ({
  value: code,
  label: currencyName(code),
  leading: (
    <>
      <span className="w-8 font-medium">{currencySymbol(code)}</span>
      <span className="w-10">{code}</span>
    </>
  ),
}))

function CurrencyPicker({
  value,
  onChange,
}: {
  value: string
  onChange: (code: string) => void
}) {
  return (
    <SearchPicker
      value={value}
      onChange={onChange}
      options={currencyOptions}
      searchPlaceholder="Search currencies"
      emptyText="No currency found."
      triggerLabel={`Currency, ${value}`}
      triggerClassName="flex items-center gap-1.5 rounded-l-lg border-r border-input px-2.5 text-sm outline-none hover:bg-muted"
    >
      <span className="font-medium">{currencySymbol(value)}</span>
      <span className="text-muted-foreground">{value}</span>
      <ChevronDownIcon className="size-3.5 text-muted-foreground" />
    </SearchPicker>
  )
}

interface AmountInputProps {
  id: string
  placeholder?: string
  value: number | null
  onChange: (value: number | null) => void
  onBlur: () => void
  error?: string
  errorId: string
}

function AmountInput({
  id,
  placeholder,
  value,
  onChange,
  onBlur,
  error,
  errorId,
}: AmountInputProps) {
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
      className="min-w-0 flex-1 rounded-r-lg bg-transparent px-2.5 text-sm outline-none placeholder:text-muted-foreground"
    />
  )
}
