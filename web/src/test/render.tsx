import { render } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'

import { routes } from '@/app/router'

// Mounts the real app routes — no module mocks — at the given path.
export function renderApp(initialPath: string) {
  const router = createMemoryRouter(routes, { initialEntries: [initialPath] })
  const user = userEvent.setup()
  return { user, router, ...render(<RouterProvider router={router} />) }
}

export * from '@testing-library/react'
export { userEvent }
