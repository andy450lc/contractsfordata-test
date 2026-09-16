import type { APIRoute } from 'astro'

// The robots spec wants the sitemap as an absolute URL, and the origin
// differs per stage, so this file is generated at build time rather than
// kept under public/.
export const GET: APIRoute = ({ site }) => {
  const body = [
    'User-agent: *',
    'Allow: /',
    '',
    `Sitemap: ${new URL('/sitemap-index.xml', site).href}`,
    '',
  ].join('\n')
  return new Response(body, { headers: { 'Content-Type': 'text/plain; charset=utf-8' } })
}
