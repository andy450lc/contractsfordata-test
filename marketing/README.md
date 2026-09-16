# marketing

The public landing site at the apex domain. Astro, rendered to static HTML
at build time, deployed as Cloudflare Workers static assets. The template
delivery dialog is a small React island; the rest of the site is static HTML.

## Run it

```bash
cp .env.example .env   # once
npm install
npm run dev
```

## Check it

```bash
npm run check && npm run format:check && npm run build && npx wrangler deploy --dry-run
```

## How it is put together

- `src/layouts/Base.astro` owns `<head>`: title, description, canonical
  URL, Open Graph and Twitter tags, and the sitemap link. Pages pass a
  title and description and nothing else.
- `src/styles/global.css` holds the theme, all self-hosted: Sofia Sans
  Extra Condensed for the 72px headline and the wordmark; Sofia Sans at
  normal width for section headings (24px), the FAQ heading (40px), and
  the hero buttons (20px); Instrument Sans for everything else, on a measured reference scale
  (body 18px at 1.6, subheads and accordion titles 18px medium, nav and
  contents 14px with 0.4px tracking). White background, near-black ink,
  and the favicon's blue for buttons and links. There is no dark mode.
- `PUBLIC_APP_ORIGIN` is the web app's origin. Links to sign in and sign
  up are built from it at build time. `PUBLIC_SITE_URL` is this site's own
  origin, used for canonical URLs and the sitemap. Both are required and
  validated; the build fails rather than shipping placeholder links.
- `PUBLIC_TEMPLATE_DELIVERY_URL` is the API base for the acknowledged public
  four-question Word configurator, for example
  `https://api.example.com/v1/public/sow-configurator`. The dialog posts the
  selected answers to `/download` and `/email` beneath it.
- `PUBLIC_CONFIGURATOR_EMAIL_ENABLED` defaults to `false`. Set it to `true`
  only after the Resend provider and sending domain have passed a real
  end-to-end attachment send.
- `public/_headers` is the Cloudflare headers file. Its CSP allows scripts
  only as same-origin files (`script-src 'self'`, no inline scripts), so
  `astro.config.mjs` keeps every script external. The origin of
  `PUBLIC_TEMPLATE_DELIVERY_URL` is stamped into `connect-src` after each
  build.
- `src/components/TemplateModal.astro` mounts the global delivery island. All
  `data-open-template` buttons enter the same two-stage disclaimer flow. The
  backend returns the Word document bytes; no static contract file is public.
- `src/pages/404.astro` becomes `dist/404.html`, which
  `not_found_handling: "404-page"` in `wrangler.jsonc` serves for unknown
  paths.

## Adding a page

Create `src/pages/<name>.astro`, wrap it in `Base`, and give it a title and
a description. The sitemap picks it up on the next build.

## Deployment

- Set `PUBLIC_SITE_URL`, `PUBLIC_APP_ORIGIN`, and
  `PUBLIC_TEMPLATE_DELIVERY_URL` for every stage before building.
- Add the marketing origin to the API's `CORS_ALLOWED_ORIGINS`. The delivery
  base must expose `POST /download` and `POST /email`.
- Configure `RESEND_API_KEY` and `EMAIL_FROM` on the backend, and verify the
  sender domain in Resend. An unconfigured development mail transport returns
  an error; the site does not claim the email was sent.
- Keep `PUBLIC_TEMPLATE_DELIVERY_URL` public. Provider credentials and sender
  configuration remain server-side.
- An Open Graph image (`og:image`) remains a future brand asset.
