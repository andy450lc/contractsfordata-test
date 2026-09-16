import { describe, expect, it } from 'vitest'

import { singularize } from './units'

describe('singularize', () => {
  it('changes only the last word', () => {
    expect(singularize('accepted usable hours')).toBe('accepted usable hour')
    expect(singularize('clips')).toBe('clip')
    expect(singularize('sessions')).toBe('session')
    expect(singularize('categories')).toBe('category')
    expect(singularize('boxes')).toBe('box')
    expect(singularize('batches')).toBe('batch')
    expect(singularize('glass')).toBe('glass')
    expect(singularize('')).toBe('')
  })
})
