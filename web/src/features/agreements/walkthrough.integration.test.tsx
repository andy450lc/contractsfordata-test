import { beforeEach, describe, expect, it } from 'vitest'

import { useSessionStore } from '@/stores/session'
import { fireEvent, renderApp, screen, waitFor, within } from '@/test/render'

import {
  defaultAgreementTerms,
  defaultCounterparty,
  defaultDelivery,
  defaultEnvironment,
  defaultPricing,
  defaultScope,
  defaultTechnical,
} from './defaults'
import { useAgreementsStore } from './store'

const organization = {
  legalName: 'Sieve Inc.',
  entityType: 'corporation',
  incorporationPlace: 'Delaware',
  governingLaw: 'Delaware',
  address: '1 Main St, Wilmington, DE',
  shortName: 'Sieve',
  signerName: 'Ishan Dhawan',
  signerTitle: 'Chief of Staff',
  signerEmail: 'ishan@sieve.example',
  contactPhone: '',
}

function seedDraft(withAllSteps: boolean) {
  const store = useAgreementsStore.getState()
  const id = store.createDraft({
    ...defaultAgreementTerms(1),
    title: 'Pilot',
    effectiveDate: '2026-08-29',
  })
  store.saveStep(id, 'scope', {
    ...defaultScope,
    deliverableDescription: 'stereo footage',
    regions: ['IN'],
    venueConstraint: 'operating businesses',
    targetVolume: 300,
  })
  if (!withAllSteps) return id
  store.saveStep(id, 'pricing', {
    ...defaultPricing(300, 'accepted usable hours'),
    unitFee: 20,
    invoiceEmail: 'ap@sievedata.com',
    milestones: [
      {
        name: 'Rolling daily uploads',
        volume: 'As available',
        deadline: '2026-09-06',
        notes: '',
        isFinal: false,
      },
      {
        name: 'Final delivery',
        volume: '300 accepted usable hours',
        deadline: '2026-09-13',
        notes: '',
        isFinal: true,
      },
    ],
  })
  store.saveStep(id, 'environment', defaultEnvironment('accepted usable hours'))
  store.saveStep(id, 'technical', defaultTechnical)
  store.saveStep(id, 'delivery', defaultDelivery)
  store.saveStep(id, 'counterparty', {
    ...defaultCounterparty,
    email: 'ishan@sieve.example',
    legalName: 'Sieve Inc.',
    entityJurisdiction: 'Delaware corporation',
    address: '1 Main St',
    shortName: 'Sieve',
    signatoryName: 'Ishan Dhawan',
    signatoryTitle: 'Chief of Staff',
    ccEmails: 'counsel@sieve.example',
  })
  return id
}

beforeEach(() => {
  window.localStorage.clear()
  useAgreementsStore.setState({ organization: null, drafts: {} })
  useSessionStore.getState().setAuthenticated('token', Date.now() + 300_000)
})

describe('steps 3 to 6', () => {
  it('saves pricing with a marked final milestone and a deposit amount', async () => {
    const id = seedDraft(false)
    const { user, router } = renderApp(`/agreements/${id}/edit?step=3`)

    expect(await screen.findByText('Step 3 of 8')).toBeInTheDocument()
    expect(
      screen.getByText(/300 accepted usable hours from the Scope step/),
    ).toBeInTheDocument()
    await user.type(screen.getByLabelText(/Fee per accepted usable hour/), '20')
    expect(screen.getByText('$6,000.00')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Currency, USD' }))
    await user.type(await screen.findByPlaceholderText('Search currencies'), 'rupee')
    await user.click(await screen.findByRole('option', { name: /INR/ }))
    expect(
      await screen.findByRole('button', { name: 'Currency, INR' }),
    ).toHaveTextContent('₹')
    expect(screen.getByText('₹6,000.00')).toBeInTheDocument()
    await user.type(screen.getByLabelText(/Invoice email/), 'ap@sievedata.com')
    const deposit = screen.getByRole('radiogroup', { name: 'Deposit or advance' })
    await user.click(within(deposit).getByRole('radio', { name: 'Yes' }))
    await user.click(screen.getByRole('button', { name: 'Next' }))
    expect((await screen.findAllByText('Required')).length).toBeGreaterThan(0)
    expect(screen.getByLabelText(/Deposit amount/)).toHaveAttribute(
      'aria-invalid',
      'true',
    )
    await user.type(screen.getByLabelText(/Deposit amount/), '1000')

    await user.type(screen.getByLabelText('Deadline, row 1'), '2026-09-06')
    await user.type(screen.getByLabelText('Deadline, row 2'), '2026-09-13')
    await user.click(screen.getByLabelText('Final, row 1'))
    await user.click(screen.getByLabelText('Final, row 2'))
    await user.click(screen.getByRole('button', { name: 'Add milestone' }))
    expect(screen.getByLabelText('Milestone, row 3')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Remove row 3' }))
    screen.getByRole('button', { name: 'Drag to reorder row 2' }).focus()
    await user.keyboard(' ')
    await user.keyboard('{ArrowUp}')
    await user.keyboard(' ')
    await waitFor(() =>
      expect(screen.getByLabelText('Milestone, row 1')).toHaveValue('Final delivery'),
    )
    expect(screen.getByLabelText('Final, row 1')).toBeChecked()
    screen.getByRole('button', { name: 'Drag to reorder row 1' }).focus()
    await user.keyboard(' ')
    await user.keyboard('{ArrowDown}')
    await user.keyboard(' ')
    await waitFor(() =>
      expect(screen.getByLabelText('Milestone, row 1')).toHaveValue(
        'Rolling daily uploads',
      ),
    )
    expect(screen.queryByRole('button', { name: /Move row/ })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(router.state.location.search).toBe('?step=4'))
    const pricing = useAgreementsStore.getState().drafts[id]?.steps.pricing
    expect(pricing?.depositAmount).toBe(1000)
    expect(pricing?.currency).toBe('INR')
    expect(pricing?.milestones[1]?.isFinal).toBe(true)
  })

  it('keeps the difficulty mix at 100% from the fields and the slider, then saves', async () => {
    const id = seedDraft(false)
    const { user, router } = renderApp(`/agreements/${id}/edit?step=4`)

    expect(await screen.findByText('Step 4 of 8')).toBeInTheDocument()
    const hard = screen.getByLabelText(/^Hard/, { selector: '#field-difficulty\\.hard' })
    const medium = screen.getByLabelText(/^Medium/, {
      selector: '#field-difficulty\\.medium',
    })
    await user.clear(hard)
    await user.type(hard, '40')
    expect(medium).toHaveValue('40')
    expect(screen.getByRole('slider', { name: 'Easy share' })).toHaveAttribute(
      'aria-valuenow',
      '20',
    )
    expect(
      screen.getByRole('slider', { name: 'Easy plus medium share' }),
    ).toHaveAttribute('aria-valuenow', '60')

    screen.getByRole('slider', { name: 'Easy share' }).focus()
    await user.keyboard('{ArrowRight}')
    expect(
      screen.getByLabelText(/^Easy/, { selector: '#field-difficulty\\.easy' }),
    ).toHaveValue('21')
    expect(medium).toHaveValue('39')

    await user.click(screen.getByRole('button', { name: 'Add item' }))
    await user.type(
      screen.getByLabelText('Must not be captured or delivered, row 1'),
      'Company logos',
    )
    await user.click(screen.getByRole('button', { name: 'Next' }))
    await waitFor(() => expect(router.state.location.search).toBe('?step=5'))
    const environment = useAgreementsStore.getState().drafts[id]?.steps.environment
    expect(environment?.difficulty).toEqual({ easy: 21, medium: 39, hard: 40 })
    expect(environment?.extraProhibited).toEqual([{ text: 'Company logos' }])
  })

  it('saves the technical and delivery steps with their defaults', async () => {
    const id = seedDraft(false)
    const { user, router } = renderApp(`/agreements/${id}/edit?step=5`)

    expect(await screen.findByText('Step 5 of 8')).toBeInTheDocument()
    expect(screen.getByLabelText('Requirement, row 7')).toHaveValue('Calibration')
    await user.click(screen.getByRole('button', { name: 'Next' }))

    expect(await screen.findByText('Step 6 of 8')).toBeInTheDocument()
    expect(screen.getByLabelText('Field, row 21')).toHaveValue('Consent-record ID')
    expect(screen.getByLabelText(/Deletion window/)).toHaveValue('30')
    const stays = screen.getAllByRole('radio', { name: 'No' })
    await user.click(stays[stays.length - 1]!)
    expect(screen.queryByLabelText(/Deletion window/)).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Next' }))

    await waitFor(() => expect(router.state.location.search).toBe('?step=7'))
    const steps = useAgreementsStore.getState().drafts[id]?.steps
    expect(steps?.technical?.requirements).toHaveLength(7)
    expect(steps?.delivery?.rejectedStaysDeveloperOwned).toBe(false)
  })
})

describe('table columns', () => {
  it('resizes a column by dragging its divider and keeps a minimum width', async () => {
    const id = seedDraft(false)
    renderApp(`/agreements/${id}/edit?step=5`)

    const divider = await screen.findByRole('separator', {
      name: 'Resize Requirement column',
    })
    const table = divider.closest('table')
    const col = table?.querySelectorAll('col')[1]
    expect(col?.style.width).toBe('180px')

    fireEvent.pointerDown(divider, { clientX: 200, pointerId: 1 })
    fireEvent.pointerMove(divider, { clientX: 320, pointerId: 1 })
    expect(col?.style.width).toBe('120px')
    fireEvent.pointerMove(divider, { clientX: -500, pointerId: 1 })
    expect(col?.style.width).toBe('72px')
    fireEvent.pointerUp(divider, { clientX: -500, pointerId: 1 })
    fireEvent.pointerMove(divider, { clientX: 900, pointerId: 1 })
    expect(col?.style.width).toBe('72px')
    expect(table?.style.width).not.toBe('')
    expect(
      screen.queryByRole('separator', { name: 'Resize Rejection criteria column' }),
    ).not.toBeInTheDocument()
  })
})

describe('review and list', () => {
  it('shows every saved section with the organization and role labels', async () => {
    useAgreementsStore.getState().setOrganization(organization)
    const id = seedDraft(true)
    renderApp(`/agreements/${id}/edit?step=8`)

    expect(
      await screen.findByRole('heading', { name: 'Review & send' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Customer (us)' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Developer' })).toBeInTheDocument()
    expect(screen.getByText('$6,000.00')).toBeInTheDocument()
    expect(
      screen.getByText(
        /Final delivery: 300 accepted usable hours by 2026-09-13 \(final\)/,
      ),
    ).toBeInTheDocument()
    expect(screen.getByText('20% easy, 30% medium, 50% hard')).toBeInTheDocument()
    expect(
      screen.getByText(/Minimum distinct physical sites: 15 sites/),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Ishan Dhawan, Chief of Staff · ishan@sieve.example/),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Organization details come from your saved organization info.'),
    ).toBeInTheDocument()
    expect(screen.getAllByRole('link', { name: 'Edit' })).toHaveLength(7)
    expect(screen.getByRole('link', { name: 'Edit organization info' })).toHaveAttribute(
      'href',
      '/organization',
    )
  })

  it('lists a complete draft, opens it at the review step, and sorts and filters', async () => {
    const id = seedDraft(true)
    const inviteId = useAgreementsStore.getState().createDraft({
      ...defaultAgreementTerms(2),
      title: 'Invite only',
      effectiveDate: '2026-08-29',
    })
    useAgreementsStore.getState().saveStep(inviteId, 'counterparty', {
      ...defaultCounterparty,
      mode: 'invite',
      email: 'someone@example.com',
    })
    const { user, router } = renderApp('/agreements')

    expect(
      await screen.findByRole('link', { name: 'someone@example.com' }),
    ).toHaveAttribute('href', `/agreements/${inviteId}/edit?step=2`)
    const link = await screen.findByRole('link', { name: 'Sieve Inc.' })
    expect(link).toHaveAttribute('href', `/agreements/${id}/edit?step=8`)
    expect(screen.getByText('ishan@sieve.example')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Signed' }))
    expect(router.state.location.search).toBe('?status=signed')
    expect(screen.getByText('Nothing here yet.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Drafts' }))
    expect(screen.getByRole('link', { name: 'Sieve Inc.' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Developer' }))
    expect(router.state.location.search).toBe('?status=drafts&dir=desc&sort=counterparty')
    await user.click(screen.getByRole('button', { name: 'Developer' }))
    expect(router.state.location.search).toBe('?status=drafts&dir=asc&sort=counterparty')
    await user.click(screen.getByRole('button', { name: 'Title' }))
    await user.click(screen.getByRole('button', { name: 'Status' }))
    await user.click(screen.getByRole('button', { name: 'Date created' }))
    await user.click(screen.getByRole('button', { name: 'Date created' }))
    expect(screen.getByRole('link', { name: 'Sieve Inc.' })).toBeInTheDocument()

    const dialogTrigger = screen.getAllByRole('button', { name: 'Delete' })[0]!
    await user.click(dialogTrigger)
    const dialog = await screen.findByRole('dialog')
    await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(useAgreementsStore.getState().drafts[id]).toBeDefined()
  })

  it('fills the short name and governing law from earlier answers', async () => {
    const { user } = renderApp('/organization')
    await user.type(
      await screen.findByLabelText(/Organization legal name/),
      'Pixels Two Corporation',
    )
    await user.type(screen.getByLabelText(/Place of incorporation/), 'Delaware')
    await user.tab()
    expect(screen.getByLabelText(/Short name used/)).toHaveValue('Pixels')
    expect(screen.getByLabelText(/Governing law jurisdiction/)).toHaveValue('Delaware')
    await user.click(screen.getByRole('link', { name: 'Cancel' }))
    expect(
      await screen.findByRole('heading', { name: 'My agreements' }),
    ).toBeInTheDocument()
  })
})
