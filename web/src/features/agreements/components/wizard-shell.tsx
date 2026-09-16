import { ArrowLeftIcon } from 'lucide-react'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'

import { Button, buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export const WIZARD_STEPS = [
  'Agreement',
  'Scope',
  'Pricing & schedule',
  'Environment & task mix',
  'Technical standards',
  'Delivery & acceptance',
  'Developer',
  'Review',
] as const

export const STEP_COUNT = WIZARD_STEPS.length

interface WizardShellProps {
  step: number
  heading: string
  subtitle: string
  onBack?: () => void
  backTo?: string
  nextLabel?: string
  nextDisabled?: boolean
  nextNote?: string
  extraAction?: ReactNode
  note?: ReactNode
  children: ReactNode
}

// WizardShell frames a step with the title, the progress bar, and the
// Back and Next controls. Next submits the enclosing form.
export function WizardShell({
  step,
  heading,
  subtitle,
  onBack,
  backTo,
  nextLabel = 'Next',
  nextDisabled,
  nextNote,
  extraAction,
  note,
  children,
}: WizardShellProps) {
  const stepName = WIZARD_STEPS[step - 1] ?? ''
  const backClass = buttonVariants({ variant: 'ghost' })

  return (
    <div className="mx-auto w-full max-w-3xl rounded-xl bg-card p-6 shadow-sm ring-1 ring-foreground/10 sm:p-8">
      <div className="relative mb-8 flex flex-col items-center gap-3">
        {backTo ? (
          <Link
            to={backTo}
            aria-label="Back to my agreements"
            className="absolute top-0 left-0 text-muted-foreground hover:text-foreground"
          >
            <ArrowLeftIcon className="size-5" />
          </Link>
        ) : (
          <button
            type="button"
            onClick={onBack}
            aria-label="Previous step"
            className="absolute top-0 left-0 text-muted-foreground hover:text-foreground"
          >
            <ArrowLeftIcon className="size-5" />
          </button>
        )}
        <h1 className="text-2xl font-semibold tracking-tight">Create my agreement</h1>
        <nav
          aria-label="Form progress"
          className="flex w-full flex-col items-center gap-2"
        >
          <div className="flex w-full items-center justify-between text-sm">
            <span className="w-12" />
            <span className="font-medium">{stepName}</span>
            <span className="w-12 text-right text-muted-foreground">
              {step}/{STEP_COUNT}
            </span>
          </div>
          <span className="sr-only">
            Step {step} of {STEP_COUNT}
          </span>
          <ol className="flex w-full max-w-sm gap-1.5">
            {WIZARD_STEPS.map((name, index) => (
              <li
                key={name}
                aria-current={index + 1 === step ? 'step' : undefined}
                className={cn(
                  'h-1 flex-1 rounded-full',
                  index + 1 <= step ? 'bg-foreground' : 'bg-muted',
                )}
              >
                <span className="sr-only">{name}</span>
              </li>
            ))}
          </ol>
        </nav>
      </div>

      <section className="flex flex-col gap-6">
        <div className="flex flex-col gap-1">
          <h2 className="text-lg font-semibold">{heading}</h2>
          <p className="text-sm text-muted-foreground">{subtitle}</p>
        </div>
        {children}
      </section>

      <div className="mt-8 flex items-center justify-between gap-4">
        {backTo ? (
          <Link to={backTo} className={backClass}>
            Back
          </Link>
        ) : (
          <Button type="button" variant="ghost" onClick={onBack}>
            Back
          </Button>
        )}
        <div className="flex flex-col items-end gap-1">
          <div className="flex items-center gap-2">
            {extraAction}
            <Button type="submit" size="lg" disabled={nextDisabled}>
              {nextLabel}
            </Button>
          </div>
          {nextNote ? (
            <span className="text-xs text-muted-foreground">{nextNote}</span>
          ) : null}
          {note}
        </div>
      </div>
    </div>
  )
}
