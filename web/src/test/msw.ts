import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

import type { components } from '@/lib/api/schema'
import { env } from '@/lib/env'

type User = components['schemas']['User']
type AccessToken = components['schemas']['AccessToken']

// defaultMe is the user record the default handlers return.
export const defaultMe: User = {
  id: 'user_test_123',
  email: 'ada@example.com',
  name: 'Ada Lovelace',
  created_at: 1756200000000,
  updated_at: 1756250000000,
}

// defaultAccessToken mints a token that expires five minutes from now.
export const defaultAccessToken = (): AccessToken => ({
  access_token: 'test_access_token',
  expires_at: Date.now() + 300_000,
})

// refreshHandler answers the session refresh with a fresh access token.
// Tests that want a signed-in visitor install it with server.use.
export const refreshHandler = (token?: AccessToken) =>
  http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () =>
    HttpResponse.json(token ?? defaultAccessToken()),
  )

// anonymousRefreshHandler refuses the refresh, the default for a visitor
// with no session cookie.
export const anonymousRefreshHandler = () =>
  http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () =>
    HttpResponse.json({ error: 'unauthorized' }, { status: 401 }),
  )

// logoutHandler ends the session and answers with no content.
export const logoutHandler = () =>
  http.post(
    `${env.VITE_API_URL}/v1/auth/logout`,
    () => new HttpResponse(null, { status: 204 }),
  )

// meHandler returns the user for a bearer request and 401 without one.
export const meHandler = (user: User = defaultMe) =>
  http.get(`${env.VITE_API_URL}/v1/me`, ({ request }) => {
    if (!request.headers.get('Authorization')?.startsWith('Bearer ')) {
      return HttpResponse.json({ error: 'unauthorized' }, { status: 401 })
    }
    return HttpResponse.json(user)
  })

// defaultPendingToken is the pending verification token the sign-in and
// sign-up handlers hand back.
export const defaultPendingToken = 'pat_test_1'

// weakPasswordMessage is the provider text a weak_password answer carries.
export const weakPasswordMessage = 'Password is too weak'

// RecordedRequest is one auth call a handler received.
export interface RecordedRequest {
  path: string
  body: unknown
}

const recorded: RecordedRequest[] = []

// recordedRequests returns the auth calls the handlers received, oldest
// first. The test setup empties it between cases.
export const recordedRequests = (): RecordedRequest[] => recorded

// clearRecordedRequests drops every recorded call.
export const clearRecordedRequests = () => {
  recorded.length = 0
}

// record stores the path and the parsed body of one call.
const record = async (request: Request) => {
  const body: unknown = await request.json().catch(() => undefined)
  recorded.push({ path: new URL(request.url).pathname, body })
}

// errorBody builds the API error envelope with a correlation id.
const errorBody = (code: string, details?: unknown) => ({
  error: code,
  request_id: 'req_test',
  ...(details === undefined ? {} : { details }),
})

const weakPassword = () =>
  HttpResponse.json(errorBody('weak_password', { message: weakPasswordMessage }), {
    status: 422,
  })

const accepted = () => new HttpResponse(null, { status: 202 })

const pending = () =>
  HttpResponse.json({ pending_token: defaultPendingToken }, { status: 202 })

// signInHandler answers the credential exchange. The outcome picks the
// contract answer the case needs.
export const signInHandler = (
  outcome: 'session' | 'pending' | 'invalid' | 'unavailable' = 'invalid',
) =>
  http.post(`${env.VITE_API_URL}/v1/auth/sign-in`, async ({ request }) => {
    await record(request)
    if (outcome === 'session') return HttpResponse.json(defaultAccessToken())
    if (outcome === 'pending') return pending()
    if (outcome === 'unavailable') {
      return HttpResponse.json(errorBody('provider_unavailable'), { status: 503 })
    }
    return HttpResponse.json(errorBody('invalid_credentials'), { status: 401 })
  })

// signUpHandler answers account creation with a pending verification, a
// started session, or a rejected password.
export const signUpHandler = (outcome: 'pending' | 'session' | 'weak' = 'pending') =>
  http.post(`${env.VITE_API_URL}/v1/auth/sign-up`, async ({ request }) => {
    await record(request)
    if (outcome === 'session') return HttpResponse.json(defaultAccessToken())
    if (outcome === 'weak') return weakPassword()
    return pending()
  })

// verifyEmailHandler answers the emailed code with a session or a refusal.
export const verifyEmailHandler = (outcome: 'session' | 'invalid' = 'invalid') =>
  http.post(`${env.VITE_API_URL}/v1/auth/verify-email`, async ({ request }) => {
    await record(request)
    if (outcome === 'session') return HttpResponse.json(defaultAccessToken())
    return HttpResponse.json(errorBody('invalid_code'), { status: 400 })
  })

// resendHandler accepts a request for a fresh verification code.
export const resendHandler = () =>
  http.post(`${env.VITE_API_URL}/v1/auth/resend-verification`, async ({ request }) => {
    await record(request)
    return accepted()
  })

// forgotPasswordHandler accepts a request for a reset link.
export const forgotPasswordHandler = () =>
  http.post(`${env.VITE_API_URL}/v1/auth/forgot-password`, async ({ request }) => {
    await record(request)
    return accepted()
  })

// resetPasswordHandler answers the reset confirmation with a session, a
// rejected token, or a rejected password.
export const resetPasswordHandler = (
  outcome: 'session' | 'invalid' | 'weak' = 'invalid',
) =>
  http.post(`${env.VITE_API_URL}/v1/auth/reset-password`, async ({ request }) => {
    await record(request)
    if (outcome === 'session') return HttpResponse.json(defaultAccessToken())
    if (outcome === 'weak') return weakPassword()
    return HttpResponse.json(errorBody('invalid_reset_token'), { status: 400 })
  })

// Contract-conformant default handlers. Tests override per-case with
// server.use(...).
export const server = setupServer(
  anonymousRefreshHandler(),
  logoutHandler(),
  meHandler(),
  signInHandler(),
  signUpHandler(),
  verifyEmailHandler(),
  resendHandler(),
  forgotPasswordHandler(),
  resetPasswordHandler(),
)
