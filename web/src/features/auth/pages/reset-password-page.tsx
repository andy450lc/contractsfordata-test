import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button, buttonVariants } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ApiError } from '@/lib/api/client'
import { adoptSession } from '@/lib/auth/session'

import { useResetPassword } from '../api'
import { AuthShell } from '../components/auth-shell'
import { authNotice, noticeFor, weakPasswordDetail, type Notice } from '../messages'
import { newPasswordSchema, type NewPasswordValues } from '../schemas'

// ResetPasswordPage serves the reset link to every visitor. A visitor
// who is already signed in gets the link's account after a successful
// reset.
export function ResetPasswordPage() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const [token] = useState(() => searchParams.get('token'))
  const [expired, setExpired] = useState(false)
  const [notice, setNotice] = useState<Notice | null>(null)
  const reset = useResetPassword()
  const form = useForm<NewPasswordValues>({
    resolver: zodResolver(newPasswordSchema),
    defaultValues: { password: '' },
  })

  useEffect(() => {
    if (!searchParams.has('token')) return
    setSearchParams({}, { replace: true })
  }, [searchParams, setSearchParams])

  function submit(values: NewPasswordValues) {
    if (token === null) return
    setNotice(null)
    reset.mutate(
      { token, password: values.password },
      {
        onSuccess: (accessToken) => {
          adoptSession(accessToken)
          void navigate('/dashboard', { replace: true })
        },
        onError: (error) => {
          const weak = weakPasswordDetail(error)
          if (weak !== null) {
            form.setError('password', { message: weak })
            return
          }
          if (error instanceof ApiError && error.code === 'invalid_reset_token') {
            setExpired(true)
            return
          }
          setNotice(noticeFor(error))
        },
      },
    )
  }

  if (token === null || expired) {
    const refused = authNotice('invalid_reset_token')
    return (
      <AuthShell title={refused.title} description={refused.description}>
        <Link to="/forgot-password" className={buttonVariants({ className: 'w-full' })}>
          Request a new link
        </Link>
      </AuthShell>
    )
  }

  const passwordError = form.formState.errors.password?.message

  return (
    <AuthShell title="Choose a new password">
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
          <Label htmlFor="reset-password">New password</Label>
          <Input
            id="reset-password"
            type="password"
            autoComplete="new-password"
            aria-invalid={passwordError ? true : undefined}
            aria-describedby="reset-password-hint"
            {...form.register('password')}
          />
          <p id="reset-password-hint" className="text-xs text-muted-foreground">
            At least 8 characters
          </p>
          {passwordError ? (
            <p className="text-xs text-destructive">{passwordError}</p>
          ) : null}
        </div>
        <Button type="submit" className="w-full" disabled={reset.isPending}>
          Save and sign in
        </Button>
      </form>
    </AuthShell>
  )
}
