import { describe, expect, it } from 'vitest'

import { env } from '@/lib/env'
import { useSessionStore } from '@/stores/session'
import {
  defaultPendingToken,
  recordedRequests,
  server,
  signUpHandler,
  verifyEmailHandler,
  weakPasswordMessage,
} from '@/test/msw'
import { renderApp, screen, userEvent, waitFor } from '@/test/render'

const details = {
  firstName: 'Ada',
  lastName: 'Lovelace',
  email: 'ada@example.com',
  password: 'correct horse',
}

// submitDetails fills the sign-up form and submits it.
async function submitDetails(user: ReturnType<typeof userEvent.setup>, values = details) {
  await user.type(await screen.findByLabelText(/^first name$/i), values.firstName)
  await user.type(screen.getByLabelText(/^last name$/i), values.lastName)
  await user.type(screen.getByLabelText(/^email$/i), values.email)
  await user.type(screen.getByLabelText(/^password$/i), values.password)
  await user.click(screen.getByRole('button', { name: /create account/i }))
}

describe('sign-up page', () => {
  it('renders the account form, the Google link, and the way back', async () => {
    renderApp('/sign-up')

    expect(await screen.findByRole('heading', { name: /sow/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/^first name$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^last name$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^email$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument()
    expect(screen.getByText('At least 8 characters')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /create account/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /continue with google/i })).toHaveAttribute(
      'href',
      `${env.VITE_API_URL}/v1/auth/login?provider=google&returnTo=%2Fdashboard`,
    )
    expect(screen.getByRole('link', { name: /^sign in$/i })).toHaveAttribute('href', '/')
  })

  it('leaves for Google through an anchor bound for the dashboard', async () => {
    renderApp('/sign-up')

    const google = await screen.findByRole('link', { name: /continue with google/i })
    expect(google.tagName).toBe('A')
    const url = new URL(google.getAttribute('href') ?? '')
    expect(url.origin).toBe(new URL(env.VITE_API_URL).origin)
    expect(url.pathname).toBe('/v1/auth/login')
    expect(url.searchParams.get('provider')).toBe('google')
    expect(url.searchParams.get('returnTo')).toBe('/dashboard')
    expect(recordedRequests()).toEqual([])
  })

  it('blocks empty names, a malformed email, and a short password before any request', async () => {
    const { user } = renderApp('/sign-up')

    await submitDetails(user, {
      firstName: '  ',
      lastName: '  ',
      email: 'ada@',
      password: '1234567',
    })

    expect(await screen.findAllByText('Required')).toHaveLength(2)
    expect(screen.getByText(/enter a valid email address/i)).toBeInTheDocument()
    expect(screen.getByText('Use at least 8 characters')).toBeInTheDocument()
    expect(recordedRequests()).toEqual([])
  })

  it('creates the account, verifies the code, and lands on the dashboard', async () => {
    server.use(verifyEmailHandler('session'))
    const { router, user } = renderApp('/sign-up')

    await submitDetails(user, { ...details, firstName: ' Ada ', lastName: ' Lovelace ' })

    expect(await screen.findByText(/check your inbox/i)).toBeInTheDocument()
    expect(screen.getByText(new RegExp(details.email, 'i'))).toBeInTheDocument()

    await user.type(screen.getByLabelText(/code/i), '123456')
    await user.click(screen.getByRole('button', { name: /^verify$/i }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/dashboard'))
    expect(useSessionStore.getState().status).toBe('authenticated')
    expect(recordedRequests()).toEqual([
      {
        path: '/v1/auth/sign-up',
        body: {
          first_name: 'Ada',
          last_name: 'Lovelace',
          email: details.email,
          password: details.password,
        },
      },
      {
        path: '/v1/auth/verify-email',
        body: { pending_token: defaultPendingToken, code: '123456' },
      },
    ])
  })

  it('shows the provider message under the password and keeps every value', async () => {
    server.use(signUpHandler('weak'))
    const { user } = renderApp('/sign-up')

    await submitDetails(user)

    expect(await screen.findByText(weakPasswordMessage)).toBeInTheDocument()
    expect(screen.getByLabelText(/^first name$/i)).toHaveValue(details.firstName)
    expect(screen.getByLabelText(/^last name$/i)).toHaveValue(details.lastName)
    expect(screen.getByLabelText(/^email$/i)).toHaveValue(details.email)
    expect(screen.getByLabelText(/^password$/i)).toHaveValue(details.password)
  })

  it('sends an already signed-in visitor to the dashboard', async () => {
    useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)

    const { router } = renderApp('/sign-up')

    await screen.findByRole('heading', { name: /welcome/i })
    expect(router.state.location.pathname).toBe('/dashboard')
  })

  it('carries the requested destination on the Google link and the way back', async () => {
    renderApp('/sign-up?redirect=%2Fagreements%2F7')

    const google = await screen.findByRole('link', { name: /continue with google/i })
    expect(new URL(google.getAttribute('href') ?? '').searchParams.get('returnTo')).toBe(
      '/agreements/7',
    )
    expect(screen.getByRole('link', { name: /^sign in$/i })).toHaveAttribute(
      'href',
      '/?redirect=%2Fagreements%2F7',
    )
  })

  it('lands on the requested destination after the code step', async () => {
    server.use(verifyEmailHandler('session'))
    const { router, user } = renderApp('/sign-up?redirect=%2Fagreements')

    await submitDetails(user)
    expect(await screen.findByText(/check your inbox/i)).toBeInTheDocument()

    await user.type(screen.getByLabelText(/code/i), '123456')
    await user.click(screen.getByRole('button', { name: /^verify$/i }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/agreements'))
  })

  it('sends an already signed-in visitor to the requested destination', async () => {
    useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)

    const { router } = renderApp('/sign-up?redirect=%2Fagreements')

    await waitFor(() => expect(router.state.location.pathname).toBe('/agreements'))
  })

  it('adopts the session when the provider requires no verification', async () => {
    server.use(signUpHandler('session'))
    const { router, user } = renderApp('/sign-up?redirect=%2Fagreements')

    await submitDetails(user)

    await waitFor(() => expect(router.state.location.pathname).toBe('/agreements'))
    expect(useSessionStore.getState().status).toBe('authenticated')
    expect(useSessionStore.getState().accessToken).toBe('test_access_token')
    expect(screen.queryByText(/check your inbox/i)).not.toBeInTheDocument()
  })
})
