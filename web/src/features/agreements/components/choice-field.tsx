import { ChevronDownIcon } from 'lucide-react'
import {
  Controller,
  type Control,
  type FieldPath,
  type FieldValues,
} from 'react-hook-form'

import { cn } from '@/lib/utils'

import { FieldError, FieldLabel } from './field'
import { SearchPicker, type PickerOption } from './search-picker'

interface ChoiceFieldProps<T extends FieldValues> {
  control: Control<T>
  name: FieldPath<T>
  label: string
  options: readonly string[]
  required?: boolean
  help?: string
  placeholder?: string
  allowCustom?: boolean
  className?: string
}

// ChoiceField picks one text value from a short list. When custom
// values are allowed, anything typed into the search can be chosen.
export function ChoiceField<T extends FieldValues>({
  control,
  name,
  label,
  options,
  required,
  help,
  placeholder = 'Pick one',
  allowCustom = false,
  className,
}: ChoiceFieldProps<T>) {
  const id = `field-${name}`
  const errorId = `${id}-error`
  const presets: PickerOption[] = options.map((option) => ({
    value: option,
    label: option,
  }))

  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => {
        const raw = typeof field.value === 'string' ? field.value : ''
        const preset = presets.find(
          (option) => option.value.toLowerCase() === raw.toLowerCase(),
        )
        const value = preset?.value ?? raw
        const error = fieldState.error?.message
        const list =
          value === '' || preset ? presets : [{ value, label: value }, ...presets]
        return (
          <div className={cn('flex flex-col gap-1.5', className)}>
            <FieldLabel htmlFor={id} required={required} help={help}>
              {label}
            </FieldLabel>
            <SearchPicker
              id={id}
              value={value}
              onChange={field.onChange}
              options={list}
              searchPlaceholder={allowCustom ? 'Search or type your own' : 'Search'}
              emptyText={allowCustom ? 'Type a value and press Enter.' : 'No match.'}
              customLabel={allowCustom ? (search) => `Use "${search}"` : undefined}
              triggerLabel={`${label}, ${value === '' ? 'none' : value}`}
              ariaDescribedBy={error ? errorId : undefined}
              triggerClassName={cn(
                'flex h-8 w-full items-center gap-2 rounded-lg border border-input px-2.5 text-left text-sm outline-none transition-colors hover:bg-muted focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50',
                error && 'border-destructive ring-3 ring-destructive/20',
              )}
            >
              <span
                className={cn('flex-1 truncate', value === '' && 'text-muted-foreground')}
              >
                {value === '' ? placeholder : value}
              </span>
              <ChevronDownIcon className="size-4 shrink-0 text-muted-foreground" />
            </SearchPicker>
            <FieldError id={errorId} message={error} />
          </div>
        )
      }}
    />
  )
}
