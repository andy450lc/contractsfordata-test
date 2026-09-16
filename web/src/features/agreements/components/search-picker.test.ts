import { describe, expect, it } from 'vitest'

import { rankMatch } from './search-picker'

describe('rankMatch', () => {
  it('ranks label prefixes above word prefixes above substrings', () => {
    expect(rankMatch('IN India', 'ind')).toBe(1)
    expect(rankMatch('IO British Indian Ocean Territory', 'ind')).toBe(0.75)
    expect(rankMatch('FI Finland', 'ind')).toBe(0)
    expect(rankMatch('GB United Kingdom', 'ngdom')).toBe(0.5)
    expect(rankMatch('INR Indian Rupee', 'inr')).toBe(1)
    expect(rankMatch('IN India', '')).toBe(1)
    expect(rankMatch('custom:clip', 'clip')).toBe(0.5)
  })
})
