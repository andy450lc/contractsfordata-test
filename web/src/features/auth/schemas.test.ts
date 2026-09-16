import { describe, expect, it } from 'vitest'

import {
  codeSchema,
  emailOnlySchema,
  emailSchema,
  nameSchema,
  newPasswordSchema,
  passwordSchema,
  signInSchema,
  signUpSchema,
  verifyCodeSchema,
} from './schemas'

describe('auth schemas', () => {
  it('trims the email and rejects a malformed or over-long address', () => {
    expect(emailSchema.parse('  ada@example.com  ')).toBe('ada@example.com')
    expect(emailSchema.safeParse('ada@').success).toBe(false)
    expect(emailSchema.safeParse('').success).toBe(false)

    const long = `${'a'.repeat(243)}@example.com`
    expect(long).toHaveLength(255)
    expect(emailSchema.safeParse(long).success).toBe(false)
    expect(emailSchema.safeParse(long.slice(1)).success).toBe(true)
  })

  it('accepts a password of eight to two hundred fifty-six characters', () => {
    expect(passwordSchema.safeParse('a'.repeat(7)).success).toBe(false)
    expect(passwordSchema.safeParse('a'.repeat(8)).success).toBe(true)
    expect(passwordSchema.safeParse('a'.repeat(256)).success).toBe(true)
    expect(passwordSchema.safeParse('a'.repeat(257)).success).toBe(false)
  })

  it('keeps the surrounding spaces of a password', () => {
    expect(passwordSchema.parse(' padded password ')).toBe(' padded password ')
  })

  it('trims a name and caps it at a hundred characters', () => {
    expect(nameSchema.parse('  Ada  ')).toBe('Ada')
    expect(nameSchema.safeParse('   ').success).toBe(false)
    expect(nameSchema.safeParse('a'.repeat(100)).success).toBe(true)
    expect(nameSchema.safeParse('a'.repeat(101)).success).toBe(false)
  })

  it('accepts exactly six digits as a code', () => {
    expect(codeSchema.parse('123456')).toBe('123456')
    expect(codeSchema.safeParse('12345').success).toBe(false)
    expect(codeSchema.safeParse('1234567').success).toBe(false)
    expect(codeSchema.safeParse('12345a').success).toBe(false)
  })

  it('composes the sign-in form values', () => {
    const parsed = signInSchema.parse({
      email: ' Ada@Example.com ',
      password: 'correct horse',
    })

    expect(parsed).toEqual({ email: 'Ada@Example.com', password: 'correct horse' })
    expect(
      signInSchema.safeParse({ email: 'ada@example.com', password: 'short' }).success,
    ).toBe(false)
  })

  it('composes the sign-up form values with trimmed names', () => {
    const parsed = signUpSchema.parse({
      firstName: ' Ada ',
      lastName: ' Lovelace ',
      email: 'ada@example.com',
      password: 'correct horse',
    })

    expect(parsed).toEqual({
      firstName: 'Ada',
      lastName: 'Lovelace',
      email: 'ada@example.com',
      password: 'correct horse',
    })
    expect(
      signUpSchema.safeParse({
        firstName: '',
        lastName: 'Lovelace',
        email: 'ada@example.com',
        password: 'correct horse',
      }).success,
    ).toBe(false)
  })

  it('composes the single-field forms', () => {
    expect(verifyCodeSchema.parse({ code: '123456' })).toEqual({ code: '123456' })
    expect(emailOnlySchema.parse({ email: ' ada@example.com ' })).toEqual({
      email: 'ada@example.com',
    })
    expect(newPasswordSchema.safeParse({ password: 'short' }).success).toBe(false)
  })
})
