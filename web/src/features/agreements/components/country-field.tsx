import { PlusIcon, XIcon } from 'lucide-react'
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
} from 'react-hook-form'

import { cn } from '@/lib/utils'

import { countriesSortedByName, countryFlag, countryName } from '../countries'
import { FieldError, FieldLabel } from './field'
import { SearchPicker, type PickerOption } from './search-picker'

const options: PickerOption[] = countriesSortedByName.map(({ code, name }) => ({
  value: code,
  label: name,
  leading: (
    <span className="w-7 text-base" aria-hidden="true">
      {countryFlag(code)}
    </span>
  ),
}))

interface CountryFieldProps<T extends FieldValues> {
  control: Control<T>
  name: FieldPath<T>
  label: string
  required?: boolean
  help?: string
  className?: string
}

// CountryField stores a list of ISO country codes. Countries are added
// from a searchable list with flags and removed from their chips.
export function CountryField<T extends FieldValues>({
  control,
  name,
  label,
  required,
  help,
  className,
}: CountryFieldProps<T>) {
  const id = `field-${name}`
  const errorId = `${id}-error`

  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => {
        const codes: string[] = Array.isArray(field.value) ? field.value : []
        const error = fieldState.error?.message

        function toggle(code: string) {
          field.onChange(
            codes.includes(code)
              ? codes.filter((item) => item !== code)
              : [...codes, code],
          )
        }

        return (
          <div className={cn('flex flex-col gap-1.5', className)}>
            <FieldLabel htmlFor={id} required={required} help={help}>
              {label}
            </FieldLabel>
            <div
              className={cn(
                'flex min-h-8 w-full flex-wrap items-center gap-1.5 rounded-lg border border-input px-1.5 py-1 transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50',
                error && 'border-destructive ring-3 ring-destructive/20',
              )}
            >
              {codes.map((code) => (
                <span
                  key={code}
                  className="flex items-center gap-1 rounded-md bg-muted py-0.5 pr-1 pl-1.5 text-sm"
                >
                  <span aria-hidden="true">{countryFlag(code)}</span>
                  <span>{countryName(code)}</span>
                  <button
                    type="button"
                    aria-label={`Remove ${countryName(code)}`}
                    onClick={() => toggle(code)}
                    className="rounded p-0.5 text-muted-foreground hover:bg-background hover:text-foreground"
                  >
                    <XIcon className="size-3" />
                  </button>
                </span>
              ))}
              <SearchPicker
                id={id}
                value={codes}
                onChange={toggle}
                options={options}
                multiple
                searchPlaceholder="Search countries"
                emptyText="No country found."
                triggerLabel={codes.length === 0 ? 'Add country' : 'Add another country'}
                ariaDescribedBy={error ? errorId : undefined}
                triggerClassName="flex h-6 items-center gap-1 rounded-md px-1.5 text-sm text-muted-foreground outline-none hover:bg-muted hover:text-foreground focus-visible:bg-muted"
              >
                <PlusIcon className="size-3.5" />
                {codes.length === 0 ? 'Add country' : 'Add'}
              </SearchPicker>
            </div>
            <FieldError id={errorId} message={error} />
          </div>
        )
      }}
    />
  )
}
