import { http, HttpResponse } from 'msw'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  adoptSession,
  bootstrap,
  getAccessToken,
  googleSignInUrl,
  refresh,
  resetSession,
  signOut,
} from './session'

import { env } from '@/lib/env'
import { useSessionStore } from '@/stores/session'
import { refreshHandler, server } from '@/test/msw'

describe('session client', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  afterEach(() => {
    resetSession()
    vi.useRealTimers()
  })

  it('bootstraps into an authenticated session from the cookie', async () => {
    server.use(refreshHandler())

    await bootstrap()

    const state = useSessionStore.getState()
    expect(state.status).toBe('authenticated')
    expect(state.accessToken).toBe('test_access_token')
  })

  it('bootstraps into an anonymous session when refresh is refused', async () => {
    await bootstrap()

    expect(useSessionStore.getState().status).toBe('anonymous')
  })

  it('shares one in-flight refresh between concurrent callers', async () => {
    let calls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () => {
        calls += 1
        return HttpResponse.json({
          access_token: 'shared',
          expires_at: Date.now() + 300_000,
        })
      }),
    )

    await Promise.all([refresh(), refresh(), refresh()])

    expect(calls).toBe(1)
  })

  it('returns the current token while it is fresh and refreshes near expiry', async () => {
    let calls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () => {
        calls += 1
        return HttpResponse.json({
          access_token: `token-${calls}`,
          expires_at: Date.now() + 300_000,
        })
      }),
    )
    useSessionStore.getState().setAuthenticated('fresh', Date.now() + 300_000)

    expect(await getAccessToken()).toBe('fresh')
    expect(calls).toBe(0)

    useSessionStore.getState().setAuthenticated('stale', Date.now() + 30_000)

    expect(await getAccessToken()).toBe('token-1')
    expect(calls).toBe(1)
  })

  it('schedules a refresh sixty seconds before expiry, never sooner than five', async () => {
    let calls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () => {
        calls += 1
        return HttpResponse.json({
          access_token: `token-${calls}`,
          expires_at: Date.now() + 120_000,
        })
      }),
    )

    await refresh()
    expect(calls).toBe(1)

    await vi.advanceTimersByTimeAsync(59_000)
    expect(calls).toBe(1)

    await vi.advanceTimersByTimeAsync(2_000)
    expect(calls).toBe(2)

    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () => {
        calls += 1
        return HttpResponse.json({
          access_token: 'short',
          expires_at: Date.now() + 1_000,
        })
      }),
    )
    await refresh()
    const before = calls
    await vi.advanceTimersByTimeAsync(4_000)
    expect(calls).toBe(before)
    await vi.advanceTimersByTimeAsync(1_500)
    expect(calls).toBe(before + 1)
  })

  it('keeps a live session when the API is unreachable and retries later', async () => {
    useSessionStore.getState().setAuthenticated('live', Date.now() + 300_000)
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () => HttpResponse.error()),
    )

    const ok = await refresh()

    expect(ok).toBe(false)
    expect(useSessionStore.getState().status).toBe('authenticated')
    expect(useSessionStore.getState().accessToken).toBe('live')
  })

  it('adopts a session from a credential exchange and schedules its refresh', async () => {
    let calls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () => {
        calls += 1
        return HttpResponse.json({
          access_token: 'refreshed',
          expires_at: Date.now() + 300_000,
        })
      }),
    )
    window.localStorage.setItem('sow.signed-out', String(Date.now()))

    adoptSession({ access_token: 'minted', expires_at: Date.now() + 300_000 })

    const state = useSessionStore.getState()
    expect(state.status).toBe('authenticated')
    expect(state.accessToken).toBe('minted')
    expect(window.localStorage.getItem('sow.signed-out')).toBeNull()

    await vi.advanceTimersByTimeAsync(239_000)
    expect(calls).toBe(0)

    await vi.advanceTimersByTimeAsync(2_000)
    expect(calls).toBe(1)
    expect(useSessionStore.getState().accessToken).toBe('refreshed')
  })

  it('builds the Google sign-in URL through the API', () => {
    expect(googleSignInUrl('/agreements/7')).toBe(
      `${env.VITE_API_URL}/v1/auth/login?provider=google&returnTo=%2Fagreements%2F7`,
    )
    expect(googleSignInUrl('//evil')).toBe(
      `${env.VITE_API_URL}/v1/auth/login?provider=google&returnTo=%2Fdashboard`,
    )
  })

  it('signs out through the API and clears the session', async () => {
    let logoutCalls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/logout`, () => {
        logoutCalls += 1
        return new HttpResponse(null, { status: 204 })
      }),
    )
    useSessionStore.getState().setAuthenticated('live', Date.now() + 300_000)

    await signOut()

    expect(logoutCalls).toBe(1)
    expect(useSessionStore.getState().status).toBe('anonymous')
    expect(useSessionStore.getState().accessToken).toBeNull()
  })
})
