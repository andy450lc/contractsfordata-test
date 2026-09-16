import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import { env } from '@/lib/env'
import { useSessionStore } from '@/stores/session'
import { refreshHandler, server } from '@/test/msw'
import { renderApp, screen, waitFor } from '@/test/render'

describe('callback page', () => {
  it('refreshes the session and lands on the return target', async () => {
    server.use(refreshHandler())
    const { router } = renderApp('/callback?returnTo=%2Fdashboard')

    expect(await screen.findByRole('heading', { name: /welcome/i })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/dashboard')
    expect(useSessionStore.getState().status).toBe('authenticated')
  })

  it('falls back to the dashboard when the return target is unsafe', async () => {
    server.use(refreshHandler())
    const { router } = renderApp('/callback?returnTo=https%3A%2F%2Fevil.example')

    await screen.findByRole('heading', { name: /welcome/i })
    expect(router.state.location.pathname).toBe('/dashboard')
  })

  it('returns to sign-in with a notice when the refresh is refused', async () => {
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, () =>
        HttpResponse.json({ error: 'unauthorized' }, { status: 401 }),
      ),
    )

    const { router } = renderApp('/callback?returnTo=%2Fdashboard')

    await waitFor(() => expect(router.state.location.pathname).toBe('/'))
    expect(router.state.location.search).toBe('?error=flow_incomplete')
    expect(useSessionStore.getState().status).toBe('anonymous')
  })
})
