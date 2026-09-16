import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ResizeObserverMock } from '../test/setup'
import {
  TemplateDeliveryDialog,
  sanitizeDownloadFilename,
} from './TemplateDeliveryDialog'

const ENDPOINT = 'https://api.example.test/v1/public/sow-configurator'
const DOCX_MIME =
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
const FALLBACK_FILENAME = 'Robot_Sensor_Data_Content_Development_Agreement.docx'
const ACKNOWLEDGMENT =
  'I acknowledge that this is not legal advice, and that I am responsible for my own legal review before use.'

function addTrigger(name = 'Download template') {
  const trigger = document.createElement('button')
  trigger.type = 'button'
  trigger.dataset.openTemplate = ''
  trigger.textContent = name
  document.body.append(trigger)
  return trigger
}

function setDisclaimerSize(clientHeight: number, scrollHeight: number) {
  Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
    configurable: true,
    get: () => clientHeight,
  })
  Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
    configurable: true,
    get: () => scrollHeight,
  })
}

async function openConfigurator() {
  const trigger = addTrigger()
  await userEvent.click(trigger)
  await userEvent.type(screen.getByRole('textbox', { name: /Company name/ }), 'Acme')
  await userEvent.selectOptions(
    screen.getByRole('combobox', { name: /Your type/ }),
    'buyer',
  )
  const floorAmount = screen.getByRole('spinbutton', { name: 'Dollar floor ($)' })
  await userEvent.clear(floorAmount)
  await userEvent.type(floorAmount, '50000')
  const multiplier = screen.getByRole('spinbutton', { name: 'Fee multiplier (×)' })
  await userEvent.clear(multiplier)
  await userEvent.type(multiplier, '3')
  return trigger
}

async function reachDisclaimer() {
  await userEvent.click(screen.getByRole('button', { name: 'Review disclaimer' }))
}

async function reachDelivery() {
  await reachDisclaimer()
  const disclaimer = screen.getByTestId('disclaimer-scroll')
  Object.defineProperty(disclaimer, 'scrollTop', {
    configurable: true,
    value: 198,
    writable: true,
  })
  fireEvent.scroll(disclaimer)
  await userEvent.click(screen.getByRole('checkbox', { name: ACKNOWLEDGMENT }))
  await userEvent.click(screen.getByRole('button', { name: 'Continue' }))
}

describe('public SOW configurator', () => {
  beforeEach(() => {
    setDisclaimerSize(100, 300)
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  })

  it('opens every template trigger at the four-question configurator with required defaults', async () => {
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    const triggers = ['Header', 'Hero', 'Inline', 'Footer', '404'].map(addTrigger)

    for (const trigger of triggers) {
      await userEvent.click(trigger)
      expect(screen.getByRole('dialog', { name: 'Customize Your SOW' })).toBeVisible()
      expect(screen.getAllByRole('group')).toHaveLength(4)
      expect(
        screen.getByRole('radio', {
          name: "B. Only with Buyer's written consent (default)",
        }),
      ).toBeChecked()
      expect(
        screen.getByRole('radio', {
          name: "B. Set in each SOW's country appendix (default)",
        }),
      ).toBeChecked()
      expect(
        screen.getByRole('radio', {
          name: 'A. Greater of a dollar floor or a fee multiple (default)',
        }),
      ).toBeChecked()
      await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    }
  })

  it('requires field text for limited exclusivity and explains each question', async () => {
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    await openConfigurator()
    await userEvent.click(
      screen.getByRole('radio', { name: 'B. Exclusive, limited to a field' }),
    )
    expect(screen.getByText(/Controls whether the developer may involve/)).toHaveClass(
      'italic',
    )
    expect(screen.getByRole('textbox', { name: 'Exclusivity field' })).toBeRequired()

    await userEvent.click(screen.getByRole('button', { name: 'Review disclaimer' }))
    expect(screen.getByRole('dialog', { name: 'Customize Your SOW' })).toBeVisible()
    await userEvent.type(
      screen.getByRole('textbox', { name: 'Exclusivity field' }),
      'autonomous warehouse robotics',
    )
    await reachDisclaimer()
    expect(screen.getByRole('dialog', { name: 'Before You Download' })).toBeVisible()
  })

  it('uses the real scroll container, a two-pixel tolerance, and active acknowledgment', async () => {
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    await openConfigurator()
    await reachDisclaimer()
    const disclaimer = screen.getByTestId('disclaimer-scroll')
    const checkbox = screen.getByRole('checkbox', { name: ACKNOWLEDGMENT })
    const continueButton = screen.getByRole('button', { name: 'Continue' })

    Object.defineProperty(disclaimer, 'scrollTop', {
      configurable: true,
      value: 197,
      writable: true,
    })
    fireEvent.scroll(disclaimer)
    expect(checkbox).toBeDisabled()
    Object.defineProperty(disclaimer, 'scrollTop', {
      configurable: true,
      value: 198,
      writable: true,
    })
    fireEvent.scroll(disclaimer)
    expect(checkbox).toBeEnabled()
    expect(continueButton).toBeDisabled()
    await userEvent.click(checkbox)
    expect(continueButton).toBeEnabled()

    Object.defineProperty(disclaimer, 'scrollTop', { configurable: true, value: 0 })
    fireEvent.scroll(disclaimer)
    expect(checkbox).toBeEnabled()
  })

  it('unlocks non-scrollable text and keeps the read state after resize', async () => {
    setDisclaimerSize(100, 101)
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    await openConfigurator()
    await reachDisclaimer()
    expect(screen.getByRole('checkbox', { name: ACKNOWLEDGMENT })).toBeEnabled()

    setDisclaimerSize(100, 500)
    act(() =>
      ResizeObserverMock.instances
        .at(-1)
        ?.callback([], ResizeObserverMock.instances.at(-1)!),
    )
    expect(screen.getByRole('checkbox', { name: ACKNOWLEDGMENT })).toBeEnabled()
  })

  it('posts all selected choices and required client information in the download body', async () => {
    let resolveFetch!: (response: Response) => void
    const fetchMock = vi.fn(
      () => new Promise<Response>((resolve) => (resolveFetch = resolve)),
    )
    vi.stubGlobal('fetch', fetchMock)
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    await openConfigurator()
    await userEvent.click(screen.getByRole('radio', { name: 'A. Never allowed' }))
    await userEvent.click(
      screen.getByRole('radio', { name: 'C. Separate Jurisdiction Rider per country' }),
    )
    await userEvent.click(screen.getByRole('radio', { name: 'B. A fee multiple only' }))
    await userEvent.click(
      screen.getByRole('radio', { name: 'A. Fully exclusive, global' }),
    )
    await reachDelivery()

    const download = screen.getByRole('button', { name: 'Download Word document' })
    fireEvent.click(download)
    fireEvent.click(download)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const call = fetchMock.mock.calls[0] as unknown as [string, RequestInit]
    const [, init] = call
    expect(call[0]).toBe(`${ENDPOINT}/download`)
    expect(JSON.parse(String(init.body))).toEqual({
      acknowledged: true,
      client_information: {
        delivery_preference: 'download',
        party_type: 'buyer',
        company_name: 'Acme',
      },
      answers: {
        subcontracting: 'never_allowed',
        jurisdiction_rules: 'separate_country_rider',
        data_ip_liability_cap: 'fee_multiple_only',
        data_ip_liability_multiplier: 3,
        exclusivity: 'fully_exclusive_global',
      },
    })
    resolveFetch(
      new Response(new Blob(['docx'], { type: DOCX_MIME }), {
        status: 200,
        headers: {
          'Content-Type': DOCX_MIME,
          'Content-Disposition':
            'attachment; filename="Robot_Sensor_Data_Content_Development_Agreement.docx"',
        },
      }),
    )
    await waitFor(() => expect(HTMLAnchorElement.prototype.click).toHaveBeenCalled())
  })

  it('shows only the liability inputs each cap option needs', async () => {
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    await openConfigurator()
    expect(screen.getByRole('spinbutton', { name: 'Dollar floor ($)' })).toBeRequired()
    expect(screen.getByRole('spinbutton', { name: 'Fee multiplier (×)' })).toBeRequired()

    await userEvent.click(screen.getByRole('radio', { name: 'B. A fee multiple only' }))
    expect(
      screen.queryByRole('spinbutton', { name: 'Dollar floor ($)' }),
    ).not.toBeInTheDocument()
    expect(screen.getByRole('spinbutton', { name: 'Fee multiplier (×)' })).toBeRequired()

    await userEvent.click(screen.getByRole('radio', { name: 'C. No special cap' }))
    expect(
      screen.queryByRole('spinbutton', { name: 'Dollar floor ($)' }),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('spinbutton', { name: 'Fee multiplier (×)' }),
    ).not.toBeInTheDocument()

    // Nothing left to fill in, so the flow can proceed without error.
    await reachDisclaimer()
    expect(screen.getByRole('dialog', { name: 'Before You Download' })).toBeVisible()
  })

  it('hides unverified email delivery and enables provider-confirmed email when configured', async () => {
    const first = render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    await openConfigurator()
    await reachDelivery()
    expect(
      screen.queryByRole('button', { name: 'Email Word document' }),
    ).not.toBeInTheDocument()
    expect(
      screen.getByText(/provider have passed an end-to-end verification/),
    ).toBeVisible()

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    first.unmount()
    document
      .querySelectorAll('[data-open-template]')
      .forEach((element) => element.remove())
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ accepted: true }), {
        status: 202,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    vi.stubGlobal('fetch', fetchMock)
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} emailDeliveryEnabled />)
    await openConfigurator()
    await reachDelivery()
    await userEvent.type(
      screen.getByRole('textbox', { name: 'Work email' }),
      'reader@example.com',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Email Word document' }))
    await waitFor(() =>
      expect(screen.getByRole('status')).toHaveTextContent('Sent. Check your inbox.'),
    )
    expect(fetchMock.mock.calls[0][0]).toBe(`${ENDPOINT}/email`)
    expect(JSON.parse(String(fetchMock.mock.calls[0][1].body))).toMatchObject({
      acknowledged: true,
      email: 'reader@example.com',
      client_information: { delivery_preference: 'email' },
      answers: {
        subcontracting: 'buyer_written_consent',
        jurisdiction_rules: 'sow_country_appendix',
        data_ip_liability_cap: 'greater_of_amount_or_fee_multiple',
        exclusivity: 'non_exclusive',
      },
    })
  })

  it('resets acknowledgment after changing answers and after reopening while preserving choices', async () => {
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    const trigger = await openConfigurator()
    await userEvent.click(screen.getByRole('radio', { name: 'A. Never allowed' }))
    await reachDelivery()
    await userEvent.click(screen.getByRole('button', { name: 'Change answers' }))
    expect(screen.getByRole('radio', { name: 'A. Never allowed' })).toBeChecked()
    await reachDisclaimer()
    expect(screen.getByRole('checkbox', { name: ACKNOWLEDGMENT })).toBeDisabled()

    await userEvent.keyboard('{Escape}')
    expect(trigger).toHaveFocus()
    await userEvent.click(trigger)
    expect(screen.getByRole('radio', { name: 'A. Never allowed' })).toBeChecked()
    await reachDisclaimer()
    expect(screen.getByRole('checkbox', { name: ACKNOWLEDGMENT })).not.toBeChecked()
  })

  it('cancels without requesting a document and reports an honest download failure', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 503 }))
    vi.stubGlobal('fetch', fetchMock)
    render(<TemplateDeliveryDialog endpoint={ENDPOINT} />)
    await openConfigurator()
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(fetchMock).not.toHaveBeenCalled()

    await openConfigurator()
    await reachDelivery()
    await userEvent.click(screen.getByRole('button', { name: 'Download Word document' }))
    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        'We could not prepare the document. Please try again.',
      ),
    )
  })
})

describe('sanitizeDownloadFilename', () => {
  it('keeps a safe DOCX name and rejects paths or other extensions', () => {
    expect(sanitizeDownloadFilename('attachment; filename="Agreement.docx"')).toBe(
      'Agreement.docx',
    )
    expect(sanitizeDownloadFilename('attachment; filename="../template.exe"')).toBe(
      FALLBACK_FILENAME,
    )
  })
})
