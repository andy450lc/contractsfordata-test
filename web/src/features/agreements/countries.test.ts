import { describe, expect, it } from 'vitest'

import {
  countriesSortedByName,
  countryFlag,
  countryName,
  isCountryCode,
} from './countries'

describe('country helpers', () => {
  it('names, flags, and validates codes', () => {
    expect(countryName('IN')).toBe('India')
    expect(countryName('')).toBe('')
    expect(countryName('zz')).toBe('zz')
    expect(countryFlag('IN')).toBe('🇮🇳')
    expect(countryFlag('')).toBe('')
    expect(isCountryCode('US')).toBe(true)
    expect(isCountryCode('XX')).toBe(false)
  })

  it('sorts the list by name', () => {
    expect(countriesSortedByName[0]?.name).toBe('Afghanistan')
    expect(countriesSortedByName.at(-1)?.name).toBe('Zimbabwe')
  })
})
