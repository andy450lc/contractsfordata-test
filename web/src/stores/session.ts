import { create } from 'zustand'

export type SessionStatus = 'unknown' | 'authenticated' | 'anonymous'

// SessionReason explains an anonymous status to the sign-in page.
export type SessionReason = 'session_expired' | null

interface SessionState {
  status: SessionStatus
  accessToken: string | null
  expiresAt: number | null
  reason: SessionReason
  setAuthenticated: (accessToken: string, expiresAt: number) => void
  setAnonymous: (reason?: SessionReason) => void
  reset: () => void
}

const initial = {
  status: 'unknown' as SessionStatus,
  accessToken: null,
  expiresAt: null,
  reason: null,
}

// useSessionStore holds the short-lived access token in memory. Nothing
// here is persisted. The refresh cookie lives on the API host.
export const useSessionStore = create<SessionState>()((set) => ({
  ...initial,
  setAuthenticated: (accessToken, expiresAt) =>
    set({ status: 'authenticated', accessToken, expiresAt, reason: null }),
  setAnonymous: (reason = null) =>
    set({ status: 'anonymous', accessToken: null, expiresAt: null, reason }),
  reset: () => set(initial),
}))
