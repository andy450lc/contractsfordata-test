import { describe, expect, it } from 'vitest'

import { ApiError, fieldErrorsFrom, shouldRetry } from './client'

describe('ApiError', () => {
  it('carries the status, code, and details of the error envelope', () => {
    const error = new ApiError(422, 'validation_failed', { any: 'shape' })

    expect(error.status).toBe(422)
    expect(error.code).toBe('validation_failed')
    expect(error.details).toEqual({ any: 'shape' })
    expect(error.message).toBe('API error 422: validation_failed')
  })
})

describe('fieldErrorsFrom', () => {
  it('returns the per-field messages a validation failure carries', () => {
    const details = [{ field: 'email', message: 'must be an email' }]

    expect(fieldErrorsFrom(new ApiError(400, 'invalid_request', details))).toEqual(
      details,
    )
  })

  it('returns an empty list for details that are not field errors', () => {
    expect(
      fieldErrorsFrom(new ApiError(400, 'invalid_request', { email: 'bad' })),
    ).toEqual([])
  })

  it('returns an empty list for an error the client did not raise', () => {
    expect(fieldErrorsFrom(new Error('network down'))).toEqual([])
  })
})

describe('shouldRetry', () => {
  it('stops after a 4xx answer', () => {
    expect(shouldRetry(0, new ApiError(404, 'not_found'))).toBe(false)
  })

  it('retries a 5xx answer twice', () => {
    const error = new ApiError(503, 'provider_unavailable')

    expect(shouldRetry(0, error)).toBe(true)
    expect(shouldRetry(1, error)).toBe(true)
    expect(shouldRetry(2, error)).toBe(false)
  })

  it('retries a network failure twice', () => {
    expect(shouldRetry(1, new TypeError('Failed to fetch'))).toBe(true)
    expect(shouldRetry(2, new TypeError('Failed to fetch'))).toBe(false)
  })
})
