import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { components } from '@/lib/api/schema'
import { adoptSession } from '@/lib/auth/session'
import { useSessionStore } from '@/stores/session'

import { useSignUp } from '../api'
import { AuthShell } from '../components/auth-shell'
import { GoogleButton } from '../components/google-button'
import { VerifyCodeForm } from '../components/verify-code-form'
import { noticeFor, weakPasswordDetail, type Notice } from '../messages'
import { safeReturnTo, withRedirect } from '../return-target'
import { signUpSchema, type SignUpValues } from '../schemas'

type AccessToken = components['schemas']['AccessToken']

// Step is the form the page shows. The code step carries the address the
// code went to and the token that pairs with it.
type Step = 'form' | { kind: 'verify'; email: string; pendingToken: string }

export function SignUpPage() {
  const status = useSessionStore((state) => state.status)
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [step, setStep] = useState<Step>('form')
  const [notice, setNotice] = useState<Notice | null>(null)
  const signUp = useSignUp()
  const form = useForm<SignUpValues>({
    resolver: zodResolver(signUpSchema),
    defaultValues: { firstName: '', lastName: '', email: '', password: '' },
  })

  const redirect = searchParams.get('redirect')
  const returnTo = safeReturnTo(redirect)

  function finish(token: AccessToken) {
    adoptSession(token)
    void navigate(returnTo, { replace: true })
  }

  function submit(values: SignUpValues) {
    setNotice(null)
    signUp.mutate(
      {
        first_name: values.firstName,
        last_name: values.lastName,
        email: values.email,
        password: values.password,
      },
      {
        onSuccess: (result) => {
          if ('pending_token' in result) {
            setStep({
              kind: 'verify',
              email: values.email,
              pendingToken: result.pending_token,
            })
            return
          }
          finish(result)
        },
        onError: (error) => {
          const weak = weakPasswordDetail(error)
          if (weak !== null) {
            form.setError('password', { message: weak })
            return
          }
          setNotice(noticeFor(error))
        },
      },
    )
  }

  if (status === 'authenticated') {
    return <Navigate to={returnTo} replace />
  }

  if (step !== 'form') {
    return (
      <VerifyCodeForm
        email={step.email}
        pendingToken={step.pendingToken}
        onSignedIn={finish}
        onRestart={() => setStep('form')}
      />
    )
  }

  const errors = form.formState.errors

  return (
    <AuthShell
      title="Create an account"
      footer={
        <>
          Already have an account?{' '}
          <Link
            to={withRedirect('/', redirect)}
            className="font-medium text-foreground underline"
          >
            Sign in
          </Link>
        </>
      }
    >
      {notice && (
        <Alert role="alert">
          <AlertTitle>{notice.title}</AlertTitle>
          {notice.description ? (
            <AlertDescription>{notice.description}</AlertDescription>
          ) : null}
        </Alert>
      )}
      <form
        onSubmit={form.handleSubmit(submit)}
        noValidate
        className="flex flex-col gap-4"
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="sign-up-first-name">First name</Label>
            <Input
              id="sign-up-first-name"
              autoComplete="given-name"
              aria-invalid={errors.firstName ? true : undefined}
              aria-describedby={errors.firstName ? 'sign-up-first-name-error' : undefined}
              {...form.register('firstName')}
            />
            {errors.firstName ? (
              <p id="sign-up-first-name-error" className="text-xs text-destructive">
                {errors.firstName.message}
              </p>
            ) : null}
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="sign-up-last-name">Last name</Label>
            <Input
              id="sign-up-last-name"
              autoComplete="family-name"
              aria-invalid={errors.lastName ? true : undefined}
              aria-describedby={errors.lastName ? 'sign-up-last-name-error' : undefined}
              {...form.register('lastName')}
            />
            {errors.lastName ? (
              <p id="sign-up-last-name-error" className="text-xs text-destructive">
                {errors.lastName.message}
              </p>
            ) : null}
          </div>
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sign-up-email">Email</Label>
          <Input
            id="sign-up-email"
            type="email"
            autoComplete="email"
            aria-invalid={errors.email ? true : undefined}
            aria-describedby={errors.email ? 'sign-up-email-error' : undefined}
            {...form.register('email')}
          />
          {errors.email ? (
            <p id="sign-up-email-error" className="text-xs text-destructive">
              {errors.email.message}
            </p>
          ) : null}
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sign-up-password">Password</Label>
          <Input
            id="sign-up-password"
            type="password"
            autoComplete="new-password"
            aria-invalid={errors.password ? true : undefined}
            aria-describedby="sign-up-password-hint"
            {...form.register('password')}
          />
          <p id="sign-up-password-hint" className="text-xs text-muted-foreground">
            At least 8 characters
          </p>
          {errors.password ? (
            <p className="text-xs text-destructive">{errors.password.message}</p>
          ) : null}
        </div>
        <Button type="submit" className="w-full" disabled={signUp.isPending}>
          Create account
        </Button>
      </form>
      <div className="flex flex-col items-center gap-4">
        <div className="flex w-full items-center gap-3 text-xs text-muted-foreground">
          <span className="h-px flex-1 bg-border" />
          or
          <span className="h-px flex-1 bg-border" />
        </div>
        <GoogleButton returnTo={returnTo} />
      </div>
    </AuthShell>
  )
}
