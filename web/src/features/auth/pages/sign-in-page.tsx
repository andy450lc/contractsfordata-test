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

import { useSignIn } from '../api'
import { AuthShell } from '../components/auth-shell'
import { GoogleButton } from '../components/google-button'
import { VerifyCodeForm } from '../components/verify-code-form'
import { authNotice, noticeFor, type Notice } from '../messages'
import { safeReturnTo, withRedirect } from '../return-target'
import { signInSchema, type SignInValues } from '../schemas'

type AccessToken = components['schemas']['AccessToken']

// Step is the form the page shows. The code step carries the address the
// code went to and the token that pairs with it.
type Step = 'form' | { kind: 'verify'; email: string; pendingToken: string }

export function SignInPage() {
  const status = useSessionStore((state) => state.status)
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [step, setStep] = useState<Step>('form')
  const [notice, setNotice] = useState<Notice | null>(null)
  const signIn = useSignIn()
  const form = useForm<SignInValues>({
    resolver: zodResolver(signInSchema),
    defaultValues: { email: '', password: '' },
  })

  const redirect = searchParams.get('redirect')
  const returnTo = safeReturnTo(redirect)
  const errorCode = searchParams.get('error')

  function finish(token: AccessToken) {
    adoptSession(token)
    void navigate(returnTo, { replace: true })
  }

  function submit(values: SignInValues) {
    setNotice(null)
    signIn.mutate(values, {
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
      onError: (error) => setNotice(noticeFor(error)),
    })
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
  const shown = notice ?? (errorCode === null ? null : authNotice(errorCode))

  return (
    <AuthShell
      title="Sign in"
      footer={
        <>
          New to SoW?{' '}
          <Link
            to={withRedirect('/sign-up', redirect)}
            className="font-medium text-foreground underline"
          >
            Create an account
          </Link>
        </>
      }
    >
      {shown && (
        <Alert role={notice ? 'alert' : 'status'}>
          <AlertTitle>{shown.title}</AlertTitle>
          {shown.description ? (
            <AlertDescription>{shown.description}</AlertDescription>
          ) : null}
        </Alert>
      )}
      <form
        onSubmit={form.handleSubmit(submit)}
        noValidate
        className="flex flex-col gap-4"
      >
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sign-in-email">Email</Label>
          <Input
            id="sign-in-email"
            type="email"
            autoComplete="email"
            aria-invalid={errors.email ? true : undefined}
            aria-describedby={errors.email ? 'sign-in-email-error' : undefined}
            {...form.register('email')}
          />
          {errors.email ? (
            <p id="sign-in-email-error" className="text-xs text-destructive">
              {errors.email.message}
            </p>
          ) : null}
        </div>
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="sign-in-password">Password</Label>
          <Input
            id="sign-in-password"
            type="password"
            autoComplete="current-password"
            aria-invalid={errors.password ? true : undefined}
            aria-describedby={errors.password ? 'sign-in-password-error' : undefined}
            {...form.register('password')}
          />
          {errors.password ? (
            <p id="sign-in-password-error" className="text-xs text-destructive">
              {errors.password.message}
            </p>
          ) : null}
        </div>
        <Button type="submit" className="w-full" disabled={signIn.isPending}>
          Sign in
        </Button>
      </form>
      <div className="flex flex-col items-center gap-4">
        <Link to="/forgot-password" className="text-sm text-muted-foreground underline">
          Forgot password?
        </Link>
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
