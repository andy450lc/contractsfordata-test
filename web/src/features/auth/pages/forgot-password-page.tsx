import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, Navigate } from 'react-router-dom'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useSessionStore } from '@/stores/session'

import { useForgotPassword } from '../api'
import { AuthShell } from '../components/auth-shell'
import { noticeFor, type Notice } from '../messages'
import { emailOnlySchema, type EmailOnlyValues } from '../schemas'

export function ForgotPasswordPage() {
  const status = useSessionStore((state) => state.status)
  const [sent, setSent] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)
  const forgot = useForgotPassword()
  const form = useForm<EmailOnlyValues>({
    resolver: zodResolver(emailOnlySchema),
    defaultValues: { email: '' },
  })

  function submit(values: EmailOnlyValues) {
    setNotice(null)
    forgot.mutate(values, {
      onSuccess: () => setSent(true),
      onError: (error) => setNotice(noticeFor(error)),
    })
  }

  if (status === 'authenticated') {
    return <Navigate to="/dashboard" replace />
  }

  const backToSignIn = (
    <Link to="/" className="font-medium text-foreground underline">
      Back to sign in
    </Link>
  )

  if (sent) {
    return (
      <AuthShell title="Check your inbox" footer={backToSignIn}>
        <p className="text-sm text-muted-foreground">
          If that address has an account, we&apos;ve emailed a reset link. It expires in
          15 minutes.
        </p>
      </AuthShell>
    )
  }

  const emailError = form.formState.errors.email?.message

  return (
    <AuthShell title="Forgot your password?" footer={backToSignIn}>
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
        <div className="flex flex-col gap-1.5">
          <Label htmlFor="forgot-email">Email</Label>
          <Input
            id="forgot-email"
            type="email"
            autoComplete="email"
            aria-invalid={emailError ? true : undefined}
            aria-describedby={emailError ? 'forgot-email-error' : undefined}
            {...form.register('email')}
          />
          {emailError ? (
            <p id="forgot-email-error" className="text-xs text-destructive">
              {emailError}
            </p>
          ) : null}
        </div>
        <Button type="submit" className="w-full" disabled={forgot.isPending}>
          Email me a reset link
        </Button>
      </form>
    </AuthShell>
  )
}
