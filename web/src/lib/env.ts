import { z } from 'zod'

const envSchema = z.object({
  VITE_API_URL: z.url('VITE_API_URL must be a URL'),
})

export type Env = z.infer<typeof envSchema>

// parseEnv validates a raw env record and returns the typed result.
// Throws with the offending variable names when validation fails.
export function parseEnv(raw: Record<string, unknown>): Env {
  const result = envSchema.safeParse(raw)

  if (!result.success) {
    const fields = result.error.issues.map((issue) => issue.path.join('.')).join(', ')
    throw new Error(`Invalid environment configuration: ${fields}`)
  }

  return result.data
}

export const env = parseEnv(import.meta.env)
