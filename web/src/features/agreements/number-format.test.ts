import { describe, expect, it } from 'vitest'

import { formatNumber, parseNumberInput } from './number-format'

describe('parseNumberInput', () => {
  it('groups thousands and returns the number', () => {
    expect(parseNumberInput('100000')).toEqual({ text: '100,000', value: 100000 })
  })

  it('keeps a trailing decimal point while typing', () => {
    expect(parseNumberInput('20.')).toEqual({ text: '20.', value: 20 })
    expect(parseNumberInput('20.5')).toEqual({ text: '20.5', value: 20.5 })
  })

  it('ignores letters and extra separators', () => {
    expect(parseNumberInput('1,2a3')).toEqual({ text: '123', value: 123 })
  })

  it('maps an empty input to null', () => {
    expect(parseNumberInput('')).toEqual({ text: '', value: null })
  })
})

describe('formatNumber', () => {
  it('formats stored numbers for display', () => {
    expect(formatNumber(4.5)).toBe('4.5')
    expect(formatNumber(Number.NaN)).toBe('')
  })
})
