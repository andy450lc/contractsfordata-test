# SoW Web

React SPA for SoW. Governed by the constitution
(`.specify/memory/constitution.md`) and `docs/frontend-conventions.md`.

## Architecture

```
src/
  main.tsx                 entry: StrictMode + RouterProvider
  app/                     router.tsx (the whole page inventory), providers.tsx
                           (query client, session bootstrap, expiry redirect),
                           root-error-boundary.tsx
  lib/env.ts               zod-validated VITE_* (only VITE_API_URL)
  lib/api/client.ts        openapi-fetch client + ApiError; schema.d.ts is
                           generated from contracts/openapi.yaml
  lib/auth/session.ts      bootstrap, single-flight refresh, scheduling,
                           bearer injection with one refresh-and-retry on 401,
                           sign-out with the cross-tab sentinel
  stores/session.ts        Zustand: in-memory access token and status
  features/auth/           sign-in, sign-up, forgot/reset-password pages,
                           the shared verify-code step, callback page;
                           ProtectedLayout (the single route guard); useMe
  features/dashboard/      the signed-in landing page
  components/ui/           shadcn primitives (generated, then owned)
  test/                    MSW server (anonymous by default), render helper
```

Sessions: the API owns the refresh cookie. The app never talks to the
identity provider directly — sign-in and sign-up post JSON straight to
the API's `/v1/auth/*` endpoints, and "Continue with Google" is the only
navigation that leaves the app for WorkOS. `POST /v1/auth/refresh` with
credentials turns the cookie into an access token held in memory only.

## Commands

```bash
npm install
npm run dev            # http://localhost:5173, needs .env.local with VITE_API_URL
npm run generate:api   # regenerate src/lib/api/schema.d.ts from the contract
npm run typecheck && npm run lint && npm run format:check
npm run test:coverage
npm run build && npx wrangler deploy --dry-run
```
