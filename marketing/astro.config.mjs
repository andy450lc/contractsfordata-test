// @ts-check
import react from '@astrojs/react'
import sitemap from '@astrojs/sitemap'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig, envField } from 'astro/config'
import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import { loadEnv } from 'vite'

// Astro reads .env files for pages and components but not for this file,
// so the public URL of the site is loaded here by hand. It becomes the
// canonical origin in every page's <head> and the base of the sitemap.
const { PUBLIC_SITE_URL, PUBLIC_TEMPLATE_DELIVERY_URL } = {
  ...loadEnv('', process.cwd(), 'PUBLIC_'),
  ...process.env,
}
let site
/** @type {string} */
let templateDeliveryOrigin
try {
  site = new URL(PUBLIC_SITE_URL ?? '').origin
} catch {
  throw new Error(
    `PUBLIC_SITE_URL must be an absolute URL (got ${JSON.stringify(PUBLIC_SITE_URL)}); copy .env.example to .env`,
  )
}
try {
  templateDeliveryOrigin = new URL(PUBLIC_TEMPLATE_DELIVERY_URL ?? '').origin
} catch {
  throw new Error(
    `PUBLIC_TEMPLATE_DELIVERY_URL must be an absolute URL (got ${JSON.stringify(PUBLIC_TEMPLATE_DELIVERY_URL)}); copy .env.example to .env`,
  )
}

/**
 * public/_headers carries the Content-Security-Policy. Its connect-src must
 * name the origin the template form posts to, which differs per stage, so
 * the checked-in file holds a token and this hook replaces it in
 * dist/_headers after each build with the delivery API origin.
 *
 * @returns {import('astro').AstroIntegration}
 */
function cspConnectSrc() {
  const TOKEN = '__CONNECT_SRC__'
  const extra = ` ${templateDeliveryOrigin}`
  return {
    name: 'sow:csp-connect-src',
    hooks: {
      'astro:build:done': ({ dir }) => {
        const file = fileURLToPath(new URL('_headers', dir))
        const contents = fs.readFileSync(file, 'utf8')
        if (!contents.includes(TOKEN))
          throw new Error(`${file} no longer contains ${TOKEN}`)
        fs.writeFileSync(file, contents.replaceAll(TOKEN, extra))
      },
    },
  }
}

// https://astro.build/config
export default defineConfig({
  site,
  // Every page is rendered to HTML at build time. Nothing here needs a
  // request-time server; a route that ever does opts out with
  // `export const prerender = false` after adding the Cloudflare adapter.
  output: 'static',
  integrations: [react(), sitemap(), cspConnectSrc()],
  env: {
    schema: {
      // Origin of the web app (sign-in, sign-up). Stamped into links at
      // build time; the build fails on a missing or malformed value.
      PUBLIC_APP_ORIGIN: envField.string({
        context: 'client',
        access: 'public',
        url: true,
      }),
      // Base URL for the public Word download and email operations.
      PUBLIC_TEMPLATE_DELIVERY_URL: envField.string({
        context: 'client',
        access: 'public',
        url: true,
      }),
      // Enables public email only after the provider has passed a real send.
      PUBLIC_CONFIGURATOR_EMAIL_ENABLED: envField.boolean({
        context: 'client',
        access: 'public',
        default: false,
      }),
    },
  },
  build: {
    // public/_headers ships a CSP with `style-src 'self'`; inlined
    // stylesheets would be blocked by it, so every stylesheet is a file.
    inlineStylesheets: 'never',
  },
  vite: {
    plugins: [tailwindcss()],
    // Small scripts would otherwise be inlined into the HTML, which the CSP
    // (`script-src 'self'`, no inline scripts) blocks. Keep them as files.
    build: { assetsInlineLimit: 0 },
  },
})
