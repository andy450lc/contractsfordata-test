// parseNumberInput reads what a person typed into a number field. It
// keeps digits and one decimal point and returns the display text with
// thousands separators plus the numeric value. An empty input maps to
// null.
export function parseNumberInput(raw: string): { text: string; value: number | null } {
  const cleaned = raw.replace(/[^\d.]/g, '')
  if (cleaned === '') return { text: '', value: null }

  const [wholePart = '', ...rest] = cleaned.split('.')
  const hasDot = rest.length > 0
  const fraction = rest.join('').slice(0, 4)
  const whole = wholePart.replace(/^0+(?=\d)/, '')
  const grouped = whole === '' ? '0' : whole.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
  const text = hasDot ? `${grouped}.${fraction}` : grouped
  const value = Number(`${whole === '' ? '0' : whole}${hasDot ? `.${fraction}` : ''}`)
  return { text, value }
}

// formatNumber renders a stored number the way the input shows it.
export function formatNumber(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return ''
  return value.toLocaleString('en-US', { maximumFractionDigits: 4 })
}
