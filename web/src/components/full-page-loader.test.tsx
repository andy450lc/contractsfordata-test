import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { FullPageLoader } from './full-page-loader'

describe('FullPageLoader', () => {
  it('announces the wait to screen readers', () => {
    render(<FullPageLoader />)

    expect(screen.getByRole('status')).toHaveTextContent('Loading')
  })
})
