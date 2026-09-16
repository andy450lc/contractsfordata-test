/// <reference types="vitest/config" />
import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import fs from 'node:fs'
import path from 'node:path'
import { defineConfig, loadEnv, type Plugin } from 'vite'

const API_ORIGIN_TOKEN = '__API_ORIGIN__'

/**
 * `public/_headers` carries the Content-Security-Policy that Cloudflare
 * serves with the app. Its `connect-src` must name the API origin, which
 * differs per stage, so the checked-in file holds a token and this plugin
 * replaces it in `dist/_headers` with the origin of `VITE_API_URL` after
 * each production build. The build fails if the URL is missing or invalid
 * rather than shipping a policy that blocks every API call.
 */
function cspApiOrigin(mode: string): Plugin {
  let outDir = 'dist'
  return {
    name: 'sow:csp-api-origin',
    apply: 'build',
    configResolved(config) {
      outDir = config.build.outDir
    },
    closeBundle() {
      const raw = loadEnv(mode, process.cwd()).VITE_API_URL ?? process.env.VITE_API_URL
      let origin: string
      try {
        origin = new URL(raw ?? '').origin
      } catch {
        throw new Error(
          `VITE_API_URL must be an absolute URL (got ${JSON.stringify(raw)}); it is stamped into the CSP connect-src in _headers`,
        )
      }
      const file = path.join(outDir, '_headers')
      const contents = fs.readFileSync(file, 'utf8')
      if (!contents.includes(API_ORIGIN_TOKEN)) {
        throw new Error(`${file} no longer contains ${API_ORIGIN_TOKEN}`)
      }
      fs.writeFileSync(file, contents.replaceAll(API_ORIGIN_TOKEN, origin))
    },
  }
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [react(), tailwindcss(), cspApiOrigin(mode)],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**'],
      exclude: [
        'src/components/ui/**', // shadcn primitives: generated, then owned
        'src/main.tsx', // pure wiring
        'src/test/**',
        'src/**/*.test.{ts,tsx}',
      ],
      thresholds: {
        lines: 85,
        functions: 85,
        branches: 85,
        statements: 85,
      },
    },
  },
}))
