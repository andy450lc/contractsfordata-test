import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'

const DOCX_MIME =
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document'
const FALLBACK_FILENAME = 'Robot_Sensor_Data_Content_Development_Agreement.docx'
const ACKNOWLEDGMENT =
  'I acknowledge that this is not legal advice, and that I am responsible for my own legal review before use.'

type Step = 'configure' | 'disclaimer' | 'delivery'
type Pending = 'download' | 'email' | null
type Notice = { kind: 'error' | 'status'; text: string } | null
type DeliveryPreference = 'download' | 'email'
type Answers = {
  subcontracting: string
  jurisdiction_rules: string
  data_ip_liability_cap: string
  data_ip_liability_floor_amount?: string
  data_ip_liability_multiplier?: string
  exclusivity: string
  exclusivity_field_scope?: string
}

interface Props {
  endpoint: string
  emailDeliveryEnabled?: boolean
}

const initialAnswers: Answers = {
  subcontracting: 'buyer_written_consent',
  jurisdiction_rules: 'sow_country_appendix',
  data_ip_liability_cap: 'greater_of_amount_or_fee_multiple',
  exclusivity: 'non_exclusive',
}

const questions = [
  {
    key: 'subcontracting',
    legend: '1. Subcontracting',
    description:
      'Controls whether the developer may involve another company or contractor in the work.',
    options: [
      ['never_allowed', 'A. Never allowed'],
      ['buyer_written_consent', "B. Only with Buyer's written consent (default)"],
      ['notice_with_buyer_objection', 'C. Allowed with notice; Buyer can object'],
    ],
  },
  {
    key: 'jurisdiction_rules',
    legend: '2. Jurisdiction Rules',
    description:
      'Chooses where country-specific collection and transfer rules are recorded.',
    options: [
      ['agreement_country_terms', 'A. Added to the Agreement per country'],
      ['sow_country_appendix', "B. Set in each SOW's country appendix (default)"],
      ['separate_country_rider', 'C. Separate Jurisdiction Rider per country'],
    ],
  },
  {
    key: 'data_ip_liability_cap',
    legend: '3. Liability Cap for data and IP claims',
    description:
      'Chooses the separate cap for data-compliance and intellectual-property claims. Enter the dollar floor and/or fee multiple for the option you choose.',
    options: [
      [
        'greater_of_amount_or_fee_multiple',
        'A. Greater of a dollar floor or a fee multiple (default)',
      ],
      ['fee_multiple_only', 'B. A fee multiple only'],
      ['no_special_cap', 'C. No special cap'],
    ],
  },
  {
    key: 'exclusivity',
    legend: '4. Exclusivity',
    description:
      'Controls whether similar work may be provided to others, globally or in a defined commercial field.',
    options: [
      ['fully_exclusive_global', 'A. Fully exclusive, global'],
      ['exclusive_limited_field', 'B. Exclusive, limited to a field'],
      ['non_exclusive', 'C. Non-exclusive'],
    ],
  },
] as const

function createIdempotencyKey() {
  return `marketing-${crypto.randomUUID()}`
}

function headerFilename(contentDisposition: string | null) {
  if (!contentDisposition) return null
  const encoded = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)?.[1]
  if (encoded) {
    try {
      return decodeURIComponent(encoded)
    } catch {
      return null
    }
  }
  return (
    contentDisposition.match(/filename="([^"]+)"/i)?.[1] ??
    contentDisposition.match(/filename=([^;]+)/i)?.[1]?.trim() ??
    null
  )
}

export function sanitizeDownloadFilename(contentDisposition: string | null) {
  const candidate = headerFilename(contentDisposition)
  if (
    !candidate ||
    candidate.length > 120 ||
    !candidate.toLowerCase().endsWith('.docx') ||
    /[/\\\u0000-\u001f\u007f]/.test(candidate)
  ) {
    return FALLBACK_FILENAME
  }
  const sanitized = candidate.replace(/[^a-zA-Z0-9 ._-]/g, '-').replace(/^\.+/, '')
  return sanitized && sanitized.toLowerCase().endsWith('.docx')
    ? sanitized
    : FALLBACK_FILENAME
}

function isAtBottom(element: HTMLElement) {
  return (
    element.scrollHeight <= element.clientHeight + 2 ||
    element.scrollTop + element.clientHeight >= element.scrollHeight - 2
  )
}

function focusableElements(dialog: HTMLDialogElement) {
  const selector =
    'button:not([disabled]), input:not([disabled]), a[href], select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
  return Array.from(dialog.querySelectorAll<HTMLElement>(selector)).filter(
    (element) => !element.hidden,
  )
}

function Question({
  question,
  value,
  onChange,
}: {
  question: (typeof questions)[number]
  value: string
  onChange: (value: string) => void
}) {
  return (
    <fieldset className="rounded-xl border border-line p-4">
      <legend className="px-1 text-sm font-semibold">{question.legend}</legend>
      <p className="mb-3 text-sm text-muted italic">{question.description}</p>
      <div className="space-y-2.5">
        {question.options.map(([optionValue, label]) => (
          <label key={optionValue} className="flex items-start gap-2.5 text-sm leading-5">
            <input
              type="radio"
              name={question.key}
              value={optionValue}
              checked={value === optionValue}
              className="mt-0.5 size-4 shrink-0 accent-primary"
              onChange={() => onChange(optionValue)}
            />
            <span>{label}</span>
          </label>
        ))}
      </div>
    </fieldset>
  )
}

export function TemplateDeliveryDialog({
  endpoint,
  emailDeliveryEnabled = false,
}: Props) {
  const dialogRef = useRef<HTMLDialogElement>(null)
  const disclaimerRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLElement | null>(null)
  const pendingRef = useRef(false)
  const idempotencyKeyRef = useRef<string | null>(null)
  const [open, setOpen] = useState(false)
  const [step, setStep] = useState<Step>('configure')
  const [hasReachedBottom, setHasReachedBottom] = useState(false)
  const [acknowledged, setAcknowledged] = useState(false)
  const [companyName, setCompanyName] = useState('')
  const [partyType, setPartyType] = useState('')
  const [answers, setAnswers] = useState<Answers>(initialAnswers)
  const [email, setEmail] = useState('')
  const [pending, setPending] = useState<Pending>(null)
  const [notice, setNotice] = useState<Notice>(null)

  const resetGate = useCallback(() => {
    if (disclaimerRef.current) disclaimerRef.current.scrollTop = 0
    setHasReachedBottom(false)
    setAcknowledged(false)
    setEmail('')
    setPending(null)
    setNotice(null)
    pendingRef.current = false
    idempotencyKeyRef.current = null
  }, [])

  const resetAttempt = useCallback(() => {
    setStep('configure')
    resetGate()
  }, [resetGate])

  const closeDialog = useCallback(() => {
    if (pendingRef.current) return
    setOpen(false)
    dialogRef.current?.close()
    resetAttempt()
  }, [resetAttempt])

  useEffect(() => {
    const openFromTrigger = (event: MouseEvent) => {
      const target = event.target
      if (!(target instanceof Element)) return
      const trigger = target.closest<HTMLElement>('[data-open-template]')
      if (!trigger) return
      event.preventDefault()
      triggerRef.current = trigger
      resetAttempt()
      setOpen(true)
    }
    document.addEventListener('click', openFromTrigger)
    return () => document.removeEventListener('click', openFromTrigger)
  }, [resetAttempt])

  useLayoutEffect(() => {
    const dialog = dialogRef.current
    if (!open || !dialog) return
    if (!dialog.open) dialog.showModal()
    dialog.querySelector<HTMLElement>('[data-initial-focus]')?.focus()
  }, [open, step])

  const measureDisclaimer = useCallback(() => {
    const disclaimer = disclaimerRef.current
    if (!disclaimer || !isAtBottom(disclaimer)) return
    setHasReachedBottom((reached) => {
      if (!reached)
        setNotice({ kind: 'status', text: 'Acknowledgment is now available.' })
      return true
    })
  }, [])

  useLayoutEffect(() => {
    const disclaimer = disclaimerRef.current
    if (!open || step !== 'disclaimer' || !disclaimer) return
    measureDisclaimer()
    const observer = new ResizeObserver(measureDisclaimer)
    observer.observe(disclaimer)
    return () => observer.disconnect()
  }, [measureDisclaimer, open, step])

  const handleDialogKeyDown = (event: React.KeyboardEvent<HTMLDialogElement>) => {
    if (event.key === 'Escape') {
      event.preventDefault()
      closeDialog()
      return
    }
    if (event.key !== 'Tab' || !dialogRef.current) return
    const focusable = focusableElements(dialogRef.current)
    const first = focusable[0]
    const last = focusable.at(-1)
    if (!first || !last) return
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first.focus()
    }
  }

  const updateAnswer = (key: keyof Answers, value: string) => {
    setAnswers((current) => ({
      ...current,
      [key]: value,
      ...(key === 'exclusivity' && value !== 'exclusive_limited_field'
        ? { exclusivity_field_scope: undefined }
        : {}),
      ...(key === 'data_ip_liability_cap' && value !== 'greater_of_amount_or_fee_multiple'
        ? { data_ip_liability_floor_amount: undefined }
        : {}),
      ...(key === 'data_ip_liability_cap' && value === 'no_special_cap'
        ? { data_ip_liability_multiplier: undefined }
        : {}),
    }))
    resetGate()
  }

  const requestBody = (preference: DeliveryPreference) => ({
    acknowledged: true,
    client_information: {
      delivery_preference: preference,
      party_type: partyType,
      company_name: companyName.trim(),
    },
    answers: {
      ...answers,
      ...(answers.exclusivity === 'exclusive_limited_field'
        ? { exclusivity_field_scope: answers.exclusivity_field_scope?.trim() }
        : {}),
      ...(answers.data_ip_liability_cap === 'greater_of_amount_or_fee_multiple'
        ? {
            data_ip_liability_floor_amount: Number(
              answers.data_ip_liability_floor_amount,
            ),
            data_ip_liability_multiplier: Number(answers.data_ip_liability_multiplier),
          }
        : {}),
      ...(answers.data_ip_liability_cap === 'fee_multiple_only'
        ? { data_ip_liability_multiplier: Number(answers.data_ip_liability_multiplier) }
        : {}),
    },
  })

  const downloadDocument = async () => {
    if (step !== 'delivery' || !acknowledged || pendingRef.current) return
    pendingRef.current = true
    setPending('download')
    setNotice({ kind: 'status', text: 'Preparing your customized Word document…' })
    try {
      const response = await fetch(`${endpoint}/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody('download')),
      })
      if (
        !response.ok ||
        response.headers.get('Content-Type')?.split(';')[0] !== DOCX_MIME
      ) {
        throw new Error('download failed')
      }
      const blob = await response.blob()
      if (blob.type.split(';')[0] !== DOCX_MIME) throw new Error('invalid document')
      const objectUrl = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = objectUrl
      link.download = sanitizeDownloadFilename(
        response.headers.get('Content-Disposition'),
      )
      link.hidden = true
      document.body.append(link)
      link.click()
      link.remove()
      setTimeout(() => URL.revokeObjectURL(objectUrl), 0)
      setNotice({ kind: 'status', text: 'Your customized Word document is ready.' })
    } catch {
      setNotice({
        kind: 'error',
        text: 'We could not prepare the document. Please try again.',
      })
    } finally {
      pendingRef.current = false
      setPending(null)
    }
  }

  const emailDocument = async (event: React.SubmitEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (
      !emailDeliveryEnabled ||
      step !== 'delivery' ||
      !acknowledged ||
      pendingRef.current
    )
      return
    const input = event.currentTarget.elements.namedItem('email')
    if (!(input instanceof HTMLInputElement) || !input.checkValidity()) {
      setNotice({ kind: 'error', text: 'Enter a valid work email address.' })
      input instanceof HTMLInputElement && input.focus()
      return
    }
    pendingRef.current = true
    setPending('email')
    setNotice({ kind: 'status', text: 'Emailing your customized Word document…' })
    idempotencyKeyRef.current ??= createIdempotencyKey()
    try {
      const response = await fetch(`${endpoint}/email`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': idempotencyKeyRef.current,
        },
        body: JSON.stringify({ ...requestBody('email'), email }),
      })
      const result: unknown = response.headers
        .get('Content-Type')
        ?.includes('application/json')
        ? await response.json()
        : null
      if (
        response.status !== 202 ||
        typeof result !== 'object' ||
        result === null ||
        !('accepted' in result) ||
        result.accepted !== true
      ) {
        throw new Error('email failed')
      }
      setNotice({ kind: 'status', text: 'Sent. Check your inbox.' })
    } catch {
      setNotice({
        kind: 'error',
        text: 'We could not send the document. Please try again.',
      })
    } finally {
      pendingRef.current = false
      setPending(null)
    }
  }

  const title =
    step === 'configure'
      ? 'Customize Your SOW'
      : step === 'disclaimer'
        ? 'Before You Download'
        : 'Choose Delivery'
  const description =
    step === 'configure'
      ? 'Choose the clauses that should appear in your editable Word agreement.'
      : step === 'disclaimer'
        ? 'Read and acknowledge the disclaimer before receiving your agreement.'
        : 'Download the configured Word agreement or send it by email.'

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby="template-dialog-title"
      aria-describedby="template-dialog-description"
      className="template-dialog m-auto w-[calc(100%_-_1.5rem)] max-w-[42rem] rounded-3xl border border-line bg-white p-0 text-ink shadow-float backdrop:bg-ink/40"
      onCancel={(event) => {
        event.preventDefault()
        closeDialog()
      }}
      onClose={() => triggerRef.current?.focus()}
      onClick={(event) => event.target === event.currentTarget && closeDialog()}
      onKeyDown={handleDialogKeyDown}
    >
      <div className="flex max-h-[calc(100svh-1.5rem)] flex-col">
        <header className="flex shrink-0 items-start justify-between gap-4 px-6 pt-6 sm:px-8 sm:pt-8">
          <div>
            <h2 id="template-dialog-title" className="text-[1.75rem] leading-tight">
              {title}
            </h2>
            <p id="template-dialog-description" className="mt-1.5 text-sm text-muted">
              {description}
            </p>
          </div>
          <button
            type="button"
            data-initial-focus
            disabled={pending !== null}
            className="-mt-1 -mr-2 grid size-9 shrink-0 place-items-center rounded-full text-muted hover:bg-card disabled:cursor-not-allowed disabled:opacity-50"
            aria-label="Close"
            onClick={closeDialog}
          >
            <span aria-hidden="true">×</span>
          </button>
        </header>

        {step === 'configure' && (
          <form
            className="flex min-h-0 flex-1 flex-col px-6 pb-6 sm:px-8 sm:pb-8"
            onSubmit={(event) => {
              event.preventDefault()
              if (!event.currentTarget.checkValidity()) {
                event.currentTarget.reportValidity()
                return
              }
              resetGate()
              setStep('disclaimer')
            }}
          >
            <div className="mt-5 min-h-0 space-y-4 overflow-y-auto pr-1">
              <div className="grid gap-4 sm:grid-cols-2">
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  Company name
                  <input
                    type="text"
                    required
                    maxLength={120}
                    value={companyName}
                    className="rounded-md border border-ink/30 bg-white px-3.5 py-2.5 text-base font-normal outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
                    onChange={(event) => {
                      setCompanyName(event.target.value)
                      resetGate()
                    }}
                  />
                </label>
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  Your type
                  <select
                    required
                    value={partyType}
                    className="rounded-md border border-ink/30 bg-white px-3.5 py-2.5 text-base font-normal outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
                    onChange={(event) => {
                      setPartyType(event.target.value)
                      resetGate()
                    }}
                  >
                    <option value="">Select a type</option>
                    <option value="supplier">Supplier</option>
                    <option value="buyer">Buyer</option>
                    <option value="other">Other</option>
                  </select>
                </label>
              </div>
              {questions.map((question) => (
                <div key={question.key} className="contents">
                  <Question
                    question={question}
                    value={answers[question.key] ?? ''}
                    onChange={(value) => updateAnswer(question.key, value)}
                  />
                  {question.key === 'data_ip_liability_cap' &&
                    answers.data_ip_liability_cap !== 'no_special_cap' && (
                      <div className="grid gap-4 sm:grid-cols-2">
                        {answers.data_ip_liability_cap ===
                          'greater_of_amount_or_fee_multiple' && (
                          <label className="flex flex-col gap-1.5 text-sm font-medium">
                            Dollar floor ($)
                            <input
                              type="number"
                              required
                              min="0.01"
                              max="100000000"
                              step="0.01"
                              inputMode="decimal"
                              value={answers.data_ip_liability_floor_amount ?? ''}
                              placeholder="e.g. 50000"
                              className="rounded-md border border-ink/30 bg-white px-3.5 py-2.5 text-base font-normal outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
                              onChange={(event) =>
                                updateAnswer(
                                  'data_ip_liability_floor_amount',
                                  event.target.value,
                                )
                              }
                            />
                          </label>
                        )}
                        <label className="flex flex-col gap-1.5 text-sm font-medium">
                          Fee multiplier (×)
                          <input
                            type="number"
                            required
                            min="0.01"
                            max="1000"
                            step="0.01"
                            inputMode="decimal"
                            value={answers.data_ip_liability_multiplier ?? ''}
                            placeholder="e.g. 3"
                            className="rounded-md border border-ink/30 bg-white px-3.5 py-2.5 text-base font-normal outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
                            onChange={(event) =>
                              updateAnswer(
                                'data_ip_liability_multiplier',
                                event.target.value,
                              )
                            }
                          />
                        </label>
                      </div>
                    )}
                </div>
              ))}
              {answers.exclusivity === 'exclusive_limited_field' && (
                <label className="flex flex-col gap-1.5 text-sm font-medium">
                  Exclusivity field
                  <input
                    type="text"
                    required
                    maxLength={160}
                    value={answers.exclusivity_field_scope ?? ''}
                    placeholder="e.g. autonomous warehouse robotics"
                    className="rounded-md border border-ink/30 bg-white px-3.5 py-2.5 text-base font-normal outline-none focus:border-primary focus:ring-2 focus:ring-primary/20"
                    onChange={(event) =>
                      updateAnswer('exclusivity_field_scope', event.target.value)
                    }
                  />
                </label>
              )}
            </div>
            <div className="mt-5 flex shrink-0 flex-col-reverse gap-2 border-t border-line pt-4 sm:flex-row sm:justify-end">
              <button type="button" className="btn btn-ghost" onClick={closeDialog}>
                Cancel
              </button>
              <button type="submit" className="btn btn-primary">
                Review disclaimer
              </button>
            </div>
          </form>
        )}

        {step === 'disclaimer' && (
          <div className="flex min-h-0 flex-1 flex-col px-6 pb-6 sm:px-8 sm:pb-8">
            <div
              ref={disclaimerRef}
              data-testid="disclaimer-scroll"
              className="mt-5 min-h-0 max-h-[min(18rem,40svh)] overflow-y-auto rounded-xl border border-line bg-card p-4 text-sm leading-6 sm:p-5"
              onScroll={measureDisclaimer}
              tabIndex={0}
            >
              <p>
                This template is provided for general reference only and does not
                constitute legal advice. It has not been reviewed by a licensed attorney
                for your specific situation, and no attorney-client relationship is
                created by downloading or using it.
              </p>
              <p className="mt-4">
                Data-collection and data-transfer laws vary by jurisdiction and change
                over time. You are responsible for reviewing, adapting, and validating
                this document — including consulting a lawyer licensed in each relevant
                jurisdiction — before relying on it or using it in any transaction.
              </p>
            </div>
            <p className="mt-2 text-xs text-muted">
              {hasReachedBottom
                ? 'You can now acknowledge the disclaimer.'
                : 'Scroll to the bottom to enable the acknowledgment.'}
            </p>
            <label className="mt-4 flex items-start gap-3 text-sm leading-5">
              <input
                type="checkbox"
                className="mt-0.5 size-4 shrink-0 accent-primary"
                disabled={!hasReachedBottom}
                checked={acknowledged}
                onChange={(event) => setAcknowledged(event.target.checked)}
              />
              <span>{ACKNOWLEDGMENT}</span>
            </label>
            <div className="mt-6 flex shrink-0 flex-col-reverse gap-2 sm:flex-row sm:justify-between">
              <button
                type="button"
                className="btn btn-ghost"
                onClick={() => {
                  setStep('configure')
                  resetGate()
                }}
              >
                Back
              </button>
              <div className="flex flex-col-reverse gap-2 sm:flex-row">
                <button type="button" className="btn btn-ghost" onClick={closeDialog}>
                  Cancel
                </button>
                <button
                  type="button"
                  className="btn btn-primary"
                  disabled={!acknowledged}
                  onClick={() => {
                    setStep('delivery')
                    setNotice(null)
                  }}
                >
                  Continue
                </button>
              </div>
            </div>
            <p role="status" aria-live="polite" className="sr-only">
              {notice?.kind === 'status' ? notice.text : ''}
            </p>
          </div>
        )}

        {step === 'delivery' && (
          <div className="flex min-h-0 flex-1 flex-col px-6 pb-6 sm:px-8 sm:pb-8">
            <div className="mt-6 min-h-0 overflow-y-auto">
              <p className="text-sm leading-6 text-muted">
                Your four clause choices are ready. Both delivery methods use the same
                customized Microsoft Word document.
              </p>
              <button
                type="button"
                className="btn btn-primary mt-5 w-full py-3 text-base"
                disabled={pending !== null}
                onClick={downloadDocument}
              >
                {pending === 'download'
                  ? 'Preparing Word document…'
                  : 'Download Word document'}
              </button>
              {emailDeliveryEnabled ? (
                <form
                  className="mt-5 border-t border-line pt-5"
                  noValidate
                  onSubmit={emailDocument}
                >
                  <label className="flex flex-col gap-1.5 text-sm font-medium">
                    Work email
                    <input
                      type="email"
                      name="email"
                      required
                      maxLength={254}
                      autoComplete="email"
                      placeholder="you@company.com"
                      value={email}
                      disabled={pending !== null}
                      className="rounded-md border border-ink/30 bg-white px-3.5 py-2.5 text-base font-normal outline-none placeholder:text-muted/70 focus:border-primary focus:ring-2 focus:ring-primary/20 disabled:bg-card"
                      onChange={(event) => {
                        setEmail(event.target.value)
                        idempotencyKeyRef.current = null
                        if (notice?.kind === 'error') setNotice(null)
                      }}
                    />
                  </label>
                  <button
                    type="submit"
                    className="btn btn-primary mt-3 w-full py-3 text-base"
                    disabled={pending !== null}
                  >
                    {pending === 'email'
                      ? 'Emailing Word document…'
                      : 'Email Word document'}
                  </button>
                </form>
              ) : (
                <p className="mt-4 rounded-lg bg-card p-3 text-sm text-muted">
                  Email delivery will appear after the production sending domain and
                  provider have passed an end-to-end verification.
                </p>
              )}
              {notice?.kind === 'error' && (
                <p role="alert" className="mt-4 text-sm text-red-700">
                  {notice.text}
                </p>
              )}
              {notice?.kind === 'status' && (
                <p role="status" aria-live="polite" className="mt-4 text-sm font-medium">
                  {notice.text}
                </p>
              )}
            </div>
            <div className="mt-6 flex shrink-0 flex-col-reverse gap-2 border-t border-line pt-4 sm:flex-row sm:justify-between">
              <button
                type="button"
                className="btn btn-ghost"
                disabled={pending !== null}
                onClick={() => {
                  setStep('configure')
                  resetGate()
                }}
              >
                Change answers
              </button>
              <button
                type="button"
                className="btn btn-ghost"
                disabled={pending !== null}
                onClick={closeDialog}
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>
    </dialog>
  )
}
