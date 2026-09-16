import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <main className="flex min-h-svh flex-col items-center justify-center gap-4 p-6 text-center">
      <h1 className="text-2xl font-semibold">Page not found</h1>
      <p className="text-muted-foreground">
        The address you opened doesn&apos;t exist on this site.
      </p>
      <Link to="/" className="font-medium underline">
        Back to sign in
      </Link>
    </main>
  )
}
