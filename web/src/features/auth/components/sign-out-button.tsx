import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { signOut } from '@/lib/auth/session'

// SignOutButton ends the session and returns to the sign-in page.
export function SignOutButton({ className }: { className?: string }) {
  const navigate = useNavigate()
  const [pending, setPending] = useState(false)

  async function handleClick() {
    setPending(true)
    await signOut()
    void navigate('/', { replace: true })
  }

  return (
    <Button
      type="button"
      variant="outline"
      className={className}
      disabled={pending}
      onClick={() => void handleClick()}
    >
      Sign out
    </Button>
  )
}
