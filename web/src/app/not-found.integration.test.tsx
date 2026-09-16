import { describe, expect, it } from 'vitest'

import { renderApp, screen } from '@/test/render'

describe('unknown addresses', () => {
  it('shows a page-not-found view with a way back to sign-in', async () => {
    const { user } = renderApp('/nope')

    expect(await screen.findByText(/page not found/i)).toBeInTheDocument()

    await user.click(screen.getByRole('link', { name: /back to sign in/i }))
    expect(await screen.findByRole('heading', { name: 'SoW' })).toBeInTheDocument()
  })
})
