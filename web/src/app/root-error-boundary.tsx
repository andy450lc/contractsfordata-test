import { Button } from '@/components/ui/button'

export function RootErrorBoundary() {
  return (
    <main className="flex min-h-svh flex-col items-center justify-center gap-4 p-6 text-center">
      <h1 className="text-2xl font-semibold">Something went wrong</h1>
      <p className="text-muted-foreground">
        An unexpected error occurred. Reloading usually fixes it.
      </p>
      <Button onClick={() => window.location.reload()}>Reload the app</Button>
    </main>
  )
}
