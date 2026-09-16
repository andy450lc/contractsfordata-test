import createClient from 'openapi-fetch'
import { z } from 'zod'

import { env } from '@/lib/env'

import type { paths } from './schema'

// apiFetch resolves the global fetch at request time, so request
// interception installed after module load still applies.
function apiFetch(request: Request): Promise<Response> {
  return globalThis.fetch(request)
}

// api is the typed client for the SoW API, generated from the OpenAPI
// contract. Requests send no cookies by default. The callback, refresh,
// logout, and six credential endpoints opt in per call.
export const api = createClient<paths>({
  baseUrl: `${env.VITE_API_URL}/v1`,
  fetch: apiFetch,
  credentials: 'omit',
})

// ApiError carries the API's error envelope with the response status.
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details: unknown

  constructor(status: number, code: string, details?: unknown) {
    super(`API error ${status}: ${code}`)
    this.status = status
    this.code = code
    this.details = details
  }
}

const fieldErrorsSchema = z.array(z.object({ field: z.string(), message: z.string() }))

export type ApiFieldError = z.infer<typeof fieldErrorsSchema>[number]

// fieldErrorsFrom reads the per-field messages a validation failure
// carries. Every other error yields an empty list.
export function fieldErrorsFrom(error: unknown): ApiFieldError[] {
  if (!(error instanceof ApiError)) return []

  const parsed = fieldErrorsSchema.safeParse(error.details)
  return parsed.success ? parsed.data : []
}

// shouldRetry is the query retry policy: a 4xx answer is final, other
// failures get two more attempts.
export function shouldRetry(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiError && error.status < 500) return false
  return failureCount < 2
}
