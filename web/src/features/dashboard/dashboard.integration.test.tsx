import { describe, expect, it } from 'vitest'

import { useSessionStore } from '@/stores/session'
import { defaultMe, meHandler, server } from '@/test/msw'
import { renderApp, screen } from '@/test/render'

function signIn() {
  useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)
}

describe('dashboard', () => {
  it('greets the signed-in user by name and shows the email', async () => {
    signIn()

    renderApp('/dashboard')

    expect(
      await screen.findByRole('heading', { name: `Welcome, ${defaultMe.name}` }),
    ).toBeInTheDocument()
    expect(screen.getByText(defaultMe.email)).toBeInTheDocument()
    expect(screen.getByText(/you're logged in/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign out/i })).toBeInTheDocument()
  })

  it('greets by email when the provider has no name', async () => {
    signIn()
    server.use(meHandler({ ...defaultMe, name: '' }))

    renderApp('/dashboard')

    expect(
      await screen.findByRole('heading', { name: `Welcome, ${defaultMe.email}` }),
    ).toBeInTheDocument()
  })
})
