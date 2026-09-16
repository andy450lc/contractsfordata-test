import { CheckIcon } from 'lucide-react'
import { useState, type ReactNode } from 'react'

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

export interface PickerOption {
  value: string
  label: string
  leading?: ReactNode
  keywords?: string
}

interface SearchPickerProps {
  id?: string
  value: string | string[]
  onChange: (value: string) => void
  options: PickerOption[]
  searchPlaceholder: string
  emptyText: string
  triggerLabel: string
  triggerClassName: string
  children: ReactNode
  ariaDescribedBy?: string
  customLabel?: (search: string) => string
  align?: 'start' | 'end'
  multiple?: boolean
}

// rankMatch scores an option against a search. Whole-label prefixes
// rank first, then word prefixes, then any substring.
export function rankMatch(value: string, search: string): number {
  const haystack = value.toLowerCase()
  const needle = search.trim().toLowerCase()
  if (needle === '') return 1
  const words = haystack.split(/\s+/)
  if (words.slice(1).join(' ').startsWith(needle) || haystack.startsWith(needle)) return 1
  if (words.some((word) => word.startsWith(needle))) return 0.75
  if (haystack.includes(needle)) return 0.5
  return 0
}

// SearchPicker is a button that opens a searchable list. The caller
// renders the button's contents and supplies the options.
export function SearchPicker({
  id,
  value,
  onChange,
  options,
  searchPlaceholder,
  emptyText,
  triggerLabel,
  triggerClassName,
  children,
  ariaDescribedBy,
  customLabel,
  align = 'start',
  multiple = false,
}: SearchPickerProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const typed = search.trim()
  const showCustom =
    customLabel !== undefined &&
    typed !== '' &&
    !options.some((option) => option.label.toLowerCase() === typed.toLowerCase())

  const selected = Array.isArray(value) ? value : [value]

  function select(next: string) {
    onChange(next)
    setSearch('')
    if (!multiple) setOpen(false)
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (!next) setSearch('')
      }}
    >
      <PopoverTrigger asChild>
        <button
          id={id}
          type="button"
          aria-haspopup="dialog"
          aria-expanded={open}
          aria-label={triggerLabel}
          aria-describedby={ariaDescribedBy}
          className={triggerClassName}
        >
          {children}
        </button>
      </PopoverTrigger>
      <PopoverContent align={align} className="w-80 p-0">
        <Command filter={rankMatch}>
          <CommandInput
            placeholder={searchPlaceholder}
            value={search}
            onValueChange={setSearch}
          />
          <CommandList>
            <CommandEmpty>{emptyText}</CommandEmpty>
            <CommandGroup>
              {showCustom ? (
                <CommandItem value={`custom:${typed}`} onSelect={() => select(typed)}>
                  <span className="flex-1 truncate">{customLabel(typed)}</span>
                </CommandItem>
              ) : null}
              {options.map((option) => (
                <CommandItem
                  key={option.value}
                  value={`${option.value} ${option.label} ${option.keywords ?? ''}`}
                  onSelect={() => select(option.value)}
                >
                  {option.leading}
                  <span className="flex-1 truncate">{option.label}</span>
                  {selected.includes(option.value) ? (
                    <CheckIcon className="size-4" />
                  ) : null}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
