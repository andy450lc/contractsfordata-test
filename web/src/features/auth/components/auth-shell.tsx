import type { ReactNode } from 'react'

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

interface AuthShellProps {
  title: string
  description?: string
  children: ReactNode
  footer?: ReactNode
}

// AuthShell is the centred card the sign-in, sign-up, and password pages
// share. The footer holds the link to the page's counterpart.
export function AuthShell({ title, description, children, footer }: AuthShellProps) {
  return (
    <main className="flex min-h-svh flex-col items-center justify-center gap-6 p-4">
      <h1 className="text-2xl font-semibold tracking-tight">SoW</h1>
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle className="text-lg">{title}</CardTitle>
          {description ? <CardDescription>{description}</CardDescription> : null}
        </CardHeader>
        <CardContent className="flex flex-col gap-6">
          {children}
          {footer ? (
            <p className="text-center text-sm text-muted-foreground">{footer}</p>
          ) : null}
        </CardContent>
      </Card>
    </main>
  )
}
