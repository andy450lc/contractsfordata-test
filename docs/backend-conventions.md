# Backend Conventions

Carried over from the PixelsTwo marketplace backend conventions on 2026-09-03.
Those rules came from a 2026-08-27 audit of `~/neuralace/agent-backend` (113 Go
files, ~11k LOC) and are stated as constitution-ready MUST/SHOULD principles for
the SoW stack (echo v5, slog, uber fx, goose, pgx, sqlc, Supabase Postgres,
Cloudflare R2). Deviations and gaps in the reference repo are called out at the
end so we don't copy them.

---

## 1. Project Layout

The backend lives at `backend/` in the monorepo (see
`docs/monorepo-and-ci.md`); all paths below are relative to it.

- Binaries live under `cmd/`, one directory per binary: `cmd/app/main.go` is
  the server; one-off operational scripts get their own dir (e.g.
  `cmd/scripts/`). All importable code lives under `internal/` — nothing else
  at the top level may contain Go code.
- `internal/` is layered, one package per layer:
  - `boot/` — one file per external dependency, each exposing an fx
    constructor (`env.go`, `database.go`, `redis.go`, `otel.go`, `http.go`, …).
  - `config/` — env-tagged config structs only, no logic.
  - `routes/` → `middleware/` → `controllers/` → `services/` → `stores/` →
    `models/` (+ `models/dto/`) — request flow goes strictly downward;
    a layer MUST NOT import a layer above it.
  - `common/` — context keys, constants, sentinel errors.
  - `logging/` — the context-aware slog wrapper (see §5).
  - `interfaces/` — tiny capability interfaces used by generic middleware.
- One file per domain entity per layer, kebab-case with a layer suffix
  (for a SoW entity "agreement": `agreement-routes.go`,
  `agreement-controller.go`, `agreement-service.go`, `agreement-store.go`).
  Finding all code for an entity = grep its name across four files.
- Migrations live in `internal/migrations/` as sequentially numbered
  `NNNNNN_description` up/down SQL pairs, driven by Makefile targets.
  Destructive targets (`down`, `force`) MUST be guarded to `STAGE=dev` only —
  the reference Makefile does this and it's worth keeping verbatim.

## 2. Dependency Injection (fx)

- `main.go` is pure wiring and MUST stay logic-free: a single `fx.New(...)`
  with `fx.Provide` groups in dependency order — bootstrap, metrics,
  middleware, stores, controllers, services — then `fx.Invoke` for route
  registration and server start. Each group gets a `// comment` header.
- Every component is a struct with unexported fields plus a
  `NewX(deps...) *X` constructor; dependencies arrive only through
  constructor parameters. No package-level singletons, no `init()` wiring.
  (Sole tolerated exception in the reference: a shared validator instance.)
- Interface binding is done at the fx layer with
  `fx.Annotate(NewImpl, fx.As(new(Iface)))`, not by constructors returning
  interfaces.
- Lifecycle (listen/serve, pool close, tracer shutdown) is managed through
  `fx.Lifecycle` hooks: bind the listener synchronously in `OnStart` so
  startup fails loudly; `OnStop` joins all shutdowns
  (`errors.Join(e.Shutdown(ctx), tp.Shutdown(ctx))`).

## 3. HTTP Layer

- **Routes**: each entity gets an `AddXRoutes(e, middlewares..., controller)`
  function, registered via `fx.Invoke`. Each route is declared vertically —
  path, handler, then the middleware chain one per line — so the auth and
  validation story of every endpoint is readable at the call site:

  ```go
  e.POST("/agreements/:agreementId/send",
      agreementController.SendAgreement,
      authMiddleware.RequireAuth,
      middleware.ValidateRequest(new(dto.SendAgreementDto)),
      middleware.AuthorizeUserForAgreement[dto.SendAgreementDto](agreementService),
  )
  ```

- **Controllers are thin.** The canonical handler shape is exactly three
  stanzas: (1) extract `ctx`, user id, and the validated request from the
  echo context; (2) one service call with its error block; (3) the response.
  Business logic in a controller is a violation.
- **Validation is middleware, not handler code.**
  `ValidateRequest(new(dto.X))` binds path/query/header/body params, applies
  struct defaults, validates (go-playground/validator), returns structured
  400s (`{"error": "validation_failed", "details": [{field, message}]}`),
  and stores the typed struct in the echo context under a well-known key.
  Handlers re-read it with a type assertion and never re-validate.
  Mis-registration (non-pointer-to-struct) panics at startup, not at
  request time.
- **Authorization is middleware too**, generic over the DTO: a per-resource
  `AuthorizeUserForX[T]` middleware requires the DTO to implement a
  capability interface (e.g. `GetAgreementId() string` from
  `internal/interfaces`), verifies ownership via the service, and caches the
  loaded entity in the context so the handler doesn't re-fetch it. Every
  resource type follows the same pattern: capability interface + generic
  authorizer.
- **DTOs** live in `models/dto`, one file per endpoint family, with
  `json`/`param`/`query` tags plus `validate` tags. Shared params are
  embedded structs (e.g. `WithAgreementIdParamDto`), which is also how DTOs
  pick up the capability interfaces. Request and response types are
  oapi-codegen-generated wherever the spec can express the shape;
  hand-written DTOs remain only for multi-source binding and
  presence-sensitive validation (see §10).
- **Server hardening** is centralized in the server constructor: RequestID,
  OTel middleware, request logger, `Recover()`, secure headers (HSTS,
  nosniff, DENY, CSP `default-src 'none'`, no-referrer), CORS allowlist from
  config, body limit (2M), plus split health endpoints: `/livez` returns a
  static ok (process up — liveness), `/readyz` checks the DB pool and Redis
  (a pod that lost Postgres stops receiving traffic — readiness). New
  projects MUST start from this stack, not accrete it later.
- **All routes live under a `/v1` path prefix** from day 1 (health and
  metrics endpoints excepted). Costs nothing now; gives breaking changes a
  home later.
- **Rate limiting is part of the day-1 stack** (the reference repo left it
  commented out — see gaps). Use echo's `RateLimiter` middleware with a
  sensible per-IP default globally, plus stricter per-route limits on
  expensive or abusable endpoints (auth, search, anything that sends email).
  The in-memory store is per-instance; the moment the service runs more
  than one replica, the limiter state MUST move to Redis so limits are
  enforced globally, not per-pod. Rate-limited responses return 429 and are
  visible in metrics via the status label.

## 4. Data Layer

- Postgres access uses **pgx v5** with `pgxpool`. Pool settings
  (max/min conns, lifetimes) come from config, never hardcoded. The
  database is Supabase Postgres, reached through its connection pooler
  (see `docs/deployment.md` for the pooler-mode gotcha).
- Stores depend on a narrow **`DBTX` interface** (`Exec`/`Query`/`QueryRow`)
  satisfied by both `*pgxpool.Pool` and `pgx.Tx`. Every store has a
  `WithTx(tx) *Store` that returns a copy bound to the transaction. This is
  exactly the interface sqlc generates, so for SoW the rule becomes:
  **all queries go through sqlc-generated code**; hand-written SQL string
  literals in stores (the reference repo's approach) are not allowed.
- **Transactions belong to services, not stores.** The pattern: service
  begins the tx on the pool, `defer tx.Rollback(ctx)`, runs multiple
  `store.WithTx(tx)` calls, commits last. Stores never begin or commit.
- Every store method sets its own `context.WithTimeout` (5s in the
  reference) before touching the database.
- "Not found" is not an error at the store boundary: `pgx.ErrNoRows` maps to
  `(nil, nil)`; callers decide whether absence is exceptional.

## 5. Logging (slog)

- One `logging` package wraps `log/slog`; nothing else logs. All logging
  helpers take `ctx` as the first argument.
- The request middleware creates a request-scoped logger
  (`requestLogger.With(slog.String("request_id", id))`) and stashes it in
  the request context. `logging.FromContext(ctx)` recovers it and enriches
  with `trace_id`/`span_id` (from OTel span context) and any domain ids
  (`user_id`, entity ids) that were attached via `ContextWithUserID`-style
  helpers. Log a message anywhere in the call stack and it automatically
  carries request, trace, and user identity.
- Output is JSON (`slog.NewJSONHandler`) to stdout. Access logs are emitted
  by echo's RequestLogger with full attributes and leveled by outcome:
  5xx/error → `Error`, 4xx → `Warn`, else `Info`. `/livez`, `/readyz`, and
  `/metrics` are skipped.
- Prefer the structured key-value forms (`logging.Error(ctx, "msg", "key",
  val)`) over the `*f` printf variants — the reference repo's `Errorf`
  helpers flatten everything into the message string, which defeats
  structured logging. SoW rule: `*f` variants SHOULD NOT exist.

### The error-log contract: every error log is debuggable

The bar: reading a single error line in production tells you *which
request*, *which user*, *which entity*, *what operation failed*, and *why*
— with a trace id that follows the request across every microservice it
touched. An error log you can't act on is a bug.

- **Mandatory correlation fields on every log line**: `request_id`,
  `trace_id`, `span_id`, and `user_id` once authenticated. These are never
  passed manually — they ride in via `logging.FromContext(ctx)`. This is
  guaranteed by construction with two rules:
  1. All logging goes through the `logging` package with `ctx` as the
     first argument. Bare `slog`, stdlib `log`, and `fmt.Print*` are
     banned outside `logging/` and `main` — enforced with a linter rule
     (`depguard`/`forbidigo` in golangci-lint), not convention.
  2. `ctx` is threaded through every function in the request path, no
     exceptions. `context.Background()` inside a request flow severs the
     correlation chain and is a review-blocking defect (background jobs
     start fresh contexts deliberately and attach their own job id).
- **Attach domain ids to the context as soon as they're known**, using
  `ContextWithUserID`-style helpers (auth middleware sets `user_id`,
  resource authorizers set the entity id). Every subsequent log line in
  that request then carries them automatically — including logs written
  three layers down that never heard of the entity.
- **Error content**: the message names the operation that failed
  ("create agreement", not "error"); the error itself goes in a structured
  `"error"` attr (not interpolated into the message); inputs needed to
  reproduce go in as typed attrs (`"agreement_id", id` — ids, never full
  payloads, which may hold PII).
- **Log an error exactly once, at the layer that handles it.** Lower
  layers wrap and return (`fmt.Errorf("fetching agreement %s: %w", id,
  err)`) so the story accumulates in the error chain; the centralized echo
  error handler (or the service that swallows/recovers) writes the single
  log line. The reference repo logs at store AND controller level — that
  doubles every error in the logs and makes counting real error rates
  impossible. Don't copy it.

### "Which line failed?" — tracebacks in Go

Go errors carry no stack (unlike Java exceptions). SoW's answer is
the wrap chain, not stack capture:

- **Every cross-package error return is wrapped with operation context**:
  `fmt.Errorf("creating agreement %s: %w", id, err)`. Done universally, the
  final logged error is a semantic traceback with runtime values in it —
  `creating agreement 7f3a…: inserting agreement row: ERROR: duplicate key…` —
  which localizes failure to the function level and often beats a stack
  trace. Enforced with `wrapcheck` in golangci-lint; a bare `return err`
  across a package boundary is a lint failure, because one lazy return
  puts a hole in every traceback that passes through it.
- **`slog.HandlerOptions{AddSource: true}`** on the JSON handler stamps
  each log line with the `file:line` of the log call site. This locates
  every scattered `Info`/`Warn`/`Debug` log precisely. Note: on *error*
  lines it always points at the central error handler (the one place
  errors are logged), so for errors the wrap chain — not `AddSource` — is
  what locates the failure.
- **Stack traces for unexpected errors** (`cockroachdb/errors`;
  `pkg/errors` is archived). Stacks must be captured where the error is
  born — by the time an error reaches the central handler, the failing
  frames have returned and the stack is gone. Unexpected errors always
  enter our code from outside (pgx, redis, HTTP clients, stdlib), and
  always at the lowest layer. Policy:
  - Stores and external-service clients wrap incoming external errors
    with `errors.Wrap(err, "inserting agreement row")` from
    `cockroachdb/errors` — capturing the stack one frame from the origin.
    All layers above wrap with plain `fmt.Errorf("...: %w", err)` as
    usual. `wrapcheck` already forces wrapping at exactly these
    boundaries; this only changes which wrap function the lowest layer
    uses.
  - Domain sentinels stay plain `errors.New` — expected errors
    (`ErrAgreementNotFound`) never need or get stacks.
  - The central echo error handler branches: known sentinel or
    `*echo.HTTPError` → mapped status, single-line log, no stack.
    Anything else → unexpected → generic 500, and log with `%+v`, which
    prints the captured stack. Stacks therefore appear only on genuine
    surprises, never as noise on routine 4xx.
  - Panics already produce full tracebacks via `Recover()` middleware.

### Cross-service correlation

- Every outbound HTTP client MUST come from the fx-provided client factory,
  which wraps transports in `otelhttp.NewTransport`. That injects the W3C
  `traceparent` header automatically, so the downstream service joins the
  same trace and its logs carry the same `trace_id`. Constructing a bare
  `http.Client`/`http.Get` in a service is banned — it silently breaks the
  correlation chain.
- Echo's `RequestID()` middleware honors an incoming `X-Request-ID`, so
  internal service-to-service calls SHOULD also forward `X-Request-ID` (the
  client factory is the place to add this) — then both `request_id` and
  `trace_id` line up across services.
- Debugging flow this buys: see an error in service B → copy its
  `trace_id` → search all services' logs for it → the full request story,
  in order, including the originating request in service A and the exact
  user. If any log line in that chain is missing the trace id, the rules
  above were violated.

## 6. Errors & Config

- Domain sentinel errors are `var ErrFooBar = errors.New(...)` declarations
  centralized in `common/errors.go`, grouped by domain with blank lines
  between groups. **Naming is standard Go `ErrXxx` — non-negotiable**
  (e.g. `ErrAgreementNotFound`, `ErrInvalidStateTransition`); the reference
  repo's SCREAMING_SNAKE names are explicitly rejected. Compare with
  `errors.Is`, wrap with `fmt.Errorf("...: %w", err)`.
- Config is a single `AppConfig` struct in `config/`, populated by
  `caarlos0/env` tags: `env:"NAME,required"`, `envDefault:`, and
  `validate:"required_if=Stage prod"` for stage-conditional secrets.
  `.env` via godotenv is loaded only when `STAGE != prod`. Config is parsed
  AND validated at boot; the app refuses to start on missing config rather
  than failing at first use.
- Context keys, middleware keys, and path-param names are named constants in
  `common/constants.go` — never string literals at call sites.

## 7. Complexity & Function Style

Measured with gocyclo over `internal/` + `cmd/`:

- **Average cyclomatic complexity: 3.46.** ~95% of functions are ≤ 9.
  Only six functions exceed 12; the worst (`CardService.ActOnCard`, 27, and
  a JSON `UnmarshalJSON`, 21) are the acknowledged pain points, not the
  norm.
- Constitution rule derived from this: cyclomatic complexity per function
  MUST be ≤ 10; up to 15 is allowed only with a written justification in
  review; above 15 is refactor-first. Enforce with `gocyclo -over 10` (or
  golangci-lint's `gocyclo`/`cyclop`) in CI.

**Blank lines separate logical stanzas inside functions.** The style is
consistent across every layer and reads as paragraphs:

1. setup (derive ctx/timeout, extract inputs) — blank line —
2. declaration (query text, struct literal being built) — blank line —
3. the call **with its error block attached** (no blank line between a call
   and its `if err != nil`) — blank line after the error block closes —
4. post-processing — blank line —
5. return.

```go
func (s *UserStore) CreateUser(ctx context.Context, user *models.User) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    row, err := s.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
        ID:       user.ID,
        Email:    user.Email,
        FullName: user.FullName,
    })
    if err != nil {
        return errors.Wrap(err, "inserting user row") // cockroachdb/errors: stack at origin
    }

    user.CreatedAt = row.CreatedAt.UnixMilli()
    return nil
}
```

(Note what the example does NOT do, per §5: no logging in the store — it
wraps and returns; the central error handler logs once with full context.)

- Comments are sparse and only state non-obvious constraints (e.g. "This
  updates the *CONTENTS* of actions… It does *NOT* add/delete actions"),
  placed above the stanza or route they govern. No narration comments.
- **Comment voice** (constitution 1.1.0, expanded 1.2.0): a comment
  describes what the code does, in isolation. Doc comments on exported
  identifiers keep the godoc name-first form ("NewX builds…") and state
  behavior plainly after that opening — e.g.
  `// NewDatabasePool builds the pgx pool from config. Pings and fails
  fast at startup on errors. Closes the pool on OnStop.`
  Sentence form: simple sentences, present tense, active voice, one idea
  per sentence. A compound sentence has at most two clauses. Contrastive
  negation is banned ("X, never Y", "X, not Z") — state the positive
  fact, and give a negative fact that matters its own sentence.
  Conditions come before actions. Word choice: plain words in their
  primary sense, no idioms, no "simply"/"easy"/"just", at most two
  stacked noun modifiers, serial comma, American spelling.
  Banned in all comments: semicolons (write separate sentences),
  references to specs/requirements/convention docs (FR-numbers, section
  citations, research ids — traceability lives in `specs/`), and design
  justification (contrasts with alternatives, cross-project context,
  compliance notes — rationale lives in specs, plans, and PRs).
  Line wrapping: a sentence that fits on one line stays on one line, and
  a wrapped sentence leaves no continuation line of one to three words.
  The Google developer documentation style guide
  (https://developers.google.com/style) is the tie-breaker for anything
  unstated. A comment earns its place only by stating behavior or a
  constraint the code cannot show. Everything else is deleted, not
  reworded.
- Multi-arg calls and struct literals that don't fit one line are broken
  one-field-per-line with trailing commas (see the fx wiring and
  RequestLoggerConfig).

## 8. Observability (day 1, non-negotiable)

Three pillars, all wired at boot before the first feature ships: structured
logs (§5), traces, and Prometheus metrics.

### Tracing

- OTel tracing wired at boot (`otelecho` middleware, SDK tracer provider,
  shutdown tied to the fx lifecycle); trace/span ids injected into every
  log line so a slow request can be followed from metric → log → trace.

### Metrics: the RED method

For an API service, three questions cover almost everything: **R**ate (how
many requests), **E**rrors (how many failed), **D**uration (how long they
took). One well-labeled histogram answers all three — this is the pattern
the reference repo's `metrics` package already implements and SoW
adopts wholesale:

```go
RequestDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name:    "http_request_duration_seconds",
        Help:    "Duration of HTTP requests in seconds",
        Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
    },
    []string{"method", "path", "status"},
)
```

A histogram automatically exposes `_count` and `_sum` alongside the
buckets, so from this single metric Grafana/PromQL derives:

- **Throughput**: `sum(rate(http_request_duration_seconds_count[5m]))`
- **Error rate**: `sum(rate(...count{status=~"5.."}[5m])) / sum(rate(...count[5m]))`
- **Latency percentiles**: `histogram_quantile(0.99, sum by (le, path) (rate(http_request_duration_seconds_bucket[5m])))`
- **Status-code breakdown**: `sum by (status) (rate(...count[5m]))`

Recorded by a small echo middleware applied globally (skipping `/livez`,
`/readyz`, and `/metrics`), with `/metrics` served by `promhttp.Handler()`
for the scraper.

**Rules that keep this correct:**

- The `path` label MUST be the route template (`c.Path()` →
  `/agreements/:agreementId`), never the raw URL — raw URLs explode label
  cardinality and will eventually kill Prometheus. Same rule for any label:
  bounded value sets only (no user ids, no emails).
- **Status-on-error gotcha** (present in the reference implementation):
  when a handler returns an error, the middleware observes
  `c.Response().Status` *before* the error handler has written the real
  status, so errors can be recorded as 200. The metrics middleware MUST
  unwrap a returned `*echo.HTTPError` for its code (default 500 for other
  errors) instead of trusting the response status blindly.
- Buckets start fine-grained at the low end (1ms) because healthy API calls
  are fast; revisit them once real traffic is visible in Grafana — they are
  a starting point, not gospel.
- The default Prometheus registry already ships Go runtime metrics
  (goroutines, GC pause, heap) for free; don't disable them.
- Register a `pgxpool` stats collector (acquired/idle/max connections,
  acquire wait time) — connection-pool exhaustion is the classic invisible
  backend failure and this makes it a dashboard line.
- Custom business metrics (e.g. agreements sent, signatures recorded, emails
  failed) go on the
  same injected `*Metrics` struct as new collectors — services take it via
  fx like any dependency. Never `prometheus.MustRegister` from scattered
  `init()` functions.

### Saturation (the fourth golden signal)

RED covers latency, traffic, and errors; the fourth golden signal —
saturation, "how full is each finite resource" — is the leading indicator
the others lag. Day-1 coverage: the `pgxpool` stats collector (pool
utilization and acquire-wait-time warn *before* latency degrades), the
default Go runtime metrics (goroutine count and heap catch leaks), and
platform-level container CPU/memory. Any future queue or worker pool MUST
ship with its own depth/utilization gauge.

### Error tracking (Sentry)

Sentry is part of the observability stack: its SDK hooks the centralized
echo error handler and panic recovery — the same single choke point the
error-log contract already mandates — and reports **unexpected errors only**
(the same "not a known sentinel, not an `*echo.HTTPError`" branch that
prints stacks). Expected domain errors are never sent. Sentry provides what
logs can't: grouping identical failures, first-seen/regression tracking
against releases, affected-user counts, and alerting. Tag events with
`trace_id` and `request_id` so a Sentry issue links back to the exact log
and trace chain.

### Alerts (once a Prometheus/Grafana is pointed at it)

Start with exactly two: 5xx error rate > 1% over 5 minutes, and p99 latency
over your SLO for 5 minutes — plus Sentry's own new-issue/regression
notifications. Add more only when an incident proves a gap — alert sprawl
kills on-call signal.

## 9. Testing & CI Gates

Decisions resolved 2026-08-27 (testing pyramid and coverage revised later
the same day when the project-wide constitution set integration-first TDD).

- **Integration tests are the primary suite** and outrank unit tests:
  route → middleware → controller → service → store → **real Postgres**
  (testcontainers), with only external vendors (WorkOS, R2, the email
  provider, Sentry) mocked at the client-factory boundary. This exercises SQL,
  constraints, transactions, validation, authz, and error mapping the way
  production does.
  - Migrations run against the test database (goose `up`) as part of suite
    setup, which doubles as the migration smoke test.
  - Each test gets isolation via per-test transactions rolled back, or a
    per-suite schema — no shared mutable fixtures.
- **Unit tests with mocks** supplement where logic density earns them
  (state machines, price/fee calculations, parsers). Services depend on
  consumer-side interfaces so mocking stays possible; mocks are generated
  (`mockery`), never hand-rolled.
- **TDD is mandatory**: test written and failing before implementation
  (constitution Test-First principle).
- Tests are table-driven, run with `-race`, and use `t.Parallel()` where
  safe.
- **Coverage: 100% meaningful coverage is the stated aim; CI hard-gates at
  85% on new/changed code.** Excluded from the denominator: generated code
  (sqlc, oapi-codegen, mocks) and pure wiring (`main.go`, `boot/`).
  **Test hacking is banned** — tests must assert behavior (outputs, state
  changes, error values), not merely execute lines; assertion-free or
  tautological tests are a review-blocking defect.
- **CI gates on every PR** (all blocking):
  1. `go build ./...`
  2. `golangci-lint` — at minimum: `staticcheck`, `errcheck`, `gocyclo`
     (>10), `wrapcheck`, `depguard`/`forbidigo` (log bans, bare
     `http.Client` ban), `ineffassign`
  3. `go test -race ./...` + new-code coverage check
  4. `govulncheck` (dependency CVEs)
  5. `sqlc generate` diff check (generated code in sync with queries)
  6. goose migration up/down smoke test
  7. OpenAPI codegen diff check (generated types in sync with the spec,
     see §10)

## 10. Outbound Calls & API Contract

Decisions resolved 2026-08-27.

### Outbound resilience (day 1)

- Every outbound client comes from the fx client factory (§5) and MUST
  have an explicit timeout — zero-timeout clients are banned.
- Idempotent outbound calls retry with exponential backoff + jitter
  (`retry-go`); non-idempotent calls never auto-retry.
- **Circuit breakers per external dependency** (e.g. `sony/gobreaker`),
  wired into the client factory so callers get them for free. Breaker
  state changes are logged and exposed as a metric (a gauge per
  dependency) — an open breaker MUST be visible on the dashboard, not
  discovered from user reports.

### API contract: spec-first OpenAPI

- **The OpenAPI spec is written before the implementation** and lives at
  `contracts/openapi.yaml` in the monorepo root — owned by neither app.
  For each feature, the Spec Kit flow extends naturally: feature spec →
  contract change → implement to the contract, all in one PR.
- `oapi-codegen` generates Go types from the spec into the `dto` package
  (`make oapi-generate`, config in `backend/oapi-codegen.yaml`, output
  `internal/models/dto/api.gen.go`). Generation is **types-only**:
  oapi-codegen has no server generator for echo v5, so routes, handlers,
  and the `ValidateRequest` middleware pattern (§3) stay hand-written.
  `x-oapi-codegen-extra-tags` annotations in the contract carry the
  `validate` tags onto generated request types, and `x-go-type-*` /
  `x-omitempty` annotations pin the generated shapes to the wire format
  the handlers emit.
- Generated types replace hand-written DTOs wherever the spec can express
  the shape. Exactly two DTO classes stay hand-written, living beside the
  converters in `models/dto`:
  1. **Multi-source binding wrappers** — structs that carry `param:` or
     `query:` tags (path-id embeds like `WithAgreementIDParamDto`, list
     endpoints with repeatable comma-split query lists). The generator emits
     no echo binding tags. Wrappers embed the generated body type where
     one applies (e.g. `CreateAgreementVersionRequest` embeds
     `AgreementWrite`).
  2. **Presence-sensitive request DTOs** — steps where `false` or `0` is
     a legal answer yet an absent field must still fail validation
     (a signer-consent step where `false` is a legal answer, for
     example). A generated value type cannot distinguish
     an absent field from its zero value, so these keep pointer fields
     with `validate:"required"`.
- Model↔DTO conversion stays in hand-written `XxxFrom` converters in
  `models/dto`, returning the generated types.
- CI regenerates and diffs (gate 7 in §9), so the spec cannot drift from
  the generated types: the spec is the source of truth, the diff proves
  it. The web workflow runs its own client diff on the same
  `contracts/**` trigger, so a contract change that regenerates either
  side lands in the same PR.
- **Timestamps are Unix milliseconds (int64) everywhere** in API requests
  and responses — no RFC3339 strings, no mixing. Postgres stores
  `timestamptz`; conversion to UnixMilli happens once at the store/DTO
  boundary.
- **Idempotency keys**: the pattern is fixed here — client sends an
  `Idempotency-Key` header; the server dedupes in Redis (fallback: a PG
  table) with ~24h retention, replaying the original response on repeat.
  MUST be implemented on any endpoint that sends an agreement, records a
  signature, or sends email, from the first such endpoint; plain CRUD
  endpoints skip it.

### Auth: WorkOS through a token-mediating backend

- WorkOS is the identity provider, reached only through its headless
  API. The hosted AuthKit page is not used. The API is the confidential
  OAuth client, following the token-mediating backend variant of the BFF
  pattern. Ten endpoints in `auth-routes.go` / `auth-controller.go` /
  `auth-service.go` own the session:
  - `GET /v1/auth/login` takes a required `provider` query (`google`),
    builds the authorization URL with a signed `state` (carrying
    `returnTo`), and redirects straight to the provider — no chooser
    page.
  - `GET /v1/auth/callback` exchanges the code with the WorkOS Go SDK
    using `WORKOS_API_KEY` and `WORKOS_CLIENT_ID`, sets the refresh cookie,
    and redirects to `WEB_APP_URL` plus the validated `returnTo`.
  - `POST /v1/auth/refresh` reads the cookie, calls WorkOS's refresh
    endpoint server-side, rotates the cookie, and returns the new access
    token as JSON.
  - `POST /v1/auth/logout` ends the WorkOS session and expires the cookie.
  - `POST /v1/auth/sign-in` authenticates with password, seals the
    cookie and returns an access token, or answers `202` with a pending
    verification token when the email is unverified.
  - `POST /v1/auth/sign-up` creates the user and always answers `202`
    with a pending verification token, including on a duplicate email
    (see the error-code rule below).
  - `POST /v1/auth/verify-email` exchanges the pending token and a code
    for the same session-sealing result as sign-in.
  - `POST /v1/auth/resend-verification` re-sends the code by email;
    always `202`.
  - `POST /v1/auth/forgot-password` issues a reset token and emails the
    reset link through `internal/mail`; always `202`.
  - `POST /v1/auth/reset-password` confirms the reset, then signs the
    user in and seals the cookie.
- The refresh cookie is encrypted with `SESSION_COOKIE_KEY` (AEAD, key
  rotation supported by key id), `HttpOnly; Secure; SameSite=Strict;
  Path=/v1/auth`, scoped to the API host, with `Max-Age` equal to the
  WorkOS refresh token lifetime. Its value never appears in logs or error
  messages.
- All ten endpoints sit in the same `/v1/auth` group: they verify
  `Origin` against `CORS_ALLOWED_ORIGINS`, allow credentialed CORS for
  those origins only, and carry a stricter rate limit than the global
  default. Cloudflare adds a second, IP-based layer in front of the
  credential endpoints (see `docs/deployment.md`).
- Every other endpoint validates the `Authorization: Bearer` JWT against
  the WorkOS JWKS endpoint — provider-agnostic JWKS verification, no
  database hit per request. Cookies are ignored outside `/v1/auth`.
- **Error-code rule**: the credential endpoints answer from a fixed set
  of codes (`invalid_credentials`, `invalid_code`, `invalid_reset_token`,
  `weak_password`) and never pass a provider error message through,
  except `weak_password`, which carries WorkOS's password-strength
  message in `details.message` because that message is the reason the
  user needs. No response distinguishes an unknown email from a wrong
  password or a taken email from a free one — sign-in, sign-up, and
  forgot-password give the identical answer either way.
- User records are provisioned/synced via WorkOS webhooks (verified
  signatures, idempotent handlers).
- WorkOS Organizations maps onto sender organizations when multi-member
  teams arrive — don't build a parallel org model without checking what
  WorkOS already provides.
- `internal/mail` is one `Mailer` interface (`Send(ctx, Message) error`)
  with two implementations selected by `boot.NewMailer`: `ResendMailer`,
  a plain `net/http` client posting to the Resend API when
  `RESEND_API_KEY` is set, and `LogMailer`, which writes the message at
  info level for local development. `AuthService` uses it for the
  password-reset and "account exists" emails — WorkOS sends the
  verification-code email itself.

---

## 11. What agent-backend is missing (don't copy these)

Gaps found in the audit — each becomes a rule or a conscious fix in
SoW:

1. **Echo v4, not v5.** SoW pins `echo/v5` from day one.
2. **golang-migrate, not goose.** The numbered up/down discipline and the
   dev-only guard on destructive targets carry over; the tool changes to
   goose.
3. **No sqlc.** All SQL is handwritten strings with manual `Scan` — verbose
   and unchecked. SoW: sqlc-generated, typed queries only; the
   `DBTX`/`WithTx` pattern is retained because sqlc emits the same shape.
4. **Tests are nearly absent**: 4 test files across 113 Go files, none for
   stores, controllers, or routes. This is the single biggest gap. The
   constitution's Test-First principle directly targets it.
5. **No linter config** — no `.golangci.yml`, no complexity gate, no CI lint
   step (CI only builds and deploys). SoW: golangci-lint (incl.
   gocyclo) required in CI.
6. **Mixed logging backends**: slog, gommon `log`, and stdlib `log` coexist
   (`log.Fatalf` inside `NewDBConn` even bypasses the fx error path and
   kills the process before fx can clean up). SoW: slog only;
   constructors return errors, never `Fatal`.
7. **Printf-style log helpers** (`Errorf` etc.) flatten structured logging
   into message strings. SoW: key-value only.
8. **Inconsistent controller error mapping**: some handlers return mapped
   JSON errors, others return the raw `err` and leak echo's default 500.
   SoW: a single centralized echo error handler that maps sentinel
   errors → status codes; handlers just `return err`.
9. **Inconsistent sentinel naming** (`CARD_NOT_FOUND` vs
   `COMPOSIO_API_5XX` with message `"composio_api_5xx"`). SoW:
   standard Go `ErrXxx` naming.
10. **Hygiene**: test routes registered in the production fx app
    (`routes.AddTestRoutes`), a hardcoded user UUID in a test-notification
    endpoint, commented-out code left in config, `app.log` and security-scan
    reports committed at the repo root. SoW: test-only routes gated
    by stage, no dead code, generated/log artifacts gitignored.
11. **Rate limiting commented out** and left as a TODO. SoW ships
    it day 1 (see §3): global per-IP default + stricter per-route limits,
    Redis-backed as soon as there is more than one replica.
12. **No README / architecture doc** — the layout has to be reverse-
    engineered (this document is that reverse-engineering). SoW
    keeps an architecture section in the README from the first commit.
