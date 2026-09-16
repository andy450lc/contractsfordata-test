import { describe, expect, it } from 'vitest'

import { useSessionStore } from '@/stores/session'
import {
  recordedRequests,
  resetPasswordHandler,
  server,
  weakPasswordMessage,
} from '@/test/msw'
import { renderApp, screen, waitFor } from '@/test/render'

const email = 'ada@example.com'
const password = 'correct horse'

describe('forgot password page', () => {
  it('asks for an address and confirms without naming an account', async () => {
    const { user } = renderApp('/forgot-password')

    await user.type(await screen.findByLabelText(/^email$/i), email)
    await user.click(screen.getByRole('button', { name: /email me a reset link/i }))

    expect(await screen.findByText(/if that address has an account/i)).toBeInTheDocument()
    expect(screen.queryByLabelText(/^email$/i)).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: /back to sign in/i })).toHaveAttribute(
      'href',
      '/',
    )
    expect(recordedRequests()).toEqual([
      { path: '/v1/auth/forgot-password', body: { email } },
    ])
  })

  it('sends an already signed-in visitor to the dashboard', async () => {
    useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)

    const { router } = renderApp('/forgot-password')

    await screen.findByRole('heading', { name: /welcome/i })
    expect(router.state.location.pathname).toBe('/dashboard')
  })
})

describe('reset password page', () => {
  it('takes the token out of the address bar on mount', async () => {
    const { router } = renderApp('/reset-password?token=abc')

    expect(await screen.findByText(/choose a new password/i)).toBeInTheDocument()
    await waitFor(() => expect(router.state.location.search).toBe(''))
  })

  it('serves the link to a visitor who is already signed in', async () => {
    useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)
    server.use(resetPasswordHandler('session'))
    const { router, user } = renderApp('/reset-password?token=abc')

    expect(await screen.findByLabelText(/new password/i)).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/reset-password')

    await user.type(screen.getByLabelText(/new password/i), password)
    await user.click(screen.getByRole('button', { name: /save and sign in/i }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/dashboard'))
    expect(useSessionStore.getState().accessToken).toBe('test_access_token')
    expect(recordedRequests()).toEqual([
      { path: '/v1/auth/reset-password', body: { token: 'abc', password } },
    ])
  })

  it('sets the password, starts the session, and lands on the dashboard', async () => {
    server.use(resetPasswordHandler('session'))
    const { router, user } = renderApp('/reset-password?token=abc')

    await user.type(await screen.findByLabelText(/new password/i), password)
    await user.click(screen.getByRole('button', { name: /save and sign in/i }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/dashboard'))
    expect(useSessionStore.getState().status).toBe('authenticated')
    expect(recordedRequests()).toEqual([
      { path: '/v1/auth/reset-password', body: { token: 'abc', password } },
    ])
  })

  it('offers a fresh link when the token is refused', async () => {
    const { user } = renderApp('/reset-password?token=abc')

    await user.type(await screen.findByLabelText(/new password/i), password)
    await user.click(screen.getByRole('button', { name: /save and sign in/i }))

    expect(await screen.findByText(/this link is no longer valid/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /request a new link/i })).toHaveAttribute(
      'href',
      '/forgot-password',
    )
  })

  it('shows the same refusal when the link carries no token', async () => {
    renderApp('/reset-password')

    expect(await screen.findByText(/this link is no longer valid/i)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /request a new link/i })).toHaveAttribute(
      'href',
      '/forgot-password',
    )
    expect(recordedRequests()).toEqual([])
  })

  it('shows the provider message under the field and keeps the value', async () => {
    server.use(resetPasswordHandler('weak'))
    const { user } = renderApp('/reset-password?token=abc')

    await user.type(await screen.findByLabelText(/new password/i), password)
    await user.click(screen.getByRole('button', { name: /save and sign in/i }))

    expect(await screen.findByText(weakPasswordMessage)).toBeInTheDocument()
    expect(screen.getByLabelText(/new password/i)).toHaveValue(password)
  })

  it('blocks a short password before any request', async () => {
    const { user } = renderApp('/reset-password?token=abc')

    await user.type(await screen.findByLabelText(/new password/i), '1234567')
    await user.click(screen.getByRole('button', { name: /save and sign in/i }))

    expect(await screen.findByText('Use at least 8 characters')).toBeInTheDocument()
    expect(recordedRequests()).toEqual([])
  })
})
