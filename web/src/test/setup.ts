import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterAll, afterEach, beforeAll } from 'vitest'

import { clearRecordedRequests, server } from './msw'

import { resetSession } from '@/lib/auth/session'
import { queryClient } from '@/lib/query-client'

// jsdom lacks the layout APIs Radix primitives call on mount.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (typeof window !== 'undefined') {
  window.ResizeObserver ??= ResizeObserverStub
  Element.prototype.hasPointerCapture ??= () => false
  Element.prototype.setPointerCapture ??= () => undefined
  Element.prototype.releasePointerCapture ??= () => undefined
  Element.prototype.scrollIntoView ??= () => undefined
}

// jsdom has no object URLs. The stubs record every URL created and
// revoked so tests can assert on both.
export const objectUrls = { created: [] as string[], revoked: [] as string[] }
let objectUrlCounter = 0
URL.createObjectURL ??= () => ''
URL.revokeObjectURL ??= () => undefined
const originalCreateObjectURL = URL.createObjectURL
const originalRevokeObjectURL = URL.revokeObjectURL

beforeAll(() => {
  server.listen({ onUnhandledRequest: 'error' })
  URL.createObjectURL = () => {
    objectUrlCounter += 1
    const url = `blob:test/${objectUrlCounter}`
    objectUrls.created.push(url)
    return url
  }
  URL.revokeObjectURL = (url: string) => {
    objectUrls.revoked.push(url)
  }
})

afterEach(() => {
  cleanup()
  objectUrls.created.length = 0
  objectUrls.revoked.length = 0
  server.resetHandlers()
  clearRecordedRequests()
  queryClient.clear()
  resetSession()
})

afterAll(() => {
  server.close()
  URL.createObjectURL = originalCreateObjectURL
  URL.revokeObjectURL = originalRevokeObjectURL
})
