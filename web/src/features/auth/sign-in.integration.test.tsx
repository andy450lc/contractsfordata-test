import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { env } from '@/lib/env'
import { useSessionStore } from '@/stores/session'
import {
  defaultPendingToken,
  recordedRequests,
  server,
  signInHandler,
  verifyEmailHandler,
} from '@/test/msw'
import { act, renderApp, screen, userEvent, waitFor } from '@/test/render'

const credentials = { email: 'ada@example.com', password: 'correct horse' }

// submitCredentials fills the sign-in form and submits it.
async function submitCredentials(
  user: ReturnType<typeof userEvent.setup>,
  values = credentials,
) {
  await user.type(await screen.findByLabelText(/^email$/i), values.email)
  await user.type(screen.getByLabelText(/^password$/i), values.password)
  await user.click(screen.getByRole('button', { name: /^sign in$/i }))
}

describe('sign-in page', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the credential form, the Google link, and the ways out', async () => {
    renderApp('/?redirect=%2Fagreements%2F7')

    expect(await screen.findByRole('heading', { name: /sow/i })).toBeInTheDocument()
    expect(screen.getByLabelText(/^email$/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/^password$/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^sign in$/i })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /forgot password/i })).toHaveAttribute(
      'href',
      '/forgot-password',
    )
    expect(screen.getByRole('link', { name: /continue with google/i })).toHaveAttribute(
      'href',
      `${env.VITE_API_URL}/v1/auth/login?provider=google&returnTo=%2Fagreements%2F7`,
    )
    expect(screen.getByRole('link', { name: /create an account/i })).toHaveAttribute(
      'href',
      '/sign-up?redirect=%2Fagreements%2F7',
    )
  })

  it('offers a bare sign-up link when no destination was requested', async () => {
    renderApp('/')

    expect(
      await screen.findByRole('link', { name: /create an account/i }),
    ).toHaveAttribute('href', '/sign-up')
  })

  it('leaves for Google through an anchor carrying the destination', async () => {
    renderApp('/?redirect=%2Fagreements%2F7')

    const google = await screen.findByRole('link', { name: /continue with google/i })
    expect(google.tagName).toBe('A')
    const url = new URL(google.getAttribute('href') ?? '')
    expect(url.origin).toBe(new URL(env.VITE_API_URL).origin)
    expect(url.pathname).toBe('/v1/auth/login')
    expect(url.searchParams.get('provider')).toBe('google')
    expect(url.searchParams.get('returnTo')).toBe('/agreements/7')
    expect(recordedRequests()).toEqual([])
  })

  it('starts a session and lands on the requested destination', async () => {
    server.use(signInHandler('session'))
    const { router, user } = renderApp('/?redirect=%2Fagreements')

    await submitCredentials(user)

    await waitFor(() => expect(router.state.location.pathname).toBe('/agreements'))
    expect(useSessionStore.getState().status).toBe('authenticated')
    expect(useSessionStore.getState().accessToken).toBe('test_access_token')
    expect(recordedRequests()).toEqual([{ path: '/v1/auth/sign-in', body: credentials }])
  })

  it('moves an unverified address to the code step and finishes there', async () => {
    server.use(signInHandler('pending'), verifyEmailHandler('session'))
    const { router, user } = renderApp('/?redirect=%2Fagreements')

    await submitCredentials(user)

    expect(await screen.findByText(/check your inbox/i)).toBeInTheDocument()
    expect(screen.getByText(new RegExp(credentials.email, 'i'))).toBeInTheDocument()

    await user.type(screen.getByLabelText(/code/i), '123456')
    await user.click(screen.getByRole('button', { name: /^verify$/i }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/agreements'))
    expect(useSessionStore.getState().status).toBe('authenticated')
    expect(recordedRequests()).toEqual([
      { path: '/v1/auth/sign-in', body: credentials },
      {
        path: '/v1/auth/verify-email',
        body: { pending_token: defaultPendingToken, code: '123456' },
      },
    ])
  })

  it('shows one refusal for a wrong password and keeps the email', async () => {
    const { user } = renderApp('/')

    await submitCredentials(user)

    expect(await screen.findByRole('alert')).toHaveTextContent(
      /that email and password don't match/i,
    )
    expect(screen.getByLabelText(/^email$/i)).toHaveValue(credentials.email)
    expect(useSessionStore.getState().status).toBe('anonymous')
  })

  it('shows the unavailable notice when the provider is down and keeps both values', async () => {
    server.use(signInHandler('unavailable'))
    const { user } = renderApp('/')

    await submitCredentials(user)

    expect(await screen.findByRole('alert')).toHaveTextContent(
      /something went wrong on our side/i,
    )
    expect(screen.getByLabelText(/^email$/i)).toHaveValue(credentials.email)
    expect(screen.getByLabelText(/^password$/i)).toHaveValue(credentials.password)
  })

  it('blocks a malformed email and a short password before any request', async () => {
    const { user } = renderApp('/')

    await submitCredentials(user, { email: 'ada@', password: '1234567' })

    expect(await screen.findByText(/enter a valid email address/i)).toBeInTheDocument()
    expect(screen.getByText(/use at least 8 characters/i)).toBeInTheDocument()
    expect(recordedRequests()).toEqual([])
  })

  it('disables the submit button while the exchange is in flight', async () => {
    let release: () => void = () => undefined
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/sign-in`, async () => {
        await gate
        return HttpResponse.json({
          access_token: 'gated_token',
          expires_at: Date.now() + 300_000,
        })
      }),
    )
    const { router, user } = renderApp('/')

    await submitCredentials(user)

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /^sign in$/i })).toBeDisabled(),
    )

    release()
    await waitFor(() => expect(router.state.location.pathname).toBe('/dashboard'))
  })

  it('re-enables the button and shows the unavailable notice when the exchange never answers', async () => {
    server.use(
      http.post(`${env.VITE_API_URL}/v1/auth/sign-in`, async () => {
        await new Promise(() => undefined)
      }),
    )
    vi.useFakeTimers({ shouldAdvanceTime: true })
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
    renderApp('/')

    await submitCredentials(user)

    await waitFor(() =>
      expect(screen.getByRole('button', { name: /^sign in$/i })).toBeDisabled(),
    )

    await act(async () => {
      await vi.advanceTimersByTimeAsync(20_000)
    })

    expect(await screen.findByRole('alert')).toHaveTextContent(
      /something went wrong on our side/i,
    )
    expect(screen.getByRole('button', { name: /^sign in$/i })).toBeEnabled()
  })

  it.each([
    ['flow_incomplete', /sign-in didn't complete/i],
    ['invalid_state', /sign-in link expired/i],
    ['provider_unavailable', /something went wrong on our side/i],
    ['session_expired', /you were signed out/i],
    ['something_else', /sign-in didn't complete/i],
  ])('shows the notice for ?error=%s', async (code, notice) => {
    renderApp(`/?error=${code}`)

    expect(await screen.findByRole('status')).toHaveTextContent(notice)
  })

  it('sends an already signed-in visitor to their destination', async () => {
    useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)

    const { router } = renderApp('/?redirect=%2Fdashboard')

    await screen.findByRole('heading', { name: /welcome/i })
    expect(router.state.location.pathname).toBe('/dashboard')
  })
})
