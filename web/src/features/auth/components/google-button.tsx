import { buttonVariants } from '@/components/ui/button'
import { googleSignInUrl } from '@/lib/auth/session'

// GoogleButton links to the API's login route, which redirects to
// Google. Following the link is a top-level navigation out of the app.
export function GoogleButton({ returnTo }: { returnTo: string }) {
  return (
    <a
      className={buttonVariants({ variant: 'outline', className: 'w-full' })}
      href={googleSignInUrl(returnTo)}
    >
      Continue with Google
    </a>
  )
}
