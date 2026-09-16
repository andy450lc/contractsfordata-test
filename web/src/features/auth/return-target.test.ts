import { describe, expect, it } from 'vitest'

import { safeReturnTo } from './return-target'

describe('safeReturnTo', () => {
  it.each([
    ['/dashboard', '/dashboard'],
    ['/agreements/42?tab=files', '/agreements/42?tab=files'],
    ['//evil.example', '/dashboard'],
    ['https://evil.example', '/dashboard'],
    ['/a\\b', '/dashboard'],
    ['dashboard', '/dashboard'],
    [null, '/dashboard'],
    [42, '/dashboard'],
  ])('maps %j to %s', (input, expected) => {
    expect(safeReturnTo(input)).toBe(expected)
  })
})
