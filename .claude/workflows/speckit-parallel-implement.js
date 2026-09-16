export const meta = {
  name: 'speckit-parallel-implement',
  description: 'Implement a speckit feature with parallel backend and web tracks',
  whenToUse:
    'After /speckit-tasks, to execute tasks.md faster: a mechanical setup stage, then the web track running beside backend foundations and implementation, then final gates. The orchestrator marks tasks.md and commits.',
  phases: [
    { title: 'Setup', detail: 'contract merge, codegen, shared content', model: 'sonnet' },
    { title: 'Backend', detail: 'foundations, then endpoints and services' },
    { title: 'Web', detail: 'routes, forms, pages' },
    { title: 'Gates', detail: 'full lint, test, and codegen-diff sweep', model: 'sonnet' },
  ],
}

// args shape:
// {
//   featureDir: 'specs/00N-slug',                 required
//   lanes: {
//     setup:    'T001-T004: one-line scope',       required, mechanical, blocks everything
//     backendFoundations: 'T005-T009: ...',        optional, mechanical, blocks backend only
//     backend:  'T010-T0xx: ...',                  required
//     web:      'T0xx-T0yy: ...',                  required
//   },
//   models: { mechanical: 'sonnet', impl: 'opus' } optional overrides
// }
//
// Lane scopes are free text naming the task ids from tasks.md each lane
// owns. The web lane starts as soon as setup finishes. Backend
// foundations never block the web lane.

const mech = args.models?.mechanical ?? 'sonnet'
const impl = args.models?.impl ?? 'opus'
const dir = args.featureDir

const shared = `
You are implementing part of the feature under ${dir} in this repo.

Required reading before writing any code:
1. .specify/memory/constitution.md — BINDING. Principle II: TDD is
   mandatory, the failing test is written and run before the
   implementation. Principle IV comment voice binds ALL comments:
   present tense, active voice, at most two clauses. Banned in
   comments: semicolons, references to specs/FR-numbers/tasks/
   convention docs, design justification, contrastive negation,
   idioms, "simply"/"just".
2. The relevant conventions doc: docs/backend-conventions.md or
   docs/frontend-conventions.md.
3. Everything under ${dir}: spec.md, plan.md, research.md,
   data-model.md, contracts/, tasks.md.
4. The existing code the tasks name — the neighbouring feature slice
   is the style template. A reader must not be able to tell two
   authors apart.

Ground rules:
- Touch ONLY the files your lane owns. Other agents work in parallel
  in the same tree.
- No git write operations. Never edit tasks.md — the orchestrator
  marks progress.
- Backend tests: run via "make -C backend test" (the target disables
  the testcontainers reaper). Focused runs during the TDD loop
  ("go test -run <Name> ./tests/..."), full suite with -race only as
  the lane's final check.
- Web tests: run single files during the loop, the full suite once at
  the end.
- Report per task id: what you did, fail-first evidence, final
  lint/test output summaries, anything incomplete and why.
`

phase('Setup')
const setup = await agent(
  `${shared}
Your lane: SETUP. It blocks every other lane, so finish fast and do
nothing beyond your scope.
Tasks: ${args.lanes.setup}
Verify both apps compile and existing suites stay green, then report.`,
  { label: 'setup', phase: 'Setup', model: mech },
)
if (setup === null) throw new Error('setup lane failed')

const backendLane = async () => {
  let foundations = ''
  if (args.lanes.backendFoundations) {
    foundations = await agent(
      `${shared}
Your lane: BACKEND FOUNDATIONS (migrations, models, store, DTOs).
Files: backend/** only.
Tasks: ${args.lanes.backendFoundations}
Setup lane report:\n${setup}`,
      { label: 'backend:foundations', phase: 'Backend', model: mech },
    )
    if (foundations === null) throw new Error('backend foundations failed')
  }
  return agent(
    `${shared}
Your lane: BACKEND IMPLEMENTATION (tests first, then services,
controllers, routes). Files: backend/** only.
Tasks: ${args.lanes.backend}
Setup lane report:\n${setup}
Foundations report:\n${foundations}`,
    { label: 'backend:impl', phase: 'Backend', model: impl },
  )
}

const webLane = () =>
  agent(
    `${shared}
Your lane: WEB IMPLEMENTATION (tests first, then routes, hooks,
schemas, pages). Files: web/** only. Develop against MSW mocks per
the contract.
Tasks: ${args.lanes.web}
Setup lane report:\n${setup}`,
    { label: 'web:impl', phase: 'Web', model: impl },
  )

const [backend, web] = await parallel([backendLane, webLane])

phase('Gates')
const gates = await agent(
  `${shared}
Your lane: FINAL GATES on the combined tree. Run the full backend gate
set (make -C backend lint test), the full web gate set (npm --prefix
web run typecheck, lint, test, format:check), and every codegen diff
check (sqlc, web generate:api — regeneration must be a no-op). Fix
only trivial mechanical findings yourself. Report every result
honestly, including coverage on new code.`,
  { label: 'gates', phase: 'Gates', model: mech },
)

return { setup, backend, web, gates }
