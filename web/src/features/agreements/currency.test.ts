import { describe, expect, it } from 'vitest'

import { currencyName, currencySymbol, formatMoney, isCurrencyCode } from './currency'

describe('currency helpers', () => {
  it('knows symbols and names', () => {
    expect(currencySymbol('USD')).toBe('$')
    expect(currencySymbol('INR')).toBe('₹')
    expect(currencySymbol('EUR')).toBe('€')
    expect(currencyName('INR')).toBe('Indian Rupee')
    expect(currencySymbol('nope')).toBe('nope')
  })

  it('formats money with the symbol', () => {
    expect(formatMoney(6000, 'USD')).toBe('$6,000.00')
    expect(formatMoney(6000, 'INR')).toBe('₹6,000.00')
    expect(formatMoney(null, 'USD')).toBe('')
    expect(formatMoney(5, 'nope')).toBe('nope 5.00')
  })

  it('validates codes', () => {
    expect(isCurrencyCode('GBP')).toBe(true)
    expect(isCurrencyCode('gbp')).toBe(false)
  })
})
