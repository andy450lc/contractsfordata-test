import { describe, expect, it } from 'vitest'

import { formatLongDate, formatTimestamp } from './dates'

describe('formatTimestamp', () => {
  it('renders a Unix-millisecond timestamp as a UTC date and time', () => {
    expect(formatTimestamp(1756200000000)).toBe('Aug 26, 2025, 9:20 AM')
  })

  it('renders the Unix epoch', () => {
    expect(formatTimestamp(0)).toBe('Jan 1, 1970, 12:00 AM')
  })
})

describe('formatLongDate', () => {
  it('spells out an ISO date as month, day, and year', () => {
    expect(formatLongDate('2026-09-06')).toBe('September 6, 2026')
    expect(formatLongDate('2026-01-01')).toBe('January 1, 2026')
  })

  it('returns an empty string for an empty value', () => {
    expect(formatLongDate('')).toBe('')
  })

  it('reads the date as UTC so the day never shifts', () => {
    expect(formatLongDate('2026-03-01')).toBe('March 1, 2026')
  })
})
