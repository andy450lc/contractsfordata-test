import { Link } from 'react-router-dom'

import { SignOutButton, useMe } from '@/features/auth'

// initialsOf returns up to two initials from a name or email.
function initialsOf(name: string, email: string): string {
  const source = name.trim() === '' ? email : name
  const parts = source.split(/[\s@._-]+/).filter((part) => part !== '')
  return parts
    .slice(0, 2)
    .map((part) => part.charAt(0).toUpperCase())
    .join('')
}

// AppHeader is the dark top bar shared by the agreements pages.
export function AppHeader() {
  const me = useMe()
  const initials = me.data ? initialsOf(me.data.name, me.data.email) : ''

  return (
    <header className="bg-foreground text-background">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between px-4">
        <Link to="/agreements" className="flex items-baseline gap-2">
          <span className="rounded bg-background px-1.5 py-0.5 text-sm font-bold text-foreground">
            SoW
          </span>
          <span className="font-semibold">Send an agreement</span>
          <span className="text-xs text-background/70">by Pixels Two</span>
        </Link>
        <div className="flex items-center gap-3">
          {initials !== '' ? (
            <span
              aria-label={me.data?.email}
              className="flex size-8 items-center justify-center rounded-full bg-background text-xs font-semibold text-foreground"
            >
              {initials}
            </span>
          ) : null}
          <SignOutButton className="text-foreground" />
        </div>
      </div>
    </header>
  )
}
