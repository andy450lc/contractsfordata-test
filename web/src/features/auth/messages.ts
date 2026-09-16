import { ApiError } from '@/lib/api/client'

// Notice is the alert one auth page shows above its form.
export interface Notice {
  title: string
  description?: string
}

const unavailable: Notice = {
  title: 'Something went wrong on our side',
  description: 'Please try again in a moment.',
}

const notices: Record<string, Notice> = {
  flow_incomplete: {
    title: "Sign-in didn't complete",
    description: "You can try again whenever you're ready.",
  },
  invalid_state: {
    title: 'That sign-in link expired',
    description: 'Please start again.',
  },
  provider_unavailable: unavailable,
  network: unavailable,
  session_expired: {
    title: 'You were signed out',
    description: 'Please sign in again.',
  },
  invalid_credentials: {
    title: "That email and password don't match.",
  },
  invalid_code: {
    title: "That code didn't work. It may have expired.",
  },
  invalid_reset_token: {
    title: 'This link is no longer valid',
    description: 'It may have expired or already been used.',
  },
}

// authNotice maps an error code to the notice the page shows. An unknown
// code gets the generic sign-in notice.
export function authNotice(code: string): Notice {
  return notices[code] ?? notices['flow_incomplete']!
}

// weakPasswordDetail reads the provider's reason out of a rejected
// password. Anything else yields nothing.
export function weakPasswordDetail(error: unknown): string | null {
  if (!(error instanceof ApiError) || error.code !== 'weak_password') return null
  const details = error.details
  if (details === null || typeof details !== 'object') return null
  const message = (details as { message?: unknown }).message
  return typeof message === 'string' ? message : null
}

// noticeFor picks the notice for a failed call. A refusal names its own
// code. An outage or an unreachable API reads as temporary.
export function noticeFor(error: unknown): Notice {
  if (error instanceof ApiError && error.status < 500) return authNotice(error.code)
  return authNotice('network')
}
