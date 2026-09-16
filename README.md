# SoW

Create a Statement of Work from a standard template, send it to the people
who need to sign, and keep the signed copy with its evidence.

## Layout

| Folder | What it is | Runs on |
|---|---|---|
| `contracts/` | `openapi.yaml`, the API source of truth | — |
| `backend/` | Go API (echo v5, fx, pgx, sqlc, goose) | Railway |
| `web/` | React SPA (Vite, TypeScript, shadcn/ui) | Cloudflare Workers static assets |
| `marketing/` | Landing site (Astro, static output, Tailwind) | Cloudflare Workers static assets |
| `docs/` | Binding conventions, deployment, project context | — |
| `.specify/` | Constitution and Spec Kit configuration | — |
| `specs/` | One folder per feature: spec, plan, tasks, contracts | — |

The rules live in `.specify/memory/constitution.md` and the documents it
binds under `docs/`.

## Run it locally

```bash
cd backend && make db-up && make migrate-up && make dev
```

```bash
cd web && npm install && npm run dev
```

```bash
cd marketing && cp .env.example .env && npm install && npm run dev
```

## Check it

```bash
cd backend && make lint && go build ./... && make test
```

```bash
cd web && npm run typecheck && npm run lint && npm run format:check && npm run test:coverage && npm run build
```

```bash
cd marketing && npm run check && npm run format:check && npm run build
```
