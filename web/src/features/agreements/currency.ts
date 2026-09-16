// currencyCodes lists the ISO 4217 codes the picker offers.
export const currencyCodes = [
  'USD',
  'EUR',
  'GBP',
  'INR',
  'JPY',
  'CNY',
  'CAD',
  'AUD',
  'SGD',
  'AED',
  'CHF',
  'HKD',
  'KRW',
  'BRL',
  'MXN',
  'SEK',
  'NOK',
  'DKK',
  'NZD',
  'ZAR',
  'PLN',
  'IDR',
  'PHP',
  'THB',
  'VND',
  'TRY',
  'SAR',
  'ILS',
  'NGN',
  'KES',
] as const

export type CurrencyCode = (typeof currencyCodes)[number]

const displayNames = new Intl.DisplayNames('en', { type: 'currency' })

export function isCurrencyCode(value: string): value is CurrencyCode {
  return (currencyCodes as readonly string[]).includes(value)
}

// currencySymbol returns the narrow symbol for a code, or the code
// itself when the runtime has no symbol for it.
export function currencySymbol(code: string): string {
  try {
    const part = new Intl.NumberFormat('en', {
      style: 'currency',
      currency: code,
      currencyDisplay: 'narrowSymbol',
    })
      .formatToParts(0)
      .find((item) => item.type === 'currency')
    return part?.value ?? code
  } catch {
    return code
  }
}

// currencyName returns the English name of a currency.
export function currencyName(code: string): string {
  try {
    return displayNames.of(code) ?? code
  } catch {
    return code
  }
}

// formatMoney renders an amount with its currency symbol and two
// decimals, or an empty string when there is no amount.
export function formatMoney(value: number | null | undefined, code: string): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return ''
  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: code,
      currencyDisplay: 'narrowSymbol',
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(value)
  } catch {
    return `${code} ${value.toLocaleString('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })}`
  }
}
