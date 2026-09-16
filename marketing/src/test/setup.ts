import '@testing-library/jest-dom/vitest'
import { afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'

class ResizeObserverMock {
  static instances: ResizeObserverMock[] = []

  readonly callback: ResizeObserverCallback

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback
    ResizeObserverMock.instances.push(this)
  }

  disconnect() {}
  observe() {}
  unobserve() {}
}

Object.defineProperty(globalThis, 'ResizeObserver', {
  configurable: true,
  value: ResizeObserverMock,
})

Object.defineProperties(HTMLDialogElement.prototype, {
  showModal: {
    configurable: true,
    value() {
      this.open = true
    },
  },
  close: {
    configurable: true,
    value() {
      this.open = false
      this.dispatchEvent(new Event('close'))
    },
  },
})

Object.defineProperty(URL, 'createObjectURL', {
  configurable: true,
  value: vi.fn(() => 'blob:template'),
})
Object.defineProperty(URL, 'revokeObjectURL', {
  configurable: true,
  value: vi.fn(),
})

afterEach(() => {
  cleanup()
  document.body.replaceChildren()
  ResizeObserverMock.instances = []
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})

export { ResizeObserverMock }
