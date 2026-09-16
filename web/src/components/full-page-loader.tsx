import { Loader2 } from 'lucide-react'

// FullPageLoader is the shared wait state for route guards and transition
// pages. It shows only a spinner. Screen readers still get a label.
export function FullPageLoader() {
  return (
    <main className="flex min-h-svh items-center justify-center p-4">
      <span role="status" aria-live="polite">
        <Loader2
          aria-hidden="true"
          className="size-6 animate-spin text-muted-foreground"
        />
        <span className="sr-only">Loading</span>
      </span>
    </main>
  )
}
