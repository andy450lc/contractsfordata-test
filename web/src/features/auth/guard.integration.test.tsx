import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import { env } from '@/lib/env'
import { useSessionStore } from '@/stores/session'
import { refreshHandler, server } from '@/test/msw'
import { renderApp, screen, waitFor } from '@/test/render'

describe('route guard', () => {
  it('sends an anonymous visitor to sign-in with the destination preserved', async () => {
    const { router } = renderApp('/dashboard?tab=files')

    await waitFor(() => expect(router.state.location.pathname).toBe('/'))
    expect(router.state.location.search).toBe('?redirect=%2Fdashboard%3Ftab%3Dfiles')
    expect(await screen.findByRole('button', { name: /^sign in$/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /continue with google/i })).toHaveAttribute(
      'href',
      `${env.VITE_API_URL}/v1/auth/login?provider=google&returnTo=%2Fdashboard%3Ftab%3Dfiles`,
    )
  })

  it('shows the loader while the session is still unknown', async () => {
    let release: () => void = () => undefined
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/refresh`, async () => {
        await gate
        return HttpResponse.json({ error: 'unauthorized' }, { status: 401 })
      }),
    )

    const { router } = renderApp('/dashboard')

    expect(await screen.findByRole('status')).toHaveTextContent(/loading/i)
    expect(router.state.location.pathname).toBe('/dashboard')

    release()
    await waitFor(() => expect(router.state.location.pathname).toBe('/'))
  })

  it('renders the page for a signed-in visitor', async () => {
    server.use(refreshHandler())

    const { router } = renderApp('/dashboard')

    expect(await screen.findByRole('heading', { name: /welcome/i })).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/dashboard')
  })

  it('signs out through the API, clears the session, and returns to sign-in', async () => {
    let logoutCalls = 0
    server.use(
      refreshHandler(),
      http.post(`${env.VITE_API_URL}/v1/auth/logout`, ({ request }) => {
        logoutCalls += 1
        expect(request.credentials).toBe('include')
        return new HttpResponse(null, { status: 204 })
      }),
    )
    const { router, user } = renderApp('/dashboard')
    await screen.findByRole('heading', { name: /welcome/i })

    await user.click(screen.getByRole('button', { name: /sign out/i }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/'))
    expect(logoutCalls).toBe(1)
    expect(useSessionStore.getState().status).toBe('anonymous')
    expect(window.localStorage.getItem('sow.signed-out')).not.toBeNull()
    expect(window.localStorage.getItem('sow.signed-out')).not.toMatch(/token/)
  })

  it('clears the session when another tab signs out', async () => {
    server.use(refreshHandler())
    const { router } = renderApp('/dashboard')
    await screen.findByRole('heading', { name: /welcome/i })

    window.dispatchEvent(
      new StorageEvent('storage', {
        key: 'sow.signed-out',
        newValue: String(Date.now()),
      }),
    )

    await waitFor(() => expect(router.state.location.pathname).toBe('/'))
    expect(useSessionStore.getState().status).toBe('anonymous')
  })
})
