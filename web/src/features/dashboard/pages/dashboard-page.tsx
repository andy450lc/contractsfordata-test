import { Link } from 'react-router-dom'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { buttonVariants } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { SignOutButton, useMe } from '@/features/auth'

// DashboardPage greets the signed-in user. It carries no other content in
// this feature.
export function DashboardPage() {
  const me = useMe()

  if (me.isPending) {
    return (
      <main className="mx-auto flex min-h-svh w-full max-w-2xl flex-col gap-6 p-6">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-48" />
      </main>
    )
  }

  if (me.isError || !me.data) {
    return (
      <main className="mx-auto flex min-h-svh w-full max-w-2xl flex-col gap-6 p-6">
        <Alert role="alert">
          <AlertTitle>We couldn&apos;t load your account</AlertTitle>
          <AlertDescription>Reload the page to try again.</AlertDescription>
        </Alert>
        <div>
          <SignOutButton />
        </div>
      </main>
    )
  }

  const displayName = me.data.name.trim() === '' ? me.data.email : me.data.name

  return (
    <main className="mx-auto flex min-h-svh w-full max-w-2xl flex-col gap-6 p-6">
      <header className="flex items-center justify-between gap-4">
        <h1 className="text-2xl font-semibold tracking-tight">Welcome, {displayName}</h1>
        <SignOutButton />
      </header>
      <section className="flex flex-col gap-2 rounded-lg border p-4">
        <p className="text-sm text-muted-foreground">Signed in as</p>
        <p className="font-medium">{me.data.email}</p>
        <p className="text-sm text-muted-foreground">You&apos;re logged in.</p>
      </section>
      <div>
        <Link to="/agreements" className={buttonVariants({ variant: 'default' })}>
          Go to my agreements
        </Link>
      </div>
    </main>
  )
}
