import { beforeEach, describe, expect, it } from 'vitest'

import { useSessionStore } from '@/stores/session'
import { renderApp, screen, waitFor, within } from '@/test/render'

import { useAgreementsStore } from './store'

function signIn() {
  useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)
}

beforeEach(() => {
  window.localStorage.clear()
  useAgreementsStore.setState({ organization: null, drafts: {} })
  signIn()
})

describe('organization info', () => {
  it('requires every field and keeps the saved values', async () => {
    const { user, router } = renderApp('/organization')

    expect(
      await screen.findByRole('heading', { name: 'Organization info' }),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    expect((await screen.findAllByText('Required')).length).toBeGreaterThan(5)

    await user.type(screen.getByLabelText(/Organization legal name/), 'Sieve Inc.')
    await user.type(screen.getByLabelText(/Entity type/), 'corporation')
    await user.type(screen.getByLabelText(/Place of incorporation/), 'Delaware')
    await user.type(screen.getByLabelText(/Governing law jurisdiction/), 'Delaware')
    await user.type(screen.getByLabelText(/Organization address/), '1 Main St')
    await user.type(screen.getByLabelText(/Short name used/), 'Sieve')
    await user.type(screen.getByLabelText(/Signer name/), 'Ishan Dhawan')
    await user.type(screen.getByLabelText(/Signer title/), 'Chief of Staff')
    await user.type(screen.getByLabelText(/Signer email/), 'ishan@sieve.example')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/agreements'))
    expect(useAgreementsStore.getState().organization?.signerName).toBe('Ishan Dhawan')
  })
})

describe('agreement wizard', () => {
  it('blocks step 1 until required fields are filled, then creates a draft', async () => {
    const { user, router } = renderApp('/agreements/new')

    expect(
      await screen.findByRole('heading', { name: 'Create my agreement' }),
    ).toBeInTheDocument()
    expect(screen.getByText('Step 1 of 8')).toBeInTheDocument()
    expect(screen.getByText('1/8')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Back to my agreements' })).toHaveAttribute(
      'href',
      '/agreements',
    )

    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findAllByText('Required')).toHaveLength(2)
    expect(Object.keys(useAgreementsStore.getState().drafts)).toHaveLength(0)

    await user.type(
      screen.getByLabelText(/SOW title/),
      'Pixels2 India Stereo Egocentric Pilot',
    )
    await user.type(screen.getByLabelText(/Effective date/), '2026-08-29')
    await user.click(screen.getByRole('button', { name: 'Next' }))

    await waitFor(() => expect(router.state.location.search).toBe('?step=2'))
    const [draft] = Object.values(useAgreementsStore.getState().drafts)
    expect(draft?.steps.agreement?.title).toBe('Pixels2 India Stereo Egocentric Pilot')
    expect(draft?.steps.agreement?.cureDays).toBe(30)
    expect(router.state.location.pathname).toBe(`/agreements/${draft?.id}/edit`)
    expect(await screen.findByText('Step 2 of 8')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Describe the work' })).toBeInTheDocument()
  })

  it('formats money with separators, computes the maximum total, and checks deadlines', async () => {
    const { user } = renderApp('/agreements/new')
    await user.type(await screen.findByLabelText(/SOW title/), 'Pilot')
    await user.type(screen.getByLabelText(/Effective date/), '2026-08-29')
    await user.click(screen.getByRole('button', { name: 'Next' }))

    await user.type(
      await screen.findByLabelText(/Deliverable description/),
      'stereo footage',
    )
    await user.click(screen.getByRole('button', { name: 'Add country' }))
    await user.type(await screen.findByPlaceholderText('Search countries'), 'ind')
    const matches = await screen.findAllByRole('option')
    expect(matches[0]).toHaveAccessibleName('India')
    await user.click(matches[0]!)
    await user.clear(screen.getByPlaceholderText('Search countries'))
    await user.type(screen.getByPlaceholderText('Search countries'), 'nepal')
    await user.click(await screen.findByRole('option', { name: 'Nepal' }))
    await user.keyboard('{Escape}')
    expect(screen.getByText('India')).toBeInTheDocument()
    expect(screen.getByText('Nepal')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Remove Nepal' }))
    expect(screen.queryByText('Nepal')).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Add another country' }),
    ).toBeInTheDocument()
    await user.type(screen.getByLabelText(/Allowed venues/), 'operating businesses')
    await user.type(screen.getByLabelText(/Target volume/), '300')
    await user.click(screen.getByRole('button', { name: 'Unit, accepted usable hours' }))
    await user.type(await screen.findByPlaceholderText('Search or type a unit'), 'clips')
    await user.click(await screen.findByRole('option', { name: 'clips' }))
    await user.click(screen.getByRole('button', { name: 'Unit, clips' }))
    await user.type(
      await screen.findByPlaceholderText('Search or type a unit'),
      'accepted usable hours',
    )
    await user.click(await screen.findByRole('option', { name: 'accepted usable hours' }))
    await user.click(screen.getByRole('button', { name: 'Next' }))

    expect(await screen.findByText('Step 3 of 8')).toBeInTheDocument()
    const fee = screen.getByLabelText(/Fee per accepted usable hour/)
    await user.type(fee, '20000')
    expect(fee).toHaveValue('20,000')
    expect(screen.getByText('$6,000,000.00')).toBeInTheDocument()

    await user.type(screen.getByLabelText(/Invoice email/), 'ap@sievedata.com')
    await user.type(screen.getByLabelText('Deadline, row 1'), '2026-08-01')
    await user.type(screen.getByLabelText('Deadline, row 2'), '2026-09-13')
    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(
      await screen.findByText('Must be on or after the effective date'),
    ).toBeInTheDocument()
    expect(screen.getByText('Step 3 of 8')).toBeInTheDocument()
  })

  it('keeps unsaved values when going back within the visit', async () => {
    const { user } = renderApp('/agreements/new')
    await user.type(await screen.findByLabelText(/SOW title/), 'Pilot')
    await user.type(screen.getByLabelText(/Effective date/), '2026-08-29')
    await user.click(screen.getByRole('button', { name: 'Next' }))

    await user.type(
      await screen.findByLabelText(/Allowed venues/),
      'operating businesses',
    )
    await user.click(screen.getByRole('button', { name: 'Back' }))
    expect(await screen.findByText('Step 1 of 8')).toBeInTheDocument()
    expect(screen.getByLabelText(/SOW title/)).toHaveValue('Pilot')
    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByLabelText(/Allowed venues/)).toHaveValue(
      'operating businesses',
    )
  })

  it('switches the counterparty step to invite-only and shows the review', async () => {
    const id = useAgreementsStore.getState().createDraft({
      sowNumber: 1,
      title: 'Pilot',
      effectiveDate: '2026-08-29',
      exclusive: true,
      materialsSummary: 'recordings',
      incidentNoticeHours: 24,
      cureDays: 30,
      convenienceNoticeDays: 30,
      recordsRetentionYears: 2,
      liabilityLookbackMonths: 12,
      excludedClaimsCapMultiplier: 2,
      dataClaimsCapFloor: 50000,
    })
    const { user, router } = renderApp(`/agreements/${id}/edit?step=7`)

    expect(await screen.findByText('Step 7 of 8')).toBeInTheDocument()
    expect(screen.getByLabelText(/Developer legal name/)).toBeInTheDocument()
    await user.click(screen.getByLabelText('Invite by email only'))
    expect(screen.queryByLabelText(/Developer legal name/)).not.toBeInTheDocument()
    expect(
      screen.getByText(
        'The developer will provide their name and details when they sign.',
      ),
    ).toBeInTheDocument()

    await user.type(screen.getByLabelText(/Developer email/), 'not-an-email')
    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Enter a valid email address')).toBeInTheDocument()

    await user.clear(screen.getByLabelText(/Developer email/))
    await user.type(screen.getByLabelText(/Developer email/), 'akshaj@pixelstwo.example')
    await user.click(screen.getByRole('button', { name: 'Next' }))

    await waitFor(() => expect(router.state.location.search).toBe('?step=8'))
    expect(screen.getByRole('heading', { name: 'Review & send' })).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Send agreement & signing links' }),
    ).toBeDisabled()
    expect(screen.getByText('Sending arrives in a later release')).toBeInTheDocument()
    expect(
      screen.getByText('Add your organization info before sending.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Pilot (Exclusive)')).toBeInTheDocument()
    expect(screen.getByText('akshaj@pixelstwo.example')).toBeInTheDocument()
  })

  it('shows a not-found page for an unknown draft', async () => {
    renderApp('/agreements/nope/edit')
    expect(await screen.findByText(/not found/i)).toBeInTheDocument()
  })
})

describe('agreements list', () => {
  it('lists drafts, resumes at the next incomplete step, and deletes after confirmation', async () => {
    const store = useAgreementsStore.getState()
    const id = store.createDraft({
      sowNumber: 1,
      title: 'Pilot',
      effectiveDate: '2026-08-29',
      exclusive: true,
      materialsSummary: 'recordings',
      incidentNoticeHours: 24,
      cureDays: 30,
      convenienceNoticeDays: 30,
      recordsRetentionYears: 2,
      liabilityLookbackMonths: 12,
      excludedClaimsCapMultiplier: 2,
      dataClaimsCapFloor: 50000,
    })
    const { user } = renderApp('/agreements')

    expect(
      await screen.findByRole('heading', { name: 'My agreements' }),
    ).toBeInTheDocument()
    const row = screen.getByRole('link', { name: 'No developer yet' })
    expect(row).toHaveAttribute('href', `/agreements/${id}/edit?step=2`)
    expect(screen.getByText('SOW 1 — Pilot')).toBeInTheDocument()
    expect(screen.getByText('Draft')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText(/"Pilot" and every value/)).toBeInTheDocument()
    await user.click(within(dialog).getByRole('button', { name: 'Delete draft' }))

    await waitFor(() =>
      expect(screen.queryByText('SOW 1 — Pilot')).not.toBeInTheDocument(),
    )
    expect(useAgreementsStore.getState().drafts[id]).toBeUndefined()
    expect(screen.getByText('Nothing here yet.')).toBeInTheDocument()
  })
})
