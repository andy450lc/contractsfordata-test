import { seedSourceDocument } from '@/features/agreements'

declare global {
  interface Window {
    __sowSeedSourceDocument?: () => string
  }
}

// installDevSeed exposes a console helper in development builds that
// fills the store with the source document's organization and draft.
export function installDevSeed(): void {
  if (!import.meta.env.DEV) return
  window.__sowSeedSourceDocument = seedSourceDocument
}
