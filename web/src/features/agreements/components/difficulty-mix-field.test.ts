import { describe, expect, it } from 'vitest'

import { rebalance } from './difficulty-mix-field'

describe('rebalance', () => {
  const mix = { easy: 20, medium: 30, hard: 50 }

  it('takes an easy change out of medium first', () => {
    expect(rebalance(mix, 'easy', 40)).toEqual({ easy: 40, medium: 10, hard: 50 })
  })

  it('takes from hard once medium is exhausted', () => {
    expect(rebalance(mix, 'easy', 60)).toEqual({ easy: 60, medium: 0, hard: 40 })
  })

  it('takes a hard change out of medium first', () => {
    expect(rebalance(mix, 'hard', 40)).toEqual({ easy: 20, medium: 40, hard: 40 })
  })

  it('takes a medium change out of hard first', () => {
    expect(rebalance(mix, 'medium', 70)).toEqual({ easy: 20, medium: 70, hard: 10 })
  })

  it('clamps to 0 to 100 and whole numbers', () => {
    expect(rebalance(mix, 'easy', 140)).toEqual({ easy: 100, medium: 0, hard: 0 })
    expect(rebalance(mix, 'easy', Number.NaN)).toEqual({ easy: 0, medium: 50, hard: 50 })
  })
})
