import { zodResolver } from '@hookform/resolvers/zod'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useForm } from 'react-hook-form'
import { describe, expect, it, vi } from 'vitest'
import { z } from 'zod'

import { VolumeField } from './volume-field'

const schema = z.object({
  amount: z.number({ error: 'Required' }),
  unit: z.string().min(1, 'Required'),
})
type Values = z.infer<typeof schema>

function Harness({
  onSubmit,
  defaultValues,
}: {
  onSubmit: (v: Values) => void
  defaultValues: Values
}) {
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues })
  return (
    <form onSubmit={form.handleSubmit(onSubmit)} noValidate>
      <VolumeField
        control={form.control}
        amountName="amount"
        unitName="unit"
        label="Target volume"
        required
      />
      <button type="submit">Save</button>
    </form>
  )
}

describe('VolumeField', () => {
  it('accepts a typed custom unit and keeps it in the list', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(
      <Harness onSubmit={onSubmit} defaultValues={{ amount: Number.NaN, unit: '' }} />,
    )

    expect(screen.getByRole('button', { name: 'Unit, none' })).toHaveTextContent('unit')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Required')).toBeInTheDocument()

    await user.type(screen.getByLabelText(/Target volume/), '1200')
    await user.click(screen.getByRole('button', { name: 'Unit, none' }))
    await user.type(
      await screen.findByPlaceholderText('Search or type a unit'),
      'widgets',
    )
    await user.click(await screen.findByRole('option', { name: 'Use "widgets"' }))
    expect(screen.getByRole('button', { name: 'Unit, widgets' })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Unit, widgets' }))
    expect(await screen.findByRole('option', { name: 'widgets' })).toBeInTheDocument()
    await user.keyboard('{Escape}')

    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(onSubmit).toHaveBeenCalledWith(
      { amount: 1200, unit: 'widgets' },
      expect.anything(),
    )
  })

  it('does not offer a custom entry that matches a preset', async () => {
    const user = userEvent.setup()
    render(<Harness onSubmit={vi.fn()} defaultValues={{ amount: 5, unit: 'hours' }} />)
    await user.click(screen.getByRole('button', { name: 'Unit, hours' }))
    await user.type(await screen.findByPlaceholderText('Search or type a unit'), 'Hours')
    expect(screen.queryByRole('option', { name: /Use "/ })).not.toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'hours' })).toBeInTheDocument()

    await user.clear(screen.getByPlaceholderText('Search or type a unit'))
    await user.type(screen.getByPlaceholderText('Search or type a unit'), 'clip')
    const options = await screen.findAllByRole('option')
    expect(options[0]).toHaveAccessibleName('clips')
    expect(options[1]).toHaveAccessibleName('Use "clip"')
  })
})
