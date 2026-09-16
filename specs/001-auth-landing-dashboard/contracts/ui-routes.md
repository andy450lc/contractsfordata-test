# Contract: SPA Routes and Landing Page

**Date**: 2026-09-03 · **Feature**: 001-auth-landing-dashboard

## Landing page (`marketing/`, apex domain)

| Element | Behavior |
|---|---|
| Heading | Product name ("SoW" until renamed). |
| Paragraph | One paragraph: create a Statement of Work from a template, send it, collect signatures. |
| "Sign in" | Link to `<app origin>/` |
| "Get started" | Link to `<app origin>/sign-up` |

Static. No scripts, forms, cookies, or network calls beyond the page's
own assets. Renders without horizontal scroll at 375 px.

## SPA routes (`web/`, app subdomain)

| Path | Guard | Element | Behavior |
|---|---|---|---|
| `/` | none | SignInPage | Product name, "Sign in" button linking to `API/v1/auth/login?screen=sign-in&returnTo=<redirect>`, link to `/sign-up`. Shows a notice when `?error=` is present. An already-authenticated visitor is sent to `safeReturnTo(redirect)`. |
| `/sign-up` | none | SignUpPage | Same shell with "Create account" linking to `API/v1/auth/login?screen=sign-up`, link to `/`. |
| `/callback` | none | CallbackPage | Runs one refresh. Success → navigate to `safeReturnTo(returnTo)`, replace. Failure → `/?error=flow_incomplete`, replace. Shows the loader meanwhile. |
| `/dashboard` | ProtectedLayout | DashboardPage | Greets by name (email when name is empty), shows the email, a "You're logged in" line, and a sign-out button. |
| `*` | none | NotFoundPage | "Page not found" with a link to `/`. |

### ProtectedLayout

- `status === 'unknown'` → full-page loader (bootstrap refresh in flight).
- `status === 'anonymous'` → `<Navigate to="/?redirect=<path+search>" replace />`.
- `status === 'authenticated'` → `<Outlet />`.

Exactly one guard component exists.

### Session bootstrap

On app load `bootstrap()` calls `POST /v1/auth/refresh` once with
credentials. 200 → `authenticated` with the token and expiry; 401 or 403
→ `anonymous`; network failure → `anonymous` with a retry on the next
protected navigation.

### Error notices on `/`

| `?error=` | Notice |
|---|---|
| `flow_incomplete` | "Sign-in didn't complete. You can try again whenever you're ready." |
| `invalid_state` | "That sign-in link expired. Please start again." |
| `provider_unavailable` | "Sign-in is temporarily unavailable. Please try again in a moment." |
| `session_expired` | "You were signed out. Please sign in again." |
| anything else | The `flow_incomplete` notice. |

### Sign-out

`POST /v1/auth/logout` with credentials → clear the in-memory session →
write the cross-tab sentinel → navigate to `/`. Failure still clears the
in-memory session and navigates; the error reports to the console in dev.

## Error boundaries

A root error boundary wraps the whole tree (crash → recovery UI with a
reload action). Each route segment under `ProtectedLayout` inherits it;
no route renders a blank screen on throw.
