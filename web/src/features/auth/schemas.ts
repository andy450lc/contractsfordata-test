import { z } from 'zod'

// emailSchema accepts a well-formed address of at most 254 characters.
// Surrounding spaces are dropped before the address is checked.
export const emailSchema = z
  .string()
  .trim()
  .pipe(
    z
      .email('Enter a valid email address')
      .max(254, 'Enter an email address of 254 characters or fewer'),
  )

// passwordSchema accepts 8 to 256 characters. Spaces count as characters.
export const passwordSchema = z
  .string()
  .min(8, 'Use at least 8 characters')
  .max(256, 'Use 256 characters or fewer')

// nameSchema accepts 1 to 100 characters after trimming.
export const nameSchema = z
  .string()
  .trim()
  .min(1, 'Required')
  .max(100, 'Use 100 characters or fewer')

// codeSchema accepts the six digits of an emailed verification code.
export const codeSchema = z
  .string()
  .trim()
  .regex(/^\d{6}$/, 'Enter the 6-digit code')

export const signInSchema = z.object({
  email: emailSchema,
  password: passwordSchema,
})

export const signUpSchema = z.object({
  firstName: nameSchema,
  lastName: nameSchema,
  email: emailSchema,
  password: passwordSchema,
})

export const verifyCodeSchema = z.object({ code: codeSchema })

export const emailOnlySchema = z.object({ email: emailSchema })

export const newPasswordSchema = z.object({ password: passwordSchema })

export type SignInValues = z.infer<typeof signInSchema>
export type SignUpValues = z.infer<typeof signUpSchema>
export type VerifyCodeValues = z.infer<typeof verifyCodeSchema>
export type EmailOnlyValues = z.infer<typeof emailOnlySchema>
export type NewPasswordValues = z.infer<typeof newPasswordSchema>
