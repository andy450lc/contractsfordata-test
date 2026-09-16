import { useMutation, useQuery } from '@tanstack/react-query'

import { api, ApiError } from '@/lib/api/client'
import type { components } from '@/lib/api/schema'

type ApiErrorBody = components['schemas']['Error']
type AccessToken = components['schemas']['AccessToken']
type PendingVerification = components['schemas']['PendingVerification']
type Credentials = components['schemas']['Credentials']
type SignUpRequest = components['schemas']['SignUpRequest']
type VerifyEmailRequest = components['schemas']['VerifyEmailRequest']
type EmailRequest = components['schemas']['EmailRequest']
type ResetPasswordRequest = components['schemas']['ResetPasswordRequest']

export const authKeys = {
  me: ['auth', 'me'] as const,
}

// authMutationTimeoutMs bounds every auth mutation, so a hung request
// answers instead of leaving the submit button disabled indefinitely.
const authMutationTimeoutMs = 20_000

// authTimeoutSignal returns a signal that aborts after ms milliseconds.
function authTimeoutSignal(ms: number): AbortSignal {
  const controller = new AbortController()
  setTimeout(() => controller.abort(), ms)
  return controller.signal
}

// unwrap returns the body of an accepted auth call. A refusal becomes an
// ApiError carrying the code and the details the API sent.
function unwrap<T>(
  data: T | undefined,
  error: ApiErrorBody | undefined,
  response: Response,
): T {
  if (data !== undefined) return data
  throw new ApiError(response.status, error?.error ?? 'unknown_error', error?.details)
}

// unwrapAccepted checks an auth call that answers with no body.
function unwrapAccepted(error: ApiErrorBody | undefined, response: Response): void {
  if (response.ok) return
  throw new ApiError(response.status, error?.error ?? 'unknown_error', error?.details)
}

// useMe loads the caller's platform user record. The API client attaches
// the bearer token.
export function useMe(enabled = true) {
  return useQuery({
    queryKey: authKeys.me,
    enabled,
    queryFn: async () => {
      const { data, error, response } = await api.GET('/me')
      if (error || !data) {
        throw new ApiError(response.status, error?.error ?? 'unknown_error')
      }
      return data
    },
  })
}

// useSignIn exchanges a password for a session. An unverified address
// yields a pending verification.
export function useSignIn() {
  return useMutation<AccessToken | PendingVerification, Error, Credentials>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST('/auth/sign-in', {
        body,
        credentials: 'include',
        signal: authTimeoutSignal(authMutationTimeoutMs),
      })
      return unwrap(data, error, response)
    },
  })
}

// useSignUp creates an account and starts email verification. A
// provider that requires no verification yields a session.
export function useSignUp() {
  return useMutation<AccessToken | PendingVerification, Error, SignUpRequest>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST('/auth/sign-up', {
        body,
        credentials: 'include',
        signal: authTimeoutSignal(authMutationTimeoutMs),
      })
      return unwrap(data, error, response)
    },
  })
}

// useVerifyEmail completes sign-in or sign-up with the emailed code.
export function useVerifyEmail() {
  return useMutation<AccessToken, Error, VerifyEmailRequest>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST('/auth/verify-email', {
        body,
        credentials: 'include',
        signal: authTimeoutSignal(authMutationTimeoutMs),
      })
      return unwrap(data, error, response)
    },
  })
}

// useResendVerification asks for a fresh verification code.
export function useResendVerification() {
  return useMutation<void, Error, EmailRequest>({
    mutationFn: async (body) => {
      const { error, response } = await api.POST('/auth/resend-verification', {
        body,
        credentials: 'include',
        signal: authTimeoutSignal(authMutationTimeoutMs),
      })
      unwrapAccepted(error, response)
    },
  })
}

// useForgotPassword asks for a password reset link by email.
export function useForgotPassword() {
  return useMutation<void, Error, EmailRequest>({
    mutationFn: async (body) => {
      const { error, response } = await api.POST('/auth/forgot-password', {
        body,
        credentials: 'include',
        signal: authTimeoutSignal(authMutationTimeoutMs),
      })
      unwrapAccepted(error, response)
    },
  })
}

// useResetPassword sets a new password from a reset link and signs the
// user in.
export function useResetPassword() {
  return useMutation<AccessToken, Error, ResetPasswordRequest>({
    mutationFn: async (body) => {
      const { data, error, response } = await api.POST('/auth/reset-password', {
        body,
        credentials: 'include',
        signal: authTimeoutSignal(authMutationTimeoutMs),
      })
      return unwrap(data, error, response)
    },
  })
}
