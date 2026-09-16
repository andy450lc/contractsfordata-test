import { Navigate, Outlet, useLocation } from 'react-router-dom'

import { FullPageLoader } from '@/components/full-page-loader'
import { useSessionStore } from '@/stores/session'

// ProtectedLayout is the single route guard. Authenticated pages nest
// under it. An anonymous visitor goes to the sign-in page with the
// intended destination preserved in the redirect param.
export function ProtectedLayout() {
  const status = useSessionStore((state) => state.status)
  const location = useLocation()

  if (status === 'unknown') {
    return <FullPageLoader />
  }

  if (status === 'anonymous') {
    const destination = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`/?redirect=${destination}`} replace />
  }

  return <Outlet />
}
