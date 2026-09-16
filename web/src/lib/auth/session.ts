import { safeReturnTo } from '@/features/auth/return-target'
import type { components } from '@/lib/api/schema'
import { api } from '@/lib/api/client'
import { env } from '@/lib/env'
import { useSessionStore } from '@/stores/session'

type AccessToken = components['schemas']['AccessToken']

// refreshBufferMs is how long before expiry the next refresh runs.
const refreshBufferMs = 60_000
// minRefreshDelayMs is the shortest wait before a scheduled refresh.
const minRefreshDelayMs = 5_000
// retryDelayMs is the wait before retrying after the API was unreachable.
const retryDelayMs = 30_000
// signedOutKey is the localStorage key other tabs watch. It carries a
// timestamp and never a token.
const signedOutKey = 'sow.signed-out'

let inFlight: Promise<boolean> | null = null
let timer: ReturnType<typeof setTimeout> | null = null

function clearTimer() {
  if (timer !== null) {
    clearTimeout(timer)
    timer = null
  }
}

function schedule(delayMs: number) {
  clearTimer()
  timer = setTimeout(() => {
    timer = null
    void refresh()
  }, delayMs)
}

function scheduleBeforeExpiry(expiresAt: number) {
  const delay = Math.max(expiresAt - Date.now() - refreshBufferMs, minRefreshDelayMs)
  schedule(delay)
}

async function doRefresh(): Promise<boolean> {
  const store = useSessionStore.getState()

  const { data, response, error } = await api
    .POST('/auth/refresh', { credentials: 'include' })
    .catch(() => ({ data: undefined, response: undefined, error: undefined }))

  if (data) {
    store.setAuthenticated(data.access_token, data.expires_at)
    scheduleBeforeExpiry(data.expires_at)
    return true
  }

  const refused = response !== undefined && response.status < 500
  if (refused || error !== undefined) {
    clearTimer()
    store.setAnonymous(store.status === 'authenticated' ? 'session_expired' : null)
    return false
  }

  if (store.status === 'unknown') {
    store.setAnonymous()
  } else if (store.status === 'authenticated') {
    schedule(retryDelayMs)
  }
  return false
}

// refresh exchanges the session cookie for a fresh access token. Concurrent
// callers share one request. Returns whether the session is authenticated
// afterwards.
export function refresh(): Promise<boolean> {
  if (inFlight === null) {
    inFlight = doRefresh().finally(() => {
      inFlight = null
    })
  }
  return inFlight
}

// bootstrap runs the first refresh of an app load. Later calls are no-ops.
export async function bootstrap(): Promise<void> {
  if (useSessionStore.getState().status !== 'unknown') return
  await refresh()
}

// getAccessToken returns a token that stays valid for at least the
// refresh buffer, refreshing first when needed. Null means anonymous.
export async function getAccessToken(): Promise<string | null> {
  const { status, accessToken, expiresAt } = useSessionStore.getState()
  const fresh =
    status === 'authenticated' &&
    accessToken !== null &&
    expiresAt !== null &&
    expiresAt - Date.now() > refreshBufferMs
  if (fresh) return accessToken

  await refresh()
  return useSessionStore.getState().accessToken
}

// googleSignInUrl builds the API's login URL for Google and a validated
// return path.
export function googleSignInUrl(returnTo: unknown): string {
  const params = new URLSearchParams({
    provider: 'google',
    returnTo: safeReturnTo(returnTo),
  })
  return `${env.VITE_API_URL}/v1/auth/login?${params.toString()}`
}

// adoptSession takes the token a credential exchange returned. The API
// sealed the session cookie on the same response.
export function adoptSession(token: AccessToken): void {
  useSessionStore.getState().setAuthenticated(token.access_token, token.expires_at)
  scheduleBeforeExpiry(token.expires_at)
  try {
    window.localStorage.removeItem(signedOutKey)
  } catch {
    // Storage can be unavailable. The sentinel is a convenience only.
  }
}

// signOut ends the session on the API and clears it here. The local
// session clears even when the API call fails. Other tabs learn about it
// through the storage sentinel.
export async function signOut(): Promise<void> {
  clearTimer()
  await api.POST('/auth/logout', { credentials: 'include' }).catch(() => undefined)
  useSessionStore.getState().setAnonymous()
  try {
    window.localStorage.setItem(signedOutKey, String(Date.now()))
  } catch {
    // Storage can be unavailable. The sentinel is a convenience only.
  }
}

// onStorage clears this tab's session when another tab signs out.
function onStorage(event: StorageEvent) {
  if (event.key !== signedOutKey || event.newValue === null) return
  clearTimer()
  useSessionStore.getState().setAnonymous()
}

if (typeof window !== 'undefined') {
  window.addEventListener('storage', onStorage)
}

// resetSession clears timers, in-flight work, and the store. Tests call it
// between cases.
export function resetSession(): void {
  clearTimer()
  inFlight = null
  useSessionStore.getState().reset()
}

// isSessionEndpoint reports whether a request targets the cookie-based
// session endpoints, which never carry a bearer token.
function isSessionEndpoint(url: string): boolean {
  return url.includes('/v1/auth/')
}

// retryable remembers a clone of each bearer request. A refused call is
// replayed once with a fresh token.
const retryable = new WeakMap<Request, Request>()

api.use({
  async onRequest({ request }) {
    if (isSessionEndpoint(request.url)) return request

    const token = await getAccessToken()
    if (token !== null) {
      request.headers.set('Authorization', `Bearer ${token}`)
      retryable.set(request, request.clone())
    }
    return request
  },
  async onResponse({ request, response }) {
    const clone = retryable.get(request)
    if (response.status !== 401 || clone === undefined) return response

    retryable.delete(request)
    const ok = await refresh()
    const token = useSessionStore.getState().accessToken
    if (!ok || token === null) return response

    clone.headers.set('Authorization', `Bearer ${token}`)
    return globalThis.fetch(clone)
  },
})
