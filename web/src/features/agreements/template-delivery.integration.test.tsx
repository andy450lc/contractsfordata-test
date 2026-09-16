import { act, fireEvent } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { env } from '@/lib/env'
import { useSessionStore } from '@/stores/session'
import { renderApp, screen, waitFor, within } from '@/test/render'
import { defaultMe, recordedRequests, server } from '@/test/msw'
import { objectUrls } from '@/test/setup'

import { seedSourceDocument, sourceSteps } from './fixtures/source-document'
import { useAgreementsStore } from './store'

const disclaimerOne =
  'This template is provided for general reference only and does not constitute legal advice. It has not been reviewed by a licensed attorney for your specific situation, and no attorney-client relationship is created by downloading or using it.'
const disclaimerTwo =
  'Data-collection and data-transfer laws vary by jurisdiction and change over time. You are responsible for reviewing, adapting, and validating this document — including consulting a lawyer licensed in each relevant jurisdiction — before relying on it or using it in any transaction.'
const acknowledgment =
  'I acknowledge that this is not legal advice, and that I am responsible for my own legal review before use.'

let clientHeight = 100
let scrollHeight = 300
let notifyResize: () => void = () => undefined
const defaultResizeObserver = window.ResizeObserver

beforeEach(() => {
  window.localStorage.clear()
  useAgreementsStore.setState({ organization: null, drafts: {} })
  useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)
  clientHeight = 100
  scrollHeight = 300
  vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockImplementation(
    () => clientHeight,
  )
  vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockImplementation(
    () => scrollHeight,
  )
  notifyResize = () => undefined
})

afterEach(() => {
  Object.defineProperty(window, 'ResizeObserver', {
    configurable: true,
    value: defaultResizeObserver,
  })
})

async function openFromReview() {
  const id = seedSourceDocument()
  const rendered = renderApp(`/agreements/${id}/edit?step=8`)
  const trigger = await screen.findByRole('button', { name: 'Get Word document' })
  await rendered.user.click(trigger)
  return { ...rendered, id, trigger }
}

async function acknowledgeAndContinue(user: ReturnType<typeof renderApp>['user']) {
  const disclaimer = screen.getByLabelText('Legal disclaimer')
  Object.defineProperty(disclaimer, 'scrollTop', {
    configurable: true,
    writable: true,
    value: 198,
  })
  fireEvent.scroll(disclaimer)
  const checkbox = screen.getByRole('checkbox', { name: acknowledgment })
  expect(checkbox).toBeEnabled()
  await user.click(checkbox)
  await user.click(screen.getByRole('button', { name: 'Continue' }))
}

describe('template delivery gate', () => {
  it('uses the exact disclaimer, measures the real bottom, and resets each attempt', async () => {
    const { user, trigger } = await openFromReview()
    const dialog = screen.getByRole('dialog', { name: 'Before You Download' })

    expect(within(dialog).getByText(disclaimerOne)).toBeInTheDocument()
    expect(within(dialog).getByText(disclaimerTwo)).toBeInTheDocument()
    expect(
      within(dialog).getByText('Scroll to the bottom to enable the acknowledgment.'),
    ).toBeInTheDocument()
    const checkbox = within(dialog).getByRole('checkbox', { name: acknowledgment })
    expect(checkbox).toHaveAttribute('type', 'checkbox')
    expect(checkbox).toBeDisabled()
    expect(within(dialog).getByRole('button', { name: 'Continue' })).toBeDisabled()

    const disclaimer = within(dialog).getByLabelText('Legal disclaimer')
    Object.defineProperty(disclaimer, 'scrollTop', {
      configurable: true,
      writable: true,
      value: 197,
    })
    fireEvent.scroll(disclaimer)
    expect(checkbox).toBeDisabled()
    Object.defineProperty(disclaimer, 'scrollTop', {
      configurable: true,
      writable: true,
      value: 198,
    })
    fireEvent.scroll(disclaimer)
    expect(checkbox).toBeEnabled()
    expect(within(dialog).getByRole('status')).toHaveTextContent(
      'Acknowledgment is now available.',
    )

    Object.defineProperty(disclaimer, 'scrollTop', {
      configurable: true,
      writable: true,
      value: 0,
    })
    fireEvent.scroll(disclaimer)
    expect(checkbox).toBeEnabled()
    await user.click(checkbox)
    expect(within(dialog).getByRole('button', { name: 'Continue' })).toBeEnabled()
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    expect(recordedRequests()).toHaveLength(0)
    await waitFor(() => expect(trigger).toHaveFocus())

    await user.click(trigger)
    expect(screen.getByRole('checkbox', { name: acknowledgment })).toBeDisabled()
    expect(screen.getByRole('checkbox', { name: acknowledgment })).not.toBeChecked()
  })

  it('unlocks immediately when the disclaimer fits and supports focus and Escape', async () => {
    clientHeight = 300
    scrollHeight = 300
    const { user, trigger } = await openFromReview()

    const disclaimer = screen.getByLabelText('Legal disclaimer')
    expect(disclaimer.clientHeight).toBe(300)
    expect(disclaimer.scrollHeight).toBe(300)
    await waitFor(() =>
      expect(screen.getByRole('checkbox', { name: acknowledgment })).toBeEnabled(),
    )
    expect(screen.getByRole('dialog').contains(document.activeElement)).toBe(true)
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(trigger).toHaveFocus()
    expect(recordedRequests()).toHaveLength(0)
  })

  it('recalculates on resize and keeps the read state monotonic', async () => {
    class ControlledResizeObserver {
      constructor(callback: ResizeObserverCallback) {
        notifyResize = () => callback([], this)
      }

      observe() {}
      unobserve() {}
      disconnect() {}
      takeRecords() {
        return []
      }
    }
    Object.defineProperty(window, 'ResizeObserver', {
      configurable: true,
      value: ControlledResizeObserver,
    })
    await openFromReview()
    const checkbox = screen.getByRole('checkbox', { name: acknowledgment })
    expect(checkbox).toBeDisabled()

    clientHeight = 300
    act(() => notifyResize())
    expect(checkbox).toBeEnabled()
    clientHeight = 100
    act(() => notifyResize())
    expect(checkbox).toBeEnabled()
  })

  it('returns to a fresh unread state when the configuration changes', async () => {
    const { user, id } = await openFromReview()
    await acknowledgeAndContinue(user)
    expect(screen.getByRole('button', { name: 'Download Word document' })).toBeEnabled()

    const steps = sourceSteps()
    act(() => {
      useAgreementsStore.getState().saveStep(id, 'pricing', {
        ...steps.pricing,
        unitFee: 25,
      })
    })

    expect(await screen.findByText(disclaimerOne)).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: acknowledgment })).toBeDisabled()
    expect(screen.getByRole('checkbox', { name: acknowledgment })).not.toBeChecked()
  })

  it('shows exactly two primary delivery options after acknowledgment', async () => {
    const { user } = await openFromReview()
    await acknowledgeAndContinue(user)
    const dialog = screen.getByRole('dialog')

    expect(within(dialog).getByLabelText('Work email')).toHaveValue(defaultMe.email)
    expect(
      within(dialog).getByRole('button', { name: 'Download Word document' }),
    ).toHaveAttribute('data-variant', 'default')
    expect(
      within(dialog).getByRole('button', { name: 'Email Word document' }),
    ).toHaveAttribute('data-variant', 'default')
    expect(dialog.querySelectorAll('button[data-variant="default"]')).toHaveLength(2)
  })

  it('blocks a configuration that fails full client validation', async () => {
    const id = seedSourceDocument()
    const draft = useAgreementsStore.getState().drafts[id]!
    useAgreementsStore.setState({
      drafts: {
        [id]: {
          ...draft,
          steps: {
            ...draft.steps,
            environment: {
              ...draft.steps.environment!,
              difficulty: { easy: 20, medium: 20, hard: 20 },
            },
          },
        },
      },
    })
    const rendered = renderApp(`/agreements/${id}/edit?step=8`)
    await rendered.user.click(
      await screen.findByRole('button', { name: 'Get Word document' }),
    )
    await acknowledgeAndContinue(rendered.user)
    await rendered.user.click(
      screen.getByRole('button', { name: 'Download Word document' }),
    )

    expect(
      screen.getByRole('alert', { name: 'Agreement needs attention' }),
    ).toHaveTextContent('Complete every required agreement field before delivery.')
    expect(recordedRequests()).toHaveLength(0)
  })
})

describe('template delivery requests', () => {
  it('downloads the validated configuration once with a sanitized response filename', async () => {
    let body: unknown
    let query = ''
    let authorization = ''
    server.use(
      http.post(
        `${env.VITE_API_URL}/v1/template-deliveries/download`,
        async ({ request }) => {
          body = await request.json()
          query = new URL(request.url).search
          authorization = request.headers.get('Authorization') ?? ''
          return new HttpResponse(Uint8Array.from([80, 75, 3, 4]), {
            status: 200,
            headers: {
              'Content-Type':
                'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
              'Content-Disposition':
                "attachment; filename*=UTF-8''..%2F..%2FMy%20Agreement.docx",
            },
          })
        },
      ),
    )
    let clickedDownload = ''
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
      this: HTMLAnchorElement,
    ) {
      clickedDownload = this.download
    })
    const { user } = await openFromReview()
    await acknowledgeAndContinue(user)

    const button = screen.getByRole('button', { name: 'Download Word document' })
    await user.dblClick(button)

    await waitFor(() => expect(objectUrls.created).toHaveLength(1))
    expect(body).toEqual(
      expect.objectContaining({
        acknowledged: true,
        configuration: expect.objectContaining({
          template_version: 'content-development-1',
          organization: expect.objectContaining({ legal_name: 'Sieve Inc.' }),
          steps: expect.objectContaining({
            agreement: expect.objectContaining({ sow_number: 1 }),
          }),
        }),
      }),
    )
    expect(query).toBe('')
    expect(authorization).toBe('Bearer token')
    expect(clickedDownload).toBe('My Agreement.docx')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Get Word document' }))
    expect(screen.getByRole('checkbox', { name: acknowledgment })).toBeDisabled()
    expect(screen.getByRole('checkbox', { name: acknowledgment })).not.toBeChecked()
  })

  it('keeps a failed download honest and available for retry', async () => {
    let calls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/template-deliveries/download`, () => {
        calls += 1
        if (calls === 1) {
          return HttpResponse.json({ error: 'generation_failed' }, { status: 500 })
        }
        return new HttpResponse(Uint8Array.from([80, 75, 3, 4]), {
          status: 200,
          headers: {
            'Content-Type':
              'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
            'Content-Disposition': 'attachment; filename="Agreement.docx"',
          },
        })
      }),
    )
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const { user } = await openFromReview()
    await acknowledgeAndContinue(user)
    await user.click(screen.getByRole('button', { name: 'Download Word document' }))

    expect(
      await screen.findByRole('alert', { name: 'Download failed' }),
    ).toHaveTextContent("We couldn't generate your Word document. Try again.")
    const retry = screen.getByRole('button', { name: 'Download Word document' })
    expect(retry).toBeEnabled()
    await user.click(retry)
    await waitFor(() => expect(objectUrls.created).toHaveLength(1))
    expect(calls).toBe(2)
  })

  it('rejects a successful response whose body is not a Word document', async () => {
    server.use(
      http.post(
        `${env.VITE_API_URL}/v1/template-deliveries/download`,
        () =>
          new HttpResponse('<html>proxy error</html>', {
            status: 200,
            headers: {
              'Content-Type': 'text/html; charset=utf-8',
              'Content-Disposition': 'attachment; filename="Agreement.docx"',
            },
          }),
      ),
    )
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => undefined)
    const { user } = await openFromReview()
    await acknowledgeAndContinue(user)
    await user.click(screen.getByRole('button', { name: 'Download Word document' }))

    expect(
      await screen.findByRole('alert', { name: 'Download failed' }),
    ).toHaveTextContent("We couldn't generate your Word document. Try again.")
    expect(objectUrls.created).toHaveLength(0)
  })

  it('validates editable email, prevents duplicates, and reports confirmed success', async () => {
    let calls = 0
    let body: unknown
    let key = ''
    let query = ''
    server.use(
      http.post(
        `${env.VITE_API_URL}/v1/template-deliveries/email`,
        async ({ request }) => {
          calls += 1
          body = await request.json()
          key = request.headers.get('Idempotency-Key') ?? ''
          query = new URL(request.url).search
          await new Promise((resolve) => setTimeout(resolve, 20))
          return HttpResponse.json({ accepted: true }, { status: 202 })
        },
      ),
    )
    const { user } = await openFromReview()
    await acknowledgeAndContinue(user)
    const input = screen.getByLabelText('Work email')
    await user.clear(input)
    await user.type(input, 'not-an-email')
    await user.click(screen.getByRole('button', { name: 'Email Word document' }))
    expect(await screen.findByText('Enter a valid work email.')).toBeInTheDocument()
    expect(calls).toBe(0)

    await user.clear(input)
    await user.type(input, 'reader@example.com')
    const emailButton = screen.getByRole('button', { name: 'Email Word document' })
    await user.dblClick(emailButton)
    expect(screen.getByRole('button', { name: 'Emailing Word document…' })).toBeDisabled()
    await user.keyboard('{Escape}')
    expect(screen.getByRole('dialog')).toBeInTheDocument()

    expect(
      await screen.findByText('Your Word document was accepted for email delivery.'),
    ).toBeInTheDocument()
    expect(calls).toBe(1)
    expect(key.length).toBeGreaterThanOrEqual(8)
    expect(query).toBe('')
    expect(body).toEqual(
      expect.objectContaining({
        acknowledged: true,
        email: 'reader@example.com',
        configuration: expect.objectContaining({
          template_version: 'content-development-1',
        }),
      }),
    )
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    await user.click(screen.getByRole('button', { name: 'Get Word document' }))
    expect(screen.getByRole('checkbox', { name: acknowledgment })).not.toBeChecked()
  })

  it('shows an honest retryable error and sends nothing after Back', async () => {
    let calls = 0
    server.use(
      http.post(`${env.VITE_API_URL}/v1/template-deliveries/email`, () => {
        calls += 1
        return HttpResponse.json({ error: 'email_delivery_unavailable' }, { status: 503 })
      }),
    )
    const { user } = await openFromReview()
    await acknowledgeAndContinue(user)
    await user.click(screen.getByRole('button', { name: 'Email Word document' }))

    expect(
      await screen.findByRole('alert', { name: 'Email delivery failed' }),
    ).toHaveTextContent("We couldn't email your Word document. Try again.")
    expect(screen.getByRole('button', { name: 'Email Word document' })).toBeEnabled()
    await user.click(screen.getByRole('button', { name: 'Back' }))
    expect(screen.getByRole('checkbox', { name: acknowledgment })).toBeChecked()
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(calls).toBe(1)
  })
})

describe('configured document entry points', () => {
  it('gates ready drafts from the agreements list and removes the old document route', async () => {
    const id = seedSourceDocument()
    const list = renderApp('/agreements')
    const trigger = await screen.findByRole('button', { name: 'Get Word document' })
    await list.user.click(trigger)
    expect(
      screen.getByRole('dialog', { name: 'Before You Download' }),
    ).toBeInTheDocument()
    await list.user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(recordedRequests()).toHaveLength(0)

    list.router.navigate(`/agreements/${id}/document`)
    expect(await screen.findByText(/page not found/i)).toBeInTheDocument()
  })
})
