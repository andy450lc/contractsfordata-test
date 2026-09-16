import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider, type RouteObject } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { RootErrorBoundary } from '@/app/root-error-boundary'

function Boom(): never {
  throw new Error('boom')
}

describe('root error boundary', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows a recovery UI instead of a blank screen when a page crashes', () => {
    // React logs the caught render error. Silencing it keeps test output clean.
    vi.spyOn(console, 'error').mockImplementation(() => {})

    const crashingRoutes: RouteObject[] = [
      {
        errorElement: <RootErrorBoundary />,
        children: [{ path: '/', element: <Boom /> }],
      },
    ]
    const router = createMemoryRouter(crashingRoutes, { initialEntries: ['/'] })
    render(<RouterProvider router={router} />)

    expect(screen.getByText(/something went wrong/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /reload the app/i })).toBeInTheDocument()
  })
})
