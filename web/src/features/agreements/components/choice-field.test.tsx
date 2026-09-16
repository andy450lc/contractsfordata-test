import { zodResolver } from '@hookform/resolvers/zod'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useForm } from 'react-hook-form'
import { describe, expect, it, vi } from 'vitest'
import { z } from 'zod'

import { ChoiceField } from './choice-field'

const schema = z.object({ cadence: z.string().min(1, 'Required') })
type Values = z.infer<typeof schema>

function Harness({
  onSubmit,
  cadence,
  allowCustom,
}: {
  onSubmit: (values: Values) => void
  cadence: string
  allowCustom?: boolean
}) {
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { cadence },
  })
  return (
    <form onSubmit={form.handleSubmit(onSubmit)} noValidate>
      <ChoiceField
        control={form.control}
        name="cadence"
        label="Delivery cadence"
        options={['Daily', 'Weekly']}
        required
        allowCustom={allowCustom}
      />
      <button type="submit">Save</button>
    </form>
  )
}

describe('ChoiceField', () => {
  it('shows the error state and picks a preset', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(<Harness onSubmit={onSubmit} cadence="" />)

    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Required')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Delivery cadence, none' }))
    await user.click(await screen.findByRole('option', { name: 'Weekly' }))
    expect(
      screen.getByRole('button', { name: 'Delivery cadence, Weekly' }),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(onSubmit).toHaveBeenCalledWith({ cadence: 'Weekly' }, expect.anything())
  })

  it('keeps a custom value in the list when allowed', async () => {
    const user = userEvent.setup()
    render(<Harness onSubmit={vi.fn()} cadence="Every Friday" allowCustom />)

    await user.click(
      screen.getByRole('button', { name: 'Delivery cadence, Every Friday' }),
    )
    expect(
      await screen.findByRole('option', { name: 'Every Friday' }),
    ).toBeInTheDocument()
    await user.type(screen.getByPlaceholderText('Search or type your own'), 'Hourly')
    await user.click(await screen.findByRole('option', { name: 'Use "Hourly"' }))
    expect(
      screen.getByRole('button', { name: 'Delivery cadence, Hourly' }),
    ).toBeInTheDocument()
  })

  it('offers no custom entry when not allowed and matches presets by case', async () => {
    const user = userEvent.setup()
    render(<Harness onSubmit={vi.fn()} cadence="daily" />)
    expect(
      screen.getByRole('button', { name: 'Delivery cadence, Daily' }),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Delivery cadence, Daily' }))
    await user.type(await screen.findByPlaceholderText('Search'), 'Hourly')
    expect(await screen.findByText('No match.')).toBeInTheDocument()
  })
})
