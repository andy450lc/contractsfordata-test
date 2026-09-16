import { describe, expect, it } from 'vitest'

import { renderApp, screen } from '@/test/render'

describe('app routes', () => {
  it.each([
    ['/', 'SoW'],
    ['/sign-up', 'SoW'],
  ])('renders the page at %s', async (path, heading) => {
    renderApp(path)

    expect(await screen.findByRole('heading', { name: heading })).toBeInTheDocument()
  })
})
