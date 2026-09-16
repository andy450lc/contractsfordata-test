// unitPresets lists the plural unit labels the volume picker offers.
// Any other label can be typed in.
export const unitPresets = [
  'accepted usable hours',
  'accepted hours',
  'hours',
  'minutes',
  'clips',
  'sessions',
  'recordings',
  'images',
  'frames',
  'items',
  'tasks',
  'documents',
  'pages',
  'samples',
  'days',
] as const

// singularize turns a plural unit label into its singular form by
// changing only the last word.
export function singularize(unit: string): string {
  const trimmed = unit.trim()
  if (trimmed === '') return ''
  const words = trimmed.split(/\s+/)
  const last = words[words.length - 1] ?? ''
  const lower = last.toLowerCase()
  let singular = last
  if (lower.endsWith('ies') && lower.length > 3) {
    singular = `${last.slice(0, -3)}y`
  } else if (/(ses|xes|zes|ches|shes)$/.test(lower)) {
    singular = last.slice(0, -2)
  } else if (lower.endsWith('s') && !lower.endsWith('ss')) {
    singular = last.slice(0, -1)
  }
  words[words.length - 1] = singular
  return words.join(' ')
}
