import { zodResolver } from '@hookform/resolvers/zod'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useForm } from 'react-hook-form'
import { describe, expect, it, vi } from 'vitest'
import { z } from 'zod'

import { MoneyField } from './money-field'

const schema = z.object({
  currency: z.string(),
  amount: z.number({ error: 'Required' }).min(1, 'Must be 1 or more'),
})
type Values = z.infer<typeof schema>

function Harness({
  onSubmit,
  defaultValues,
}: {
  onSubmit: (values: Values) => void
  defaultValues: Partial<Values>
}) {
  const form = useForm<Values>({ resolver: zodResolver(schema), defaultValues })
  return (
    <form onSubmit={form.handleSubmit(onSubmit)} noValidate>
      <MoneyField
        control={form.control}
        currencyName="currency"
        amountName="amount"
        label="Fee"
        required
        placeholder="20.00"
      />
      <button type="button" onClick={() => form.setValue('amount', 250)}>
        Set 250
      </button>
      <button
        type="button"
        onClick={() => form.reset({ currency: 'EUR', amount: Number.NaN })}
      >
        Reset
      </button>
      <button type="submit">Save</button>
    </form>
  )
}

describe('MoneyField', () => {
  it('shows the error state and clears it when a value is typed', async () => {
    const user = userEvent.setup()
    const onSubmit = vi.fn()
    render(
      <Harness
        onSubmit={onSubmit}
        defaultValues={{ currency: 'USD', amount: Number.NaN }}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Required')).toBeInTheDocument()
    const amount = screen.getByLabelText(/Fee/)
    expect(amount).toHaveAttribute('aria-invalid', 'true')

    await user.type(amount, '1234.5')
    expect(amount).toHaveValue('1,234.5')
    await user.click(screen.getByRole('button', { name: 'Save' }))
    expect(onSubmit).toHaveBeenCalledWith(
      { currency: 'USD', amount: 1234.5 },
      expect.anything(),
    )
  })

  it('follows values set from outside and falls back to USD', async () => {
    const user = userEvent.setup()
    render(<Harness onSubmit={vi.fn()} defaultValues={{ amount: 12 }} />)

    expect(screen.getByRole('button', { name: 'Currency, USD' })).toHaveTextContent('$')
    expect(screen.getByLabelText(/Fee/)).toHaveValue('12')
    await user.click(screen.getByRole('button', { name: 'Set 250' }))
    expect(screen.getByLabelText(/Fee/)).toHaveValue('250')
    await user.click(screen.getByRole('button', { name: 'Reset' }))
    expect(screen.getByLabelText(/Fee/)).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Currency, EUR' })).toHaveTextContent('€')
  })

  it('filters the currency list and reports no match', async () => {
    const user = userEvent.setup()
    render(<Harness onSubmit={vi.fn()} defaultValues={{ currency: 'USD', amount: 1 }} />)

    await user.click(screen.getByRole('button', { name: 'Currency, USD' }))
    await user.type(await screen.findByPlaceholderText('Search currencies'), 'qxqxqx')
    expect(await screen.findByText('No currency found.')).toBeInTheDocument()
    await user.clear(screen.getByPlaceholderText('Search currencies'))
    await user.type(screen.getByPlaceholderText('Search currencies'), 'pound')
    await user.click(await screen.findByRole('option', { name: /GBP/ }))
    expect(screen.getByRole('button', { name: 'Currency, GBP' })).toHaveTextContent('£')
  })
})
