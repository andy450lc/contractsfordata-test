import { useCallback, useLayoutEffect, useRef, useState, type ReactElement } from 'react'
import { z } from 'zod'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useMe } from '@/features/auth'

import { useDownloadTemplateDocument, useEmailTemplateDocument } from '../delivery/api'
import { contractConfiguration } from '../delivery/configuration'
import type { Organization } from '../schemas'
import type { Draft } from '../store'

const disclaimerParagraphs = [
  'This template is provided for general reference only and does not constitute legal advice. It has not been reviewed by a licensed attorney for your specific situation, and no attorney-client relationship is created by downloading or using it.',
  'Data-collection and data-transfer laws vary by jurisdiction and change over time. You are responsible for reviewing, adapting, and validating this document — including consulting a lawyer licensed in each relevant jurisdiction — before relying on it or using it in any transaction.',
] as const

const acknowledgmentLabel =
  'I acknowledge that this is not legal advice, and that I am responsible for my own legal review before use.'

const workEmailSchema = z.email('Enter a valid work email.').max(254)

interface DeliveryError {
  title: string
  message: string
}

interface TemplateDeliveryDialogProps {
  draft: Draft
  organization: Organization | null
  trigger: ReactElement
}

function saveDocument(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.append(anchor)
  anchor.click()
  anchor.remove()
  window.setTimeout(() => URL.revokeObjectURL(url), 0)
}

function configurationFingerprint(draft: Draft, organization: Organization | null) {
  return JSON.stringify({
    templateVersion: draft.templateVersion,
    organization,
    steps: draft.steps,
  })
}

// TemplateDeliveryDialog gates configured Word delivery behind an acknowledgment.
export function TemplateDeliveryDialog({
  draft,
  organization,
  trigger,
}: TemplateDeliveryDialogProps) {
  const me = useMe()
  const download = useDownloadTemplateDocument()
  const emailDelivery = useEmailTemplateDocument()
  const [open, setOpen] = useState(false)
  const [step, setStep] = useState<'disclaimer' | 'delivery'>('disclaimer')
  const [hasReachedBottom, setHasReachedBottom] = useState(false)
  const [acknowledged, setAcknowledged] = useState(false)
  const [email, setEmail] = useState('')
  const [emailError, setEmailError] = useState('')
  const [error, setError] = useState<DeliveryError | null>(null)
  const [emailAccepted, setEmailAccepted] = useState(false)
  const [disclaimerElement, setDisclaimerElement] = useState<HTMLDivElement | null>(null)
  const inFlightRef = useRef(false)
  const emailKeyRef = useRef<{ fingerprint: string; key: string } | null>(null)
  const fingerprint = configurationFingerprint(draft, organization)
  const [attemptFingerprint, setAttemptFingerprint] = useState(fingerprint)
  const activeFingerprintRef = useRef(fingerprint)
  const pending = download.isPending || emailDelivery.isPending

  const measureDisclaimer = useCallback((container: HTMLDivElement) => {
    const atBottom =
      container.scrollHeight <= container.clientHeight + 2 ||
      container.scrollTop + container.clientHeight >= container.scrollHeight - 2
    if (atBottom) setHasReachedBottom(true)
  }, [])

  const captureDisclaimer = useCallback(
    (element: HTMLDivElement | null) => {
      if (element !== null) {
        element.scrollTop = 0
        window.queueMicrotask(() => measureDisclaimer(element))
      }
      setDisclaimerElement(element)
    },
    [measureDisclaimer],
  )

  if (attemptFingerprint !== fingerprint) {
    setAttemptFingerprint(fingerprint)
    setStep('disclaimer')
    setHasReachedBottom(false)
    setAcknowledged(false)
    setEmail(me.data?.email ?? '')
    setEmailError('')
    setError(null)
    setEmailAccepted(false)
  }

  function resetAttempt() {
    setStep('disclaimer')
    setHasReachedBottom(false)
    setAcknowledged(false)
    setEmail(me.data?.email ?? '')
    setEmailError('')
    setError(null)
    setEmailAccepted(false)
    emailKeyRef.current = null
    inFlightRef.current = false
    download.reset()
    emailDelivery.reset()
  }

  function changeOpen(nextOpen: boolean) {
    if (!nextOpen && inFlightRef.current) return
    if (nextOpen) resetAttempt()
    setOpen(nextOpen)
    if (!nextOpen) resetAttempt()
  }

  useLayoutEffect(() => {
    if (!open || step !== 'disclaimer') return
    const container = disclaimerElement
    if (container === null) return
    const observer = new ResizeObserver(() => measureDisclaimer(container))
    observer.observe(container)
    return () => observer.disconnect()
  }, [disclaimerElement, measureDisclaimer, open, step])

  useLayoutEffect(() => {
    activeFingerprintRef.current = fingerprint
  }, [fingerprint])

  function configurationForRequest() {
    try {
      return contractConfiguration(draft, organization)
    } catch {
      setError({
        title: 'Agreement needs attention',
        message: 'Complete every required agreement field before delivery.',
      })
      return null
    }
  }

  async function downloadDocument() {
    if (step !== 'delivery' || !acknowledged || inFlightRef.current) return
    const configuration = configurationForRequest()
    if (configuration === null) return
    inFlightRef.current = true
    const requestFingerprint = fingerprint
    setError(null)
    try {
      const document = await download.mutateAsync(configuration)
      if (activeFingerprintRef.current !== requestFingerprint) {
        inFlightRef.current = false
        return
      }
      saveDocument(document.blob, document.filename)
      inFlightRef.current = false
      changeOpen(false)
    } catch {
      inFlightRef.current = false
      setError({
        title: 'Download failed',
        message: "We couldn't generate your Word document. Try again.",
      })
    }
  }

  async function emailDocument() {
    if (step !== 'delivery' || !acknowledged || inFlightRef.current) return
    const normalizedEmail = email.trim()
    const parsed = workEmailSchema.safeParse(normalizedEmail)
    if (!parsed.success) {
      setEmailError(parsed.error.issues[0]?.message ?? 'Enter a valid work email.')
      return
    }
    const configuration = configurationForRequest()
    if (configuration === null) return
    const idempotencyFingerprint = `${fingerprint}:${normalizedEmail}`
    if (emailKeyRef.current?.fingerprint !== idempotencyFingerprint) {
      emailKeyRef.current = {
        fingerprint: idempotencyFingerprint,
        key: crypto.randomUUID(),
      }
    }
    inFlightRef.current = true
    setEmailError('')
    setError(null)
    setEmailAccepted(false)
    try {
      await emailDelivery.mutateAsync({
        configuration,
        email: normalizedEmail,
        idempotencyKey: emailKeyRef.current.key,
      })
      if (activeFingerprintRef.current !== fingerprint) return
      setEmailAccepted(true)
    } catch {
      setError({
        title: 'Email delivery failed',
        message: "We couldn't email your Word document. Try again.",
      })
    } finally {
      inFlightRef.current = false
    }
  }

  return (
    <Dialog open={open} onOpenChange={changeOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent
        className="flex max-h-[calc(100svh-2rem)] flex-col sm:max-w-lg"
        showCloseButton={!pending}
        onEscapeKeyDown={(event) => pending && event.preventDefault()}
        onInteractOutside={(event) => pending && event.preventDefault()}
      >
        <DialogHeader className="shrink-0 pr-8">
          <DialogTitle>Before You Download</DialogTitle>
          <DialogDescription>
            {step === 'disclaimer'
              ? 'Review this information before choosing a delivery method.'
              : 'Choose how you want to receive your configured Word document.'}
          </DialogDescription>
        </DialogHeader>

        {step === 'disclaimer' ? (
          <>
            {/* A scrollable region needs a keyboard focus target. */}
            {/* eslint-disable jsx-a11y/no-noninteractive-tabindex */}
            <div
              ref={captureDisclaimer}
              aria-label="Legal disclaimer"
              role="region"
              className="min-h-0 overflow-y-auto rounded-lg border bg-muted/30 p-4 text-sm leading-relaxed sm:max-h-56"
              onScroll={(event) => measureDisclaimer(event.currentTarget)}
              tabIndex={0}
            >
              <p>{disclaimerParagraphs[0]}</p>
              <p className="mt-4">{disclaimerParagraphs[1]}</p>
            </div>
            {/* eslint-enable jsx-a11y/no-noninteractive-tabindex */}
            <p className="shrink-0 text-xs text-muted-foreground">
              Scroll to the bottom to enable the acknowledgment.
            </p>
            <div className="flex shrink-0 items-start gap-3 rounded-lg border p-3">
              <input
                id={`acknowledgment-${draft.id}`}
                type="checkbox"
                checked={acknowledged}
                disabled={!hasReachedBottom}
                className="mt-0.5 size-4 shrink-0 accent-primary outline-none focus-visible:ring-3 focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50"
                onChange={(event) => setAcknowledged(event.target.checked)}
              />
              <Label
                htmlFor={`acknowledgment-${draft.id}`}
                className="items-start leading-relaxed"
              >
                {acknowledgmentLabel}
              </Label>
            </div>
            {hasReachedBottom ? (
              <p role="status" className="sr-only">
                Acknowledgment is now available.
              </p>
            ) : null}
            <DialogFooter className="shrink-0">
              <Button type="button" variant="ghost" onClick={() => changeOpen(false)}>
                Cancel
              </Button>
              <Button
                type="button"
                disabled={!acknowledged}
                onClick={() => setStep('delivery')}
              >
                Continue
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <div className="flex min-h-0 flex-col gap-4 overflow-y-auto">
              {error ? (
                <Alert variant="destructive" aria-label={error.title}>
                  <AlertTitle>{error.title}</AlertTitle>
                  <AlertDescription>{error.message}</AlertDescription>
                </Alert>
              ) : null}
              {emailAccepted ? (
                <Alert aria-live="polite">
                  <AlertTitle>Email accepted</AlertTitle>
                  <AlertDescription>
                    Your Word document was accepted for email delivery.
                  </AlertDescription>
                </Alert>
              ) : null}
              <Button
                type="button"
                size="lg"
                disabled={pending}
                onClick={() => void downloadDocument()}
              >
                {download.isPending
                  ? 'Downloading Word document…'
                  : 'Download Word document'}
              </Button>
              <div className="flex flex-col gap-2">
                <Label htmlFor={`delivery-email-${draft.id}`}>Work email</Label>
                <Input
                  id={`delivery-email-${draft.id}`}
                  type="email"
                  autoComplete="email"
                  value={email}
                  disabled={pending}
                  aria-invalid={emailError !== ''}
                  aria-describedby={
                    emailError === '' ? undefined : `email-error-${draft.id}`
                  }
                  onChange={(event) => {
                    setEmail(event.target.value)
                    setEmailError('')
                    setEmailAccepted(false)
                  }}
                />
                {emailError !== '' ? (
                  <p id={`email-error-${draft.id}`} className="text-sm text-destructive">
                    {emailError}
                  </p>
                ) : null}
              </div>
              <Button
                type="button"
                size="lg"
                disabled={pending}
                onClick={() => void emailDocument()}
              >
                {emailDelivery.isPending
                  ? 'Emailing Word document…'
                  : 'Email Word document'}
              </Button>
            </div>
            <DialogFooter className="shrink-0">
              <Button
                type="button"
                variant="ghost"
                disabled={pending}
                onClick={() => changeOpen(false)}
              >
                Cancel
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled={pending}
                onClick={() => {
                  setError(null)
                  setEmailAccepted(false)
                  setStep('disclaimer')
                }}
              >
                Back
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
