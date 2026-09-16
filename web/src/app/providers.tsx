import { QueryClientProvider } from '@tanstack/react-query'
import { useEffect } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'

import { TooltipProvider } from '@/components/ui/tooltip'
import { bootstrap } from '@/lib/auth/session'
import { queryClient } from '@/lib/query-client'
import { useSessionStore } from '@/stores/session'

// useSessionExpiryRedirect sends the user to the sign-in page with a
// notice when a live session stops being accepted.
function useSessionExpiryRedirect() {
  const navigate = useNavigate()
  const location = useLocation()
  const reason = useSessionStore((state) => state.reason)

  useEffect(() => {
    if (reason !== 'session_expired' || location.pathname === '/') return
    useSessionStore.getState().setAnonymous()
    void navigate('/?error=session_expired', { replace: true })
  }, [location.pathname, navigate, reason])
}

// Providers is the root layout route element. The query client wraps
// every page, and the session bootstraps once on mount.
export function Providers() {
  useSessionExpiryRedirect()

  useEffect(() => {
    void bootstrap()
  }, [])

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <Outlet />
      </TooltipProvider>
    </QueryClientProvider>
  )
}
