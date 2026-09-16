import { QueryClient } from '@tanstack/react-query'

import { shouldRetry } from '@/lib/api/client'

// queryClient is the app-wide query cache. Tests clear it between
// cases so cached server state never leaks across tests.
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: shouldRetry,
      refetchOnWindowFocus: false,
    },
  },
})
