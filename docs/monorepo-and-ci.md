# Monorepo Layout & CI

Carried over from the PixelsTwo marketplace on 2026-09-03. The project is a
**monorepo**: two apps in different languages sharing one contract, plus a
static landing page. No monorepo build tooling (Turborepo/Nx/Bazel) — plain
folders and path-filtered CI are the whole architecture.

## Layout

```
sow/
  .specify/            # constitution + Spec Kit specs (sees the whole system)
  contracts/           # openapi.yaml — the API source of truth
  backend/             # Go service (backend-conventions.md applies)
  web/                 # React SPA (frontend-conventions.md applies)
  marketing/           # landing site (Astro, static output)
  docs/                # conventions + architecture docs
  .github/workflows/   # backend.yml, web.yml, marketing.yml — path-filtered
```

- `contracts/openapi.yaml` is owned by neither app. Spec-first flow:
  feature spec → contract change → backend implements, frontend
  regenerates — all in **one branch, one PR**, atomically.
- Each app is self-contained (`backend/go.mod`, `web/package.json`); a
  developer or agent works in one folder without touching the other.
- `marketing/` is an Astro site rendered to static HTML by `astro build`.
  It has its own theme in `marketing/src/styles/global.css` and shares
  only the body typeface and radius scale with `web/`, by convention, not
  by importing across packages.

## CI (GitHub Actions)

- **Per-app workflows with `paths` filters** — a web-only change never
  runs backend CI, and vice versa:

  ```yaml
  # backend.yml
  on:
    pull_request:
      paths: ['backend/**', 'contracts/**']
  # web.yml
  on:
    pull_request:
      paths: ['web/**', 'contracts/**']
  # marketing.yml
  on:
    pull_request:
      paths: ['marketing/**']
  ```

- `contracts/**` appears in **both** app filters: a contract change MUST
  re-verify both sides (backend OpenAPI diff gate, frontend client regen
  gate) in the same PR.
- **Required-checks gotcha** (the standard monorepo/Actions trap): a
  required status check that never runs blocks merging forever. Handle it
  with `dorny/paths-filter` — one always-running workflow per app that
  detects relevant changes and skips its jobs internally, so a success is
  always reported — or GitHub rulesets that scope required checks by
  path. This MUST be set up when branch protection is enabled, not
  discovered afterward.
- Deploys stay independent: each workflow builds and deploys only its own
  app. Sharing a repo does not couple releases.

## Branches

One branch strategy for the whole repo (constitution rule): Spec Kit
feature branches as `feature/NNN-kebab-slug` (e.g.
`feature/001-create-agreement`), bugfixes as `fix/kebab-slug`. A branch
may touch contracts, backend, and web together — that is the point of the
monorepo.
