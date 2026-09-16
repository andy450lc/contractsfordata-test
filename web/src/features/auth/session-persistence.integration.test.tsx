import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import { env } from '@/lib/env'
import { useSessionStore } from '@/stores/session'
import { defaultMe, refreshHandler, server } from '@/test/msw'
import { renderApp, screen, waitFor } from '@/test/render'

describe('session persistence', () => {
  it('restores the session from the cookie on app load', async () => {
    server.use(refreshHandler())
    const { router } = renderApp('/dashboard')

    expect(await screen.findByRole('heading', { name: /welcome/i })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/dashboard')
    expect(useSessionStore.getState().status).toBe('authenticated')
  })

  it('refreshes once and retries once when a bearer call is refused', async () => {
    let meCalls = 0
    let refreshCalls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () => {
        refreshCalls += 1
        return HttpResponse.json({
          access_token: `token-${refreshCalls}`,
          expires_at: Date.now() + 300_000,
        })
      }),
      http.get(`${env.VITE_API_URL}/v1/me`, ({ request }) => {
        meCalls += 1
        if (request.headers.get('Authorization') !== 'Bearer token-2') {
          return HttpResponse.json({ error: 'unauthorized' }, { status: 401 })
        }
        return HttpResponse.json(defaultMe)
      }),
    )

    renderApp('/dashboard')

    expect(await screen.findByRole('heading', { name: /welcome/i })).toBeInTheDocument()
    expect(refreshCalls).toBe(2)
    expect(meCalls).toBe(2)
  })

  it('returns to sign-in with a notice when the session is revoked', async () => {
    server.use(
      http.get(`${env.VITE_API_URL}/v1/me`, () =>
        HttpResponse.json({ error: 'unauthorized' }, { status: 401 }),
      ),
    )
    let refreshCalls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () => {
        refreshCalls += 1
        if (refreshCalls === 1) {
          return HttpResponse.json({
            access_token: 'token-1',
            expires_at: Date.now() + 300_000,
          })
        }
        return HttpResponse.json({ error: 'unauthorized' }, { status: 401 })
      }),
    )

    const { router } = renderApp('/dashboard')

    await waitFor(() => expect(router.state.location.pathname).toBe('/'))
    expect(router.state.location.search).toBe('?error=session_expired')
    expect(await screen.findByRole('status')).toHaveTextContent(/you were signed out/i)
  })
})
