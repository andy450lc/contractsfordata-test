import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { components } from '@/lib/api/schema'

import { useResendVerification, useVerifyEmail } from '../api'
import { authNotice, noticeFor, type Notice } from '../messages'
import { verifyCodeSchema, type VerifyCodeValues } from '../schemas'

import { AuthShell } from './auth-shell'

type AccessToken = components['schemas']['AccessToken']

// cooldownSeconds is how long the resend button stays disabled after a
// code goes out.
const cooldownSeconds = 30

// secondsUntil counts the whole seconds left before a deadline.
function secondsUntil(deadline: number): number {
  return Math.max(0, Math.ceil((deadline - Date.now()) / 1_000))
}

interface VerifyCodeFormProps {
  email: string
  pendingToken: string
  onSignedIn: (token: AccessToken) => void
  onRestart: () => void
}

// VerifyCodeForm is the code step the sign-in and sign-up pages share. It
// exchanges the emailed code for a session and hands the token to the
// page, which adopts it.
export function VerifyCodeForm({
  email,
  pendingToken,
  onSignedIn,
  onRestart,
}: VerifyCodeFormProps) {
  const [notice, setNotice] = useState<Notice | null>(null)
  const [resent, setResent] = useState(false)
  const [readyAt, setReadyAt] = useState(0)
  const [remaining, setRemaining] = useState(0)
  const verify = useVerifyEmail()
  const resend = useResendVerification()
  const form = useForm<VerifyCodeValues>({
    resolver: zodResolver(verifyCodeSchema),
    defaultValues: { code: '' },
  })

  useEffect(() => {
    if (remaining === 0) return
    const timer = setInterval(() => setRemaining(secondsUntil(readyAt)), 1_000)
    return () => clearInterval(timer)
  }, [readyAt, remaining])

  function submit(values: VerifyCodeValues) {
    setNotice(null)
    setResent(false)
    verify.mutate(
      { pending_token: pendingToken, code: values.code },
      {
        onSuccess: (token) => onSignedIn(token),
        onError: (error) => {
          setNotice(noticeFor(error))
          form.reset({ code: '' })
        },
      },
    )
  }

  function sendAgain() {
    setNotice(null)
    resend.mutate(
      { email },
      {
        onSuccess: () => {
          setResent(true)
          setReadyAt(Date.now() + cooldownSeconds * 1_000)
          setRemaining(cooldownSeconds)
        },
        onError: () => setNotice(authNotice('network')),
      },
    )
  }

  const codeError = form.formState.errors.code?.message

  return (
    <AuthShell
      title="Check your inbox"
      description={`We emailed a 6-digit code to ${email}.`}
    >
      {notice && (
        <Alert role="alert">
          <AlertTitle>{notice.title}</AlertTitle>
          {notice.description ? (
            <AlertDescription>{notice.description}</AlertDescription>
          ) : null}
        </Alert>
      )}
      {resent && (
        <p role="status" className="text-sm text-muted-foreground">
          We sent a new code.
        </p>
      )}
      <form
        onSubmit={form.handleSubmit(submit)}
        noValidate
        className="flex flex-col gap-4"
      >
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="verify-code">Code</Label>
          <Input
            id="verify-code"
            inputMode="numeric"
            autoComplete="one-time-code"
            maxLength={6}
            aria-invalid={codeError ? true : undefined}
            aria-describedby={codeError ? 'verify-code-error' : undefined}
            {...form.register('code')}
          />
          {codeError ? (
            <p id="verify-code-error" className="text-xs text-destructive">
              {codeError}
            </p>
          ) : null}
        </div>
        <Button type="submit" className="w-full" disabled={verify.isPending}>
          Verify
        </Button>
      </form>
      <div className="flex flex-col items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          onClick={sendAgain}
          disabled={remaining > 0 || resend.isPending}
        >
          {remaining > 0 ? `Send a new code (${remaining}s)` : 'Send a new code'}
        </Button>
        <Button type="button" variant="link" onClick={onRestart}>
          Use a different email
        </Button>
      </div>
    </AuthShell>
  )
}
