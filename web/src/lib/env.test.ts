import { describe, expect, it } from 'vitest'

import { parseEnv } from './env'

const valid = {
  VITE_API_URL: 'http://localhost:8080',
}

describe('parseEnv', () => {
  it('returns the typed values for a valid environment', () => {
    expect(parseEnv(valid)).toEqual(valid)
  })

  it('ignores variables the schema does not declare', () => {
    expect(parseEnv({ ...valid, MODE: 'test' })).toEqual(valid)
  })

  it('rejects a missing API URL and names the variable', () => {
    expect(() => parseEnv({})).toThrow(/VITE_API_URL/)
  })

  it('rejects an empty API URL', () => {
    expect(() => parseEnv({ VITE_API_URL: '' })).toThrow(/VITE_API_URL/)
  })

  it('rejects an API URL that is not a URL', () => {
    expect(() => parseEnv({ VITE_API_URL: 'not a url' })).toThrow(/VITE_API_URL/)
  })
})
