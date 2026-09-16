import { useMutation } from '@tanstack/react-query'

import { api, ApiError } from '@/lib/api/client'
import type { components } from '@/lib/api/schema'

type ApiErrorBody = components['schemas']['Error']
type ContractConfiguration = components['schemas']['ContractConfiguration']
type EmailDeliveryAccepted = components['schemas']['EmailDeliveryAccepted']

const docxMime = 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'

export interface DownloadedDocument {
  blob: Blob
  filename: string
}

export interface EmailDocumentVariables {
  configuration: ContractConfiguration
  email: string
  idempotencyKey: string
}

function deliveryError(response: Response, error: ApiErrorBody | undefined): ApiError {
  return new ApiError(response.status, error?.error ?? 'unknown_error', error?.details)
}

function decodedFilename(value: string): string {
  try {
    return decodeURIComponent(value)
  } catch {
    return value
  }
}

function baseMime(value: string | null): string {
  return value?.split(';')[0]?.trim().toLowerCase() ?? ''
}

// isBlobLike duck-types Blob instead of using `instanceof`, since a native
// fetch response and the test environment's global can construct Blobs
// from different realms.
function isBlobLike(value: unknown): value is Blob {
  return (
    typeof value === 'object' &&
    value !== null &&
    typeof (value as Blob).type === 'string' &&
    typeof (value as Blob).size === 'number' &&
    typeof (value as Blob).arrayBuffer === 'function'
  )
}

// documentFilename extracts a safe local filename from Content-Disposition.
export function documentFilename(contentDisposition: string | null): string {
  const extended = contentDisposition?.match(/filename\*\s*=\s*(?:UTF-8'')?([^;]+)/i)
  const regular = contentDisposition?.match(/filename\s*=\s*(?:"([^"]*)"|([^;]+))/i)
  const encoded = extended?.[1]?.trim().replace(/^"|"$/g, '')
  const plain = regular?.[1] ?? regular?.[2]?.trim()
  const candidate = decodedFilename(encoded ?? plain ?? 'agreement.docx')
    .replaceAll('\\', '/')
    .split('/')
    .pop()
    ?.replace(/[^\x20-\x7e]/g, '')
    .replace(/[^a-zA-Z0-9 ._()'-]/g, '_')
    .replace(/^\.+/, '')
    .trim()
    .slice(0, 120)
  const safe = candidate === undefined || candidate === '' ? 'agreement.docx' : candidate
  if (safe.toLowerCase().endsWith('.docx')) return safe
  const stem = safe.replace(/\.[^.]*$/, '') || 'agreement'
  return `${stem}.docx`
}

// useDownloadTemplateDocument requests one configured OOXML artifact.
export function useDownloadTemplateDocument() {
  return useMutation<DownloadedDocument, Error, ContractConfiguration>({
    mutationFn: async (configuration) => {
      const { data, error, response } = await api.POST('/template-deliveries/download', {
        body: { acknowledged: true, configuration },
        parseAs: 'blob',
      })
      if (error !== undefined || data === undefined) {
        throw deliveryError(response, error)
      }
      const payload: unknown = data
      const responseMime = baseMime(response.headers.get('Content-Type'))
      if (
        !isBlobLike(payload) ||
        responseMime !== docxMime ||
        baseMime(payload.type) !== docxMime
      ) {
        throw new ApiError(response.status, 'invalid_document_response')
      }
      return {
        blob: payload,
        filename: documentFilename(response.headers.get('Content-Disposition')),
      }
    },
  })
}

// useEmailTemplateDocument requests provider-confirmed email delivery.
export function useEmailTemplateDocument() {
  return useMutation<EmailDeliveryAccepted, Error, EmailDocumentVariables>({
    mutationFn: async ({ configuration, email, idempotencyKey }) => {
      const { data, error, response } = await api.POST('/template-deliveries/email', {
        params: { header: { 'Idempotency-Key': idempotencyKey } },
        body: { acknowledged: true, email, configuration },
      })
      if (error !== undefined || data === undefined || data.accepted !== true) {
        throw deliveryError(response, error)
      }
      return data
    },
  })
}
