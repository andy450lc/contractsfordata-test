# Quickstart: Gated Word Template Delivery

1. Configure the backend and set `RESEND_API_KEY` plus a verified `EMAIL_FROM`
   for real email delivery.
2. Add both frontend origins to `CORS_ALLOWED_ORIGINS`.
3. Set web `VITE_API_URL` and marketing `PUBLIC_TEMPLATE_DELIVERY_URL`.
4. Run backend, web, and marketing development servers.
5. Exercise every CTA, keyboard/scroll/reset behavior, configured download,
   blank-template download, confirmed email, provider failure, and retry.
6. Inspect a returned file as ZIP and in Word-compatible software.
7. Run codegen-diff, format, lint, typecheck, unit/integration, coverage, build,
   and Cloudflare dry-run gates.

