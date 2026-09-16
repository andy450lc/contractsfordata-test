// safeReturnTo validates a post-authentication destination. Values
// arrive from the redirect query param or from the callback, both of
// which are untrusted input. Only same-app absolute paths pass.
// Anything else falls back to the dashboard.
export function safeReturnTo(value: unknown): string {
  if (typeof value !== 'string') return '/dashboard'
  if (!value.startsWith('/') || value.startsWith('//')) return '/dashboard'
  if (value.includes('\\')) return '/dashboard'
  return value
}

// withRedirect builds a link to another auth page that carries the
// current destination. A visitor who asked for no destination gets the
// bare path.
export function withRedirect(path: string, redirect: string | null): string {
  if (redirect === null) return path
  return `${path}?redirect=${encodeURIComponent(safeReturnTo(redirect))}`
}
