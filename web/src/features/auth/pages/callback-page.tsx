import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { safeReturnTo } from '../return-target'

import { FullPageLoader } from '@/components/full-page-loader'
import { refresh } from '@/lib/auth/session'

// CallbackPage hosts the return leg of the Google sign-in flow. The API
// has already set the session cookie. One refresh turns it into an access
// token. Success continues to the return target. A cancelled consent
// screen returns to the sign-in page with a notice.
export function CallbackPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const returnTo = safeReturnTo(searchParams.get('returnTo'))

  useEffect(() => {
    let cancelled = false

    void refresh().then((ok) => {
      if (cancelled) return
      void navigate(ok ? returnTo : '/?error=flow_incomplete', { replace: true })
    })

    return () => {
      cancelled = true
    }
  }, [navigate, returnTo])

  return <FullPageLoader />
}
