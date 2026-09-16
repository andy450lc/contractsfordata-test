import { QueryClientProvider } from '@tanstack/react-query'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { VerifyCodeForm } from './verify-code-form'

import { queryClient } from '@/lib/query-client'
import { recordedRequests, server, verifyEmailHandler } from '@/test/msw'
import { act, render, screen, userEvent, waitFor } from '@/test/render'

const email = 'ada@example.com'
const pendingToken = 'pat_test_1'

// mountForm renders the code step with spies for its two callbacks.
function mountForm() {
  const onSignedIn = vi.fn()
  const onRestart = vi.fn()
  const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime })
  render(
    <QueryClientProvider client={queryClient}>
      <VerifyCodeForm
        email={email}
        pendingToken={pendingToken}
        onSignedIn={onSignedIn}
        onRestart={onRestart}
      />
    </QueryClientProvider>,
  )
  return { user, onSignedIn, onRestart }
}

describe('verify code form', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('names the address the code went to and offers every way forward', () => {
    mountForm()

    expect(screen.getByText(/check your inbox/i)).toBeInTheDocument()
    expect(screen.getByText(new RegExp(email, 'i'))).toBeInTheDocument()
    expect(screen.getByLabelText(/code/i)).toHaveAttribute('inputmode', 'numeric')
    expect(screen.getByRole('button', { name: /^verify$/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /send a new code/i })).toBeEnabled()
    expect(
      screen.getByRole('button', { name: /use a different email/i }),
    ).toBeInTheDocument()
  })

  it('blocks a five-digit code before any request', async () => {
    const { user } = mountForm()

    await user.type(screen.getByLabelText(/code/i), '12345')
    await user.click(screen.getByRole('button', { name: /^verify$/i }))

    expect(await screen.findByText(/enter the 6-digit code/i)).toBeInTheDocument()
    expect(recordedRequests()).toEqual([])
  })

  it('exchanges the code for a session and hands the token back', async () => {
    server.use(verifyEmailHandler('session'))
    const { user, onSignedIn } = mountForm()

    await user.type(screen.getByLabelText(/code/i), '123456')
    await user.click(screen.getByRole('button', { name: /^verify$/i }))

    await waitFor(() => expect(onSignedIn).toHaveBeenCalledTimes(1))
    expect(onSignedIn).toHaveBeenCalledWith({
      access_token: 'test_access_token',
      expires_at: expect.any(Number),
    })
    expect(recordedRequests()).toEqual([
      {
        path: '/v1/auth/verify-email',
        body: { pending_token: pendingToken, code: '123456' },
      },
    ])
  })

  it('refuses a stale code and clears the input', async () => {
    const { user } = mountForm()

    await user.type(screen.getByLabelText(/code/i), '123456')
    await user.click(screen.getByRole('button', { name: /^verify$/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      /that code didn't work\. it may have expired\./i,
    )
    expect(screen.getByLabelText(/code/i)).toHaveValue('')
  })

  it('sends a new code, then holds the button for thirty seconds', async () => {
    const { user } = mountForm()

    await user.click(screen.getByRole('button', { name: /send a new code/i }))

    expect(await screen.findByRole('status')).toHaveTextContent(/we sent a new code\./i)
    expect(recordedRequests()).toEqual([
      { path: '/v1/auth/resend-verification', body: { email } },
    ])
    expect(screen.getByRole('button', { name: /send a new code/i })).toBeDisabled()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(29_000)
    })
    expect(screen.getByRole('button', { name: /send a new code/i })).toBeDisabled()

    await act(async () => {
      await vi.advanceTimersByTimeAsync(2_000)
    })
    expect(screen.getByRole('button', { name: /send a new code/i })).toBeEnabled()
  })

  it('returns to the credential form on request', async () => {
    const { user, onRestart } = mountForm()

    await user.click(screen.getByRole('button', { name: /use a different email/i }))

    expect(onRestart).toHaveBeenCalledTimes(1)
  })
})
