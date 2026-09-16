# SoW owner setup and GitHub integration guide

This guide takes the repository from the current local clone to a working local
environment, a reviewed GitHub pull request, and a production deployment.

## How to use this guide if you are not technical

The product is made of three programs that cooperate:

1. The **marketing site** is the public website people see at
   `contractsfordata.com`. Astro is the software used to build it.
2. The **API**, or backend, is the private engine that validates answers,
   builds the Word file, and asks the email provider to send it. It is written
   in Go and is expected to run on Railway.
3. The **authenticated web app** is the separate signed-in product containing
   the existing eight-step agreement configurator. It must remain unchanged by
   the public-configurator project described below.

When this guide says "environment variable," it means a named setting supplied
to a program when it starts. Passwords and API keys are kept in environment
variables so they do not appear in source code or GitHub history. When it says
"endpoint," it means a particular web address on the API that performs one
job, such as generating a Word document.

Work through the guide in order. Each external product has a different job;
none of the accounts should be given broader permissions than that job needs.

## Plain-English product map

| Product | What it does | Why this project needs it | What the owner supplies |
|---|---|---|---|
| GitHub | Stores versioned source code and runs automated checks | Provides a safe review history and prevents untested code from going live | Repository access or a personal fork |
| Docker Desktop | Runs isolated software containers on the development Mac | Starts a local PostgreSQL database and allows backend integration tests to reproduce production-like behavior | Installation and permission to run locally |
| Astro | Builds the public marketing website | Provides the pages and public no-login questionnaire visitors interact with | Nothing beyond Node.js; it is already a project dependency |
| Go | Compiles and runs the backend API | Keeps document generation, validation, email credentials, and provider calls off the public browser | Go 1.25.14 locally; Railway supplies the production runtime |
| Supabase | Hosts PostgreSQL | Gives the complete signed-in application a durable database managed outside the app server | A project and protected database connection string |
| WorkOS | Handles accounts, sign-in, password recovery, and user events | Avoids building and securing an identity system from scratch | API key, client ID, redirect URLs, and webhook secret |
| Resend | Delivers transactional email | Sends the generated DOCX as a real attachment and confirms whether the provider accepted it | Verified sending domain, API key, and sender address |
| Railway | Runs the Go API continuously on the internet | Gives both frontends a secure public API address and manages backend secrets | Connected GitHub repository, service variables, and custom domain |
| Cloudflare | Hosts the two static frontends and manages DNS | Publishes the public site and authenticated app close to users and connects domain names to services | Account, domain/DNS control, and a deployment connection |
| Sentry | Records production application errors | The current backend refuses to start in production without a Sentry DSN and it helps diagnose failures without exposing them to visitors | A Sentry project and DSN |

## Required next phase: public no-login configurator

### Context and boundary

- Repository: `Pixels-Two/SoW`.
- Working branch: `feature/005-gated-template-delivery`.
- Public frontend: the Astro marketing site at `contractsfordata.com`.
- Backend: the Go API, locally at `http://localhost:8080`.
- Existing signed-in configurator: `/agreements/new` in the separate web app.

The existing eight-step signed-in configurator is explicitly out of scope and
must not be edited, reused as a hidden dependency, or have its API behavior
changed. The public configurator must have its own components, request schema,
endpoint, tests, and document-selection model.

### Goal in plain language

Replace the marketing site's generic-template download with a short public
questionnaire. A visitor answers four legal-structure questions without
creating an account. The browser sends those answers to a dedicated public API
endpoint. The API validates them, selects exactly one clause for each answer,
and returns a genuine customized Microsoft Word `.docx` file.

This separation is necessary because public visitors have no login session,
while the existing eight-step flow contains richer account and agreement data.
Keeping the paths separate reduces the chance that a public request can read or
change authenticated agreement information.

### Public marketing form

The form must include optional client-information controls:

- **Delivery preference:** Download or Email.
- **Work email:** shown and validated only when Email is selected.
- **Type:** Supplier, Buyer, or Other.
- **Company name:** free text.

These values provide delivery context only. Company name may be inserted only
where the public template explicitly permits it. An email address must never be
treated as a legal party name or inserted into the agreement.

**TODO - owner decision:** confirm whether all client-information fields are
truly skippable or whether any must be completed before download. Until this is
decided, implementation should treat them as optional and must not invent
required-field rules.

The four fixed multiple-choice questions are:

1. **Subcontracting**
   - A: Never allowed.
   - B: Only with Buyer's written consent. **Default.**
   - C: Allowed with notice; Buyer can object.
   - *Why this is asked:* it determines whether the supplier can pass work to
     another company and what control the buyer retains. The interface should
     show this explanation in short italic text or an accessible
     click-to-reveal help control.
2. **Jurisdiction Rules**
   - A: Added to the Agreement per country.
   - B: Set in each SOW's country appendix. **Default.**
   - C: Separate Jurisdiction Rider per country.
   - *Why this is asked:* data and transfer requirements vary by country, so
     the answer controls where country-specific obligations are documented.
3. **Liability Cap for data-compliance and IP claims**
   - A: Greater of `$[TBD]` or `[TBD]x` fees. **Default.**
   - B: `[TBD]x` fees only.
   - C: No special cap.
   - *Why this is asked:* it determines whether higher-risk data-compliance and
     intellectual-property claims use a separate, higher financial cap than
     ordinary claims.
   - **TODO - owner/legal decision:** supply the exact dollar floor and fee
     multiplier for options A and B. Keep visible placeholders until approved;
     do not guess amounts. Options A and B must not produce a supposedly final
     agreement until those values are supplied; the API should reject delivery
     rather than place unresolved `[TBD]` tokens in the DOCX.
4. **Exclusivity**
   - A: Fully exclusive, global.
   - B: Exclusive, limited to field: `[____]`.
   - C: Non-exclusive.
   - *Why this is asked:* it defines whether the supplier may perform similar
     work for other buyers. When B is selected, show and require the field-scope
     text; hide it or ignore it for A and C.

Each question must use real radio controls, expose its description to screen
readers, and work with keyboard, touch, and pointer input. Defaults may be
preselected only where marked above. A summary should let the visitor review
answers before delivery.

### Disclaimer and delivery behavior

The public form must retain the legal acknowledgment before delivery. The
checkbox text is:

> I acknowledge this is not legal advice, and that I am responsible for my own
> legal review before use.

The Download button must remain disabled until the visitor actively checks the
acknowledgment. Client-side gating is for usability; the public API must also
reject any request that does not include explicit acknowledgment.

Download is the required first delivery method. Email may be exposed only after
a real end-to-end test confirms that Resend accepts the message and the same
generated DOCX arrives as an attachment. If Resend is unconfigured or rejects
the send, hide or disable the email choice with an honest explanation; never
show a simulated success.

### Dedicated public Go API

Add a new unauthenticated endpoint dedicated to this short public flow, for
example:

```text
POST /v1/public/sow-configurator/download
```

If verified email delivery is exposed, use a sibling endpoint such as:

```text
POST /v1/public/sow-configurator/email
```

Define both in `contracts/openapi.yaml` before implementation. Do not alter the
endpoint or payload used by `/agreements/new`.

The public request must contain only:

- Explicit acknowledgment.
- One A/B/C answer for each of the four questions.
- Optional delivery preference, party type, company name, and email.
- The exclusivity field-scope text only when Exclusivity B is selected.

The API must allow only the documented enum values, reject unknown fields,
apply body-size and rate limits, validate email on the server, and prevent
duplicate sends. Company name and exclusivity scope must be trimmed, length
limited, stripped of invalid XML control characters, and safely XML-escaped by
the DOCX writer before interpolation. Sanitization must not silently turn
arbitrary text into legal clauses.

### Word template changes

Update `Robot_Data_Content_Development_Agreement_TEMPLATE` through the shared
DOCX generation service. Both direct download and any enabled email path must
use the same bytes.

Add these fixed sections:

- **Security:** reasonable safeguards and breach notification within 24 hours.
  This fixed period should be presented for legal approval before launch.
- **Representations and Warranties:** the supplier has necessary rights and
  consents; work does not infringe third-party rights; and collection excludes
  minors, nonconsenting individuals, and sensitive data prohibited by the
  agreement.
- **Two-tier Limitation of Liability:** one general cap and a distinct higher
  cap for data-compliance and intellectual-property claims.

Add exactly one conditional clause for each A/B/C answer in Subcontracting,
Jurisdiction Rules, Liability Cap, and Exclusivity. The generator must never
leave unselected alternatives, drafting instructions, `[TBD]` tokens presented
as final values, `undefined`, `null`, or `NaN` in the output.

**TODO - owner/legal decision:** decide whether jurisdiction-specific overlays
remain fixed template content or become a fifth public question. Do not add a
fifth question until that decision is explicit.

### Non-goals for this phase

- Do not change the authenticated eight-step configurator at
  `/agreements/new`.
- Do not add or change GitHub Actions deployment steps. The existing Wrangler
  commands remain dry runs unless separately instructed.
- Do not link or migrate `contractsfordata.com`; domain linking is a separate
  owner-approved deployment task.

### Required tests for this phase

- All marketing template CTAs open the new public form rather than the generic
  download.
- Defaults are B for questions 1-3; question 4 requires an explicit choice
  unless the owner specifies a default.
- The acknowledgment starts unchecked and blocks browser and API delivery.
- All free-text sanitization, length limits, conditional requirements, invalid
  enums, unknown fields, and malformed emails are tested.
- The public endpoints remain unauthenticated but cannot call or mutate the
  signed-in configurator path.
- Each of the 81 answer combinations (`3 x 3 x 3 x 3`) produces a valid OOXML
  ZIP with exactly one selected clause per category.
- The package contains `[Content_Types].xml` and `word/document.xml` and opens
  in Word, Apple Pages, and LibreOffice where practical.
- Direct and email delivery for the same answers produce byte-identical DOCX
  attachments where request metadata does not intentionally vary.
- Provider rejection produces an error, never success.
- No PDF or generic-template bypass remains on the marketing site.
- Existing `/agreements/new` tests and behavior remain unchanged.

### Estimated Codex compute and delivery effort

Codex usage is not a fixed unit that can be priced exactly per feature. It
depends on repository size, test failures, review iterations, document-render
checks, and whether product TODOs are resolved before coding. ChatGPT Work and
Codex also share account usage under the current OpenAI plan model.

For the complete public-configurator phase above, a reasonable planning range
is **12-24 percentage points of a weekly Plus Codex allowance**, normally over
one to three focused sessions. A rough allocation is:

| Work | Estimated weekly allowance |
|---|---:|
| Contract/API design and test scaffolding | 2-4 points |
| Accessible Astro form and interaction tests | 3-6 points |
| Go validation, clause branching, and sanitization | 3-6 points |
| DOCX template work and 81-case matrix | 2-5 points |
| Browser, email, build, and visual QA | 2-3 points |

This is a planning estimate, not a guaranteed charge or limit. At the time this
manual was amended, the account showed approximately **8% remaining in the
active five-hour window and 49% remaining in the weekly window**. The weekly
balance is probably sufficient, but the implementation should start after the
five-hour window resets or use one of the account's available reset credits if
the owner explicitly chooses to redeem one. Resolve the three TODO decisions
before coding to reduce rework and usage.

## 1. Understand the current state

- The local clone points at `https://github.com/Pixels-Two/SoW.git` as `origin`.
- The current branch is `feature/005-gated-template-delivery`.
- The gated-delivery work is present locally but is not committed or pushed.
- The landing-page preview is only a frontend. Its download and email buttons
  call the Go API at port 8080, so both fail while that API is stopped.
- The eight-step configurator is in the authenticated web application at
  `/agreements/new`. It is not embedded in the public marketing site.
- The marketing CTA currently requests a generic editable template. If the
  intended public journey is marketing page -> public survey -> customized
  document, that is an additional code change, not an account setting.
- The GitHub Actions workflows currently validate code. They do not deploy the
  Cloudflare sites; the web and marketing workflows end with `wrangler deploy
  --dry-run`.

Decide the public product journey before launch:

1. Keep the marketing download as a generic Word template and keep the
   customized configurator behind sign-in; or
2. Send the marketing CTA to the signed-in configurator; or
3. Build a new public/no-account configurator.

Option 2 is the smallest change if every delivered document should be
customized. Option 3 requires additional product and security work.

## 2. Accounts and tools to obtain

### Local development tools

- Git.
- Node.js 22.23.2 and npm 10.x.
- Go 1.25.14.
- Docker Desktop. The local Postgres container and backend integration tests
  require a running Docker engine.
- Optional: GitHub CLI (`gh`) for creating pull requests from Terminal.

### Hosted-service accounts

- GitHub: source control and CI.
- A domain and DNS access.
- Supabase: hosted PostgreSQL only.
- WorkOS: authentication.
- Resend: transactional email and Word attachments.
- Railway: Go backend hosting.
- Cloudflare: DNS and the two static frontend Workers.
- Sentry: the current backend configuration requires `SENTRY_DSN` when
  `STAGE=prod`.

Cloudflare R2 is mentioned in the broader deployment plan, but it is not
required for the new download/email flow. Those documents are generated in
memory and are not persisted.

## 3. Protect secrets

Never commit `.env` files or paste production secrets into source files,
issues, pull requests, or chat. The repository already ignores:

- `backend/.env`
- `web/.env`
- `marketing/.env`

Use local `.env` files only for development. Put production backend secrets in
Railway service variables. If GitHub Actions is later used for Cloudflare
deployment, put the Cloudflare token in GitHub Actions secrets.

## 4. Set up the project locally

From the repository root:

```bash
cp backend/.env.example backend/.env
cp web/.env.example web/.env
cp marketing/.env.example marketing/.env
```

### Backend environment

Edit `backend/.env` and set at least:

```dotenv
STAGE=dev
PORT=8080
DATABASE_URL='postgres://sow:sow@localhost:5432/sow?sslmode=disable'
CORS_ALLOWED_ORIGINS=http://localhost:5173,http://127.0.0.1:5173,http://localhost:4321,http://127.0.0.1:4321
WORKOS_API_KEY=replace-me
WORKOS_CLIENT_ID=replace-me
WORKOS_WEBHOOK_SECRET=replace-me
WORKOS_REDIRECT_URI=http://localhost:8080/v1/auth/callback
WEB_APP_URL=http://localhost:5173
SESSION_COOKIE_KEY=k1:replace-with-32-byte-base64-value
```

Generate the session-key value with:

```bash
openssl rand -base64 32
```

Append the output to `k1:` in `SESSION_COOKIE_KEY`.

`RESEND_API_KEY` and `EMAIL_FROM` may remain empty during local development.
Direct Word download will work, but email delivery will honestly report that
it is unavailable.

If marketing runs on port 4322, add both
`http://localhost:4322` and `http://127.0.0.1:4322` to
`CORS_ALLOWED_ORIGINS`, and set `PUBLIC_SITE_URL` to the port actually used.

### Frontend environments

Use these values in `web/.env`:

```dotenv
VITE_API_URL=http://localhost:8080
```

Use these values in `marketing/.env` when using the standard Astro port:

```dotenv
PUBLIC_SITE_URL=http://localhost:4321
PUBLIC_APP_ORIGIN=http://localhost:5173
PUBLIC_TEMPLATE_DELIVERY_URL=http://localhost:8080/v1/template-deliveries
```

### Install and start

Start Docker Desktop first. Then use three Terminal windows.

Terminal 1, backend:

```bash
cd backend
make db-up
make migrate-up
make run
```

Terminal 2, authenticated configurator:

```bash
cd web
npm ci
npm run dev
```

Terminal 3, marketing site:

```bash
cd marketing
npm ci
npm run dev
```

Verify the backend:

```bash
curl http://127.0.0.1:8080/livez
curl http://127.0.0.1:8080/readyz
```

Both should return HTTP 200 before testing delivery.

### Test direct Word download locally

```bash
curl --fail-with-body \
  --request POST \
  --header 'Origin: http://127.0.0.1:4321' \
  --header 'Content-Type: application/json' \
  --data '{"acknowledged":true}' \
  --output /tmp/sow-template.docx \
  http://127.0.0.1:8080/v1/template-deliveries/download
```

Check that it is a real OOXML file:

```bash
unzip -l /tmp/sow-template.docx
```

The listing must include `[Content_Types].xml` and `word/document.xml`.

## 5. Set up Resend email

1. Create a Resend account.
2. In Resend, add a domain or preferably a sending subdomain such as
   `mail.example.com`.
3. Copy the SPF and DKIM records Resend displays into the authoritative DNS
   provider, normally Cloudflare for this stack.
4. Wait until Resend reports the domain as **Verified**.
5. Create an API key with sending access and copy it once.
6. Choose a sender on the verified domain, for example
   `documents@mail.example.com`. Use a plain address because the backend's
   configuration validator expects an email address.
7. Put the values in local `backend/.env` for a private test:

   ```dotenv
   RESEND_API_KEY=re_replace_me
   EMAIL_FROM=documents@mail.example.com
   ```

8. Restart the backend after changing variables.
9. Test through the website with an inbox you control.
10. Put the production values in Railway, not GitHub and not the frontend.

Before domain verification, Resend's test sender is restricted to the account
owner's email. A verified domain is required to send to customers.

No local mail-server software is needed. The backend sends the generated DOCX
to Resend over HTTPS.

## 6. Set up Supabase PostgreSQL

1. Create a Supabase project and save the database password in a password
   manager.
2. Open the project's **Connect** dialog.
3. Copy the Session pooler connection string on port 5432 for the Railway
   backend. This is the safe choice when the host needs IPv4 and it supports
   the prepared statements used by pgx.
4. Replace the password placeholder and percent-encode reserved characters.
5. Save that complete string as Railway's `DATABASE_URL`.
6. Copy the direct database connection separately for migrations from an
   IPv6-capable trusted machine, if desired.
7. Run the migrations before serving traffic:

   ```bash
   go tool goose -dir internal/migrations postgres "$DATABASE_URL" up
   ```

Do not expose `DATABASE_URL` to either frontend.

## 7. Set up WorkOS

1. Create a WorkOS environment for the deployment stage.
2. Enable email/password authentication and email verification.
3. Enable Google OAuth only if it should appear as a sign-in option.
4. Register this redirect URI:

   ```text
   https://api.example.com/v1/auth/callback
   ```

5. Set the password-reset URL to:

   ```text
   https://app.example.com/reset-password
   ```

6. Create a webhook endpoint pointing to:

   ```text
   https://api.example.com/v1/webhooks/workos
   ```

7. Subscribe it to `user.created`, `user.updated`, and `user.deleted`.
8. Copy the environment API key, client ID, and webhook signing secret into
   Railway as `WORKOS_API_KEY`, `WORKOS_CLIENT_ID`, and
   `WORKOS_WEBHOOK_SECRET`.
9. Set `WORKOS_REDIRECT_URI`, `WEB_APP_URL`, and `MAGIC_LINK_BASE_URL` only if
   that latter variable is added to the application in a future feature; the
   current backend configuration does not read `MAGIC_LINK_BASE_URL`.

WorkOS production redirect URIs must use HTTPS.

## 8. Deploy the Go backend on Railway

1. Create a Railway project.
2. Add a service and connect the GitHub repository containing the merged code.
3. Set the service root directory to `/backend`.
4. Configure the build command:

   ```text
   go build -o app ./cmd/app
   ```

5. Configure the start command:

   ```text
   ./app
   ```

6. Set the pre-deploy migration command:

   ```text
   go tool goose -dir internal/migrations postgres "$DATABASE_URL" up
   ```

7. Configure the health-check path as `/readyz`.
8. Add a public Railway domain, then attach the intended custom API domain.
9. Add these required production variables in Railway:

   ```text
   STAGE=prod
   DATABASE_URL
   CORS_ALLOWED_ORIGINS
   WORKOS_API_KEY
   WORKOS_CLIENT_ID
   WORKOS_WEBHOOK_SECRET
   WORKOS_REDIRECT_URI
   WEB_APP_URL
   SESSION_COOKIE_KEY
   RESEND_API_KEY
   EMAIL_FROM
   SENTRY_DSN
   ```

10. Keep `PORT` managed by Railway. The remaining pool, timeout, and rate-limit
    variables have validated defaults and can be tuned later.
11. Set `CORS_ALLOWED_ORIGINS` to the exact production origins, with no paths:

    ```text
    https://app.example.com,https://example.com
    ```

12. Deploy and confirm both `https://api.example.com/livez` and
    `https://api.example.com/readyz` return 200.

Railway variables are injected at build and runtime. Seal sensitive variables
where practical.

## 9. Deploy the web and marketing sites on Cloudflare

There are two separate static applications:

| Application | Directory | Worker name | Required build variables |
|---|---|---|---|
| Authenticated app | `/web` | `sow-web` | `VITE_API_URL=https://api.example.com` |
| Marketing site | `/marketing` | `sow-marketing` | `PUBLIC_SITE_URL=https://example.com`, `PUBLIC_APP_ORIGIN=https://app.example.com`, `PUBLIC_TEMPLATE_DELIVERY_URL=https://api.example.com/v1/template-deliveries` |

Choose one deployment method.

### Option A: Cloudflare Workers Builds

1. Connect the GitHub repository in Cloudflare Workers Builds.
2. Create one project for `/web` and one for `/marketing`.
3. Use `npm ci && npm run build` as the build command for each.
4. Configure each project with its build variables from the table above.
5. Deploy from `main` after pull requests merge.
6. Attach `app.example.com` to `sow-web` and the apex domain to
   `sow-marketing`.

### Option B: GitHub Actions

1. Create a least-privilege Cloudflare API token with Workers edit access,
   scoped only to the required account.
2. In GitHub, open **Settings -> Secrets and variables -> Actions**.
3. Add repository or production-environment secrets:

   ```text
   CLOUDFLARE_API_TOKEN
   CLOUDFLARE_ACCOUNT_ID
   ```

4. Store the four public build URLs as GitHub Actions variables, not secrets.
5. Add deployment jobs using `cloudflare/wrangler-action` or authenticated
   `wrangler deploy` commands after the existing checks pass on `main`.

The repository does not currently contain step 5. Merely adding Cloudflare
secrets will not deploy anything until deployment jobs are added, or Workers
Builds is connected.

## 10. Put the current local work on GitHub

The correct path depends on your access to `Pixels-Two/SoW`.

### Path A: you can push to the original repository

Keep the current remote and use the existing feature branch:

```bash
git remote -v
git branch --show-current
git status --short
git diff --check
git add .
git diff --cached --stat
git status --short
git commit -m "Add gated Word template delivery"
git push -u origin feature/005-gated-template-delivery
```

Then open GitHub and create a pull request:

- Base repository: `Pixels-Two/SoW`
- Base branch: `main`
- Compare branch: `feature/005-gated-template-delivery`

### Path B: you cannot push to the original repository

1. On GitHub, fork `Pixels-Two/SoW` into your own account.
2. In the existing clone, preserve the original repository as `upstream` and
   make your fork the writable `origin`:

   ```bash
   git remote rename origin upstream
   git remote add origin https://github.com/YOUR-USERNAME/SoW.git
   git remote -v
   ```

3. Review and commit the current local changes:

   ```bash
   git branch --show-current
   git status --short
   git diff --check
   git add .
   git diff --cached --stat
   git status --short
   git commit -m "Add gated Word template delivery"
   git push -u origin feature/005-gated-template-delivery
   ```

4. Create a pull request from
   `YOUR-USERNAME/SoW:feature/005-gated-template-delivery` into
   `Pixels-Two/SoW:main`.

If GitHub CLI is installed and you can push directly, the PR can instead be
created from Terminal:

```bash
gh pr create \
  --repo Pixels-Two/SoW \
  --base main \
  --title "Add gated Word template delivery" \
  --body "Adds disclaimer acknowledgment, direct DOCX delivery, and Resend DOCX attachment delivery."
```

For a fork, identify the fork branch explicitly:

```bash
gh pr create \
  --repo Pixels-Two/SoW \
  --base main \
  --head YOUR-USERNAME:feature/005-gated-template-delivery \
  --title "Add gated Word template delivery" \
  --body "Adds disclaimer acknowledgment, direct DOCX delivery, and Resend DOCX attachment delivery."
```

### Review the pull request

The changed paths automatically trigger the backend, web, and marketing
workflows. Do not merge until all required checks are green. In particular,
confirm:

- Backend build, lint, race/coverage, generated-code diff, vulnerability scan,
  and migration smoke test pass.
- Web typecheck, lint, formatting, coverage, build, and Wrangler dry-run pass.
- Marketing typecheck, formatting, tests, build, and Wrangler dry-run pass.
- No `.env` file or credential appears in the changed-files list.
- The deleted PDF generator and route are intentional.

After approval, merge the pull request into `main`. Railway and Cloudflare will
only deploy automatically if their GitHub integrations or real deployment jobs
have been configured as described above.

## 11. Production smoke test

Run this checklist after deployment:

1. Open `/livez` and `/readyz` on the API and confirm HTTP 200.
2. Open the marketing site and activate every template CTA.
3. Confirm each CTA opens **Before You Download**.
4. Confirm the checkbox stays unavailable until the disclaimer bottom is
   reached, then requires an active check before Continue works.
5. Download the Word document and open it in Microsoft Word or Apple Pages.
6. Confirm the file extension is `.docx`, not PDF.
7. Sign in to the app, complete `/agreements/new`, and download a customized
   document.
8. Confirm the chosen clauses appear once and unselected alternatives do not.
9. Email the same configuration to a controlled test inbox.
10. Confirm the attachment name, MIME type, and document contents match the
    direct download.
11. Attempt delivery without acknowledgment and confirm the API rejects it.
12. Confirm recipient and contract data never appear in the browser URL.

## Official setup references

- [GitHub: work with forks](https://docs.github.com/en/pull-requests/how-tos/work-with-forks)
- [GitHub: Actions secrets](https://docs.github.com/en/actions/reference/security/secrets)
- [Railway: deploy a monorepo](https://docs.railway.com/deployments/monorepo)
- [Railway: service variables](https://docs.railway.com/variables)
- [Railway: pre-deploy commands](https://docs.railway.com/deployments/pre-deploy-command)
- [Supabase: connect to Postgres](https://supabase.com/docs/guides/database/connecting-to-postgres)
- [WorkOS: redirect URI requirements](https://workos.com/docs/reference/sso/get-authorization-url)
- [Resend: domain verification](https://resend.com/docs/dashboard/domains/introduction)
- [Resend: sender and API-key setup](https://resend.com/docs/knowledge-base/how-do-I-create-an-email-address-or-sender-in-resend)
- [Cloudflare: GitHub Actions deployment](https://developers.cloudflare.com/workers/ci-cd/external-cicd/github-actions/)
- [Cloudflare: Workers CI/CD options](https://developers.cloudflare.com/workers/ci-cd/)
- [OpenAI: Codex and ChatGPT Work pricing and shared usage](https://learn.chatgpt.com/docs/pricing)
