import { InfoIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import {
  Controller,
  type Control,
  type FieldErrors,
  type FieldPath,
  type FieldValues,
} from 'react-hook-form'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import { formatNumber, parseNumberInput } from '../number-format'
import { useState } from 'react'

interface FieldLabelProps {
  htmlFor?: string
  required?: boolean
  help?: string
  children: ReactNode
}

// FieldLabel shows the label, a required marker, and a help tooltip.
export function FieldLabel({ htmlFor, required, help, children }: FieldLabelProps) {
  return (
    <div className="flex items-center gap-1.5">
      <Label htmlFor={htmlFor} className="text-sm font-medium">
        {children}
        {required ? <span aria-hidden="true">*</span> : null}
      </Label>
      {help ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              className="inline-flex text-muted-foreground hover:text-foreground"
              aria-label="More info"
            >
              <InfoIcon className="size-3.5" />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top">{help}</TooltipContent>
        </Tooltip>
      ) : null}
    </div>
  )
}

// FieldError renders a validation message beneath a field.
export function FieldError({ id, message }: { id?: string; message?: string }) {
  if (!message) return null
  return (
    <p id={id} role="alert" className="text-xs text-destructive">
      {message}
    </p>
  )
}

// errorAt reads the message at a dotted path inside a form's errors.
export function errorAt(errors: FieldErrors, path: string): string | undefined {
  let node: unknown = errors
  for (const segment of path.split('.')) {
    if (node === null || typeof node !== 'object') return undefined
    node = (node as Record<string, unknown>)[segment]
  }
  if (node === null || typeof node !== 'object') return undefined
  const message = (node as { message?: unknown }).message
  return typeof message === 'string' ? message : undefined
}

interface BaseFieldProps<T extends FieldValues> {
  control: Control<T>
  name: FieldPath<T>
  label: string
  required?: boolean
  help?: string
  placeholder?: string
  className?: string
}

interface TextFieldProps<T extends FieldValues> extends BaseFieldProps<T> {
  multiline?: boolean
  type?: 'text' | 'email' | 'date' | 'tel'
  rows?: number
  maxRows?: number
}

// TextField is a labelled single-line or multi-line text input.
export function TextField<T extends FieldValues>({
  control,
  name,
  label,
  required,
  help,
  placeholder,
  multiline,
  type = 'text',
  rows = 2,
  maxRows = 5,
  className,
}: TextFieldProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => {
        const id = `field-${name}`
        const errorId = `${id}-error`
        const value = typeof field.value === 'string' ? field.value : ''
        const shared = {
          id,
          name: field.name,
          value,
          onChange: field.onChange,
          onBlur: field.onBlur,
          ref: field.ref,
          placeholder,
          'aria-invalid': fieldState.error ? true : undefined,
          'aria-describedby': fieldState.error ? errorId : undefined,
        }
        return (
          <div className={cn('flex flex-col gap-1.5', className)}>
            <FieldLabel htmlFor={id} required={required} help={help}>
              {label}
            </FieldLabel>
            {multiline ? (
              <Textarea
                rows={rows}
                className="resize-none overflow-y-auto field-sizing-content"
                style={{ maxHeight: `${maxRows * 1.5 + 1}rem` }}
                {...shared}
              />
            ) : (
              <Input type={type} {...shared} />
            )}
            <FieldError id={errorId} message={fieldState.error?.message} />
          </div>
        )
      }}
    />
  )
}

interface NumberFieldProps<T extends FieldValues> extends BaseFieldProps<T> {
  prefix?: string
  suffix?: string
}

// NumberField is a text input that formats digits with thousands
// separators and stores a number. An empty input stores NaN so the
// schema reports it as required.
export function NumberField<T extends FieldValues>({
  control,
  name,
  label,
  required,
  help,
  placeholder,
  prefix,
  suffix,
  className,
}: NumberFieldProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field, fieldState }) => (
        <NumberInput
          id={`field-${name}`}
          label={label}
          required={required}
          help={help}
          placeholder={placeholder}
          prefix={prefix}
          suffix={suffix}
          className={className}
          value={typeof field.value === 'number' ? field.value : null}
          onChange={(value) => field.onChange(value ?? Number.NaN)}
          onBlur={field.onBlur}
          error={fieldState.error?.message}
        />
      )}
    />
  )
}

interface NumberInputProps {
  id: string
  label: string
  required?: boolean
  help?: string
  placeholder?: string
  prefix?: string
  suffix?: string
  className?: string
  value: number | null
  onChange: (value: number | null) => void
  onBlur?: () => void
  error?: string
}

export function NumberInput({
  id,
  label,
  required,
  help,
  placeholder,
  prefix,
  suffix,
  className,
  value,
  onChange,
  onBlur,
  error,
}: NumberInputProps) {
  const normalized = value !== null && Number.isFinite(value) ? value : null
  const [text, setText] = useState(() => formatNumber(normalized))
  const [lastValue, setLastValue] = useState(normalized)
  if (!Object.is(normalized, lastValue)) {
    setLastValue(normalized)
    if (parseNumberInput(text).value !== normalized) setText(formatNumber(normalized))
  }
  const errorId = `${id}-error`

  return (
    <div className={cn('flex flex-col gap-1.5', className)}>
      <FieldLabel htmlFor={id} required={required} help={help}>
        {label}
      </FieldLabel>
      <div className="relative flex items-center">
        {prefix ? (
          <span className="pointer-events-none absolute left-2.5 text-sm text-muted-foreground">
            {prefix}
          </span>
        ) : null}
        <Input
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
          style={{
            paddingLeft: prefix ? `${prefix.length * 0.6 + 1.25}rem` : undefined,
            paddingRight: suffix ? `${suffix.length * 0.5 + 1.25}rem` : undefined,
          }}
        />
        {suffix ? (
          <span className="pointer-events-none absolute right-2.5 text-sm text-muted-foreground">
            {suffix}
          </span>
        ) : null}
      </div>
      <FieldError id={errorId} message={error} />
    </div>
  )
}

interface ChoiceCardOption {
  value: string
  title: string
  description?: string
}

interface ChoiceCardsProps<T extends FieldValues> {
  control: Control<T>
  name: FieldPath<T>
  label: string
  options: ChoiceCardOption[]
  columns?: 1 | 2
}

// ChoiceCards renders radio options as bordered cards, one per option.
export function ChoiceCards<T extends FieldValues>({
  control,
  name,
  label,
  options,
  columns = 2,
}: ChoiceCardsProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => (
        <RadioGroup
          value={typeof field.value === 'string' ? field.value : ''}
          onValueChange={field.onChange}
          aria-label={label}
          className={cn('grid gap-3', columns === 2 ? 'sm:grid-cols-2' : 'grid-cols-1')}
        >
          {options.map((option) => {
            const id = `field-${name}-${option.value}`
            const selected = field.value === option.value
            return (
              <label
                key={option.value}
                htmlFor={id}
                className={cn(
                  'flex cursor-pointer items-start gap-3 rounded-lg border p-3 text-sm transition-colors',
                  selected ? 'border-foreground' : 'border-border hover:bg-muted/50',
                )}
              >
                <RadioGroupItem id={id} value={option.value} className="mt-0.5" />
                <span className="flex flex-col gap-1">
                  <span className="font-medium">{option.title}</span>
                  {option.description ? (
                    <span className="text-muted-foreground">{option.description}</span>
                  ) : null}
                </span>
              </label>
            )
          })}
        </RadioGroup>
      )}
    />
  )
}

// FieldGroup titles a cluster of related fields.
export function FieldGroup({
  title,
  description,
  children,
  error,
}: {
  title: string
  description?: string
  children: ReactNode
  error?: string
}) {
  return (
    <fieldset className="flex flex-col gap-4 rounded-lg border p-4">
      <legend className="px-1 text-sm font-medium">{title}</legend>
      {description ? (
        <p className="-mt-2 text-sm text-muted-foreground">{description}</p>
      ) : null}
      {children}
      <FieldError message={error} />
    </fieldset>
  )
}

interface ToggleFieldProps<T extends FieldValues> {
  control: Control<T>
  name: FieldPath<T>
  label: string
  description?: string
  help?: string
  yesLabel?: string
  noLabel?: string
  className?: string
}

// ToggleField is a card with a label, a description, and a two-way
// Yes or No switch, stored as a boolean.
export function ToggleField<T extends FieldValues>({
  control,
  name,
  label,
  description,
  help,
  yesLabel = 'Yes',
  noLabel = 'No',
  className,
}: ToggleFieldProps<T>) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => {
        const options = [
          { value: true, text: yesLabel },
          { value: false, text: noLabel },
        ]
        return (
          <div
            className={cn(
              'flex items-start justify-between gap-4 rounded-lg border p-3',
              className,
            )}
          >
            <div className="flex min-w-0 flex-col gap-1">
              <FieldLabel help={help}>{label}</FieldLabel>
              {description ? (
                <p className="text-sm text-muted-foreground">{description}</p>
              ) : null}
            </div>
            <div
              role="radiogroup"
              aria-label={label}
              className="inline-flex shrink-0 rounded-lg bg-muted p-0.5"
            >
              {options.map((option) => {
                const selected = field.value === option.value
                return (
                  <button
                    key={option.text}
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    tabIndex={selected ? 0 : -1}
                    onClick={() => field.onChange(option.value)}
                    onKeyDown={(event) => {
                      if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
                        event.preventDefault()
                        field.onChange(!field.value)
                      }
                    }}
                    className={cn(
                      'rounded-md px-3 py-1 text-sm font-medium transition-colors outline-none focus-visible:ring-3 focus-visible:ring-ring/50',
                      selected && option.value
                        ? 'bg-background text-emerald-700 shadow-sm dark:text-emerald-300'
                        : selected
                          ? 'bg-background text-rose-700 shadow-sm dark:text-rose-300'
                          : 'text-muted-foreground hover:text-foreground',
                    )}
                  >
                    {option.text}
                  </button>
                )
              })}
            </div>
          </div>
        )
      }}
    />
  )
}
