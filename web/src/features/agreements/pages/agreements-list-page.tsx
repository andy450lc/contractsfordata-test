import { ArrowUpDownIcon, PlusIcon } from 'lucide-react'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { z } from 'zod'

import { Badge } from '@/components/ui/badge'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { formatTimestamp } from '@/lib/dates'
import { cn } from '@/lib/utils'

import { AppHeader } from '../components/app-header'
import { TemplateDeliveryDialog } from '../components/template-delivery-dialog'
import { STEP_COUNT } from '../components/wizard-shell'
import { documentReadiness } from '../readiness'
import { useAgreementsStore, type Draft } from '../store'

const filters = ['all', 'signed', 'awaiting', 'drafts'] as const
const sorts = ['counterparty', 'title', 'status', 'created'] as const

const paramsSchema = z.object({
  status: z.enum(filters).catch('all'),
  sort: z.enum(sorts).catch('created'),
  dir: z.enum(['asc', 'desc']).catch('desc'),
})

const filterLabels: Record<(typeof filters)[number], string> = {
  all: 'All',
  signed: 'Signed',
  awaiting: 'Awaiting',
  drafts: 'Drafts',
}

// counterpartyLabel names the other party the way the list shows it.
function counterpartyLabel(draft: Draft): { primary: string; secondary: string } {
  const party = draft.steps.counterparty
  if (party === undefined) return { primary: 'No developer yet', secondary: '' }
  if (party.mode === 'invite' || party.legalName === '') {
    return { primary: party.email, secondary: '' }
  }
  return { primary: party.legalName, secondary: party.email }
}

// resumeStep is the first step without saved values, or the review step.
function resumeStep(draft: Draft): number {
  const order = [
    'agreement',
    'scope',
    'pricing',
    'environment',
    'technical',
    'delivery',
    'counterparty',
  ] as const
  const index = order.findIndex((key) => draft.steps[key] === undefined)
  return index === -1 ? STEP_COUNT : index + 1
}

function StatTile({
  title,
  subtitle,
  count,
}: {
  title: string
  subtitle: string
  count: number
}) {
  return (
    <div className="flex items-center justify-between rounded-xl bg-card p-4 ring-1 ring-foreground/10">
      <div>
        <p className="font-semibold">{title}</p>
        <p className="text-sm text-muted-foreground">{subtitle}</p>
      </div>
      <p className="text-3xl font-semibold">{count}</p>
    </div>
  )
}

// AgreementsListPage is the signed-in home page. It lists every draft
// with its status and launches new agreements.
export function AgreementsListPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const params = paramsSchema.parse(Object.fromEntries(searchParams))
  const drafts = useAgreementsStore((state) => state.drafts)
  const organization = useAgreementsStore((state) => state.organization)
  const deleteDraft = useAgreementsStore((state) => state.deleteDraft)
  const [pendingDelete, setPendingDelete] = useState<Draft | null>(null)

  const all = Object.values(drafts)
  const visible = params.status === 'all' || params.status === 'drafts' ? all : []
  const sorted = [...visible].sort((a, b) => {
    const direction = params.dir === 'asc' ? 1 : -1
    switch (params.sort) {
      case 'counterparty':
        return (
          counterpartyLabel(a).primary.localeCompare(counterpartyLabel(b).primary) *
          direction
        )
      case 'title':
        return (
          (a.steps.agreement?.title ?? '').localeCompare(b.steps.agreement?.title ?? '') *
          direction
        )
      case 'status':
        return 0
      default:
        return (a.createdAt - b.createdAt) * direction
    }
  })

  function setParam(key: 'status' | 'sort', value: string) {
    const next = new URLSearchParams(searchParams)
    if (key === 'sort') {
      const dir = params.sort === value && params.dir === 'desc' ? 'asc' : 'desc'
      next.set('dir', dir)
    }
    next.set(key, value)
    setSearchParams(next)
  }

  return (
    <div className="min-h-svh bg-muted/40">
      <AppHeader />
      <main className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">My agreements</h1>
            <p className="text-sm text-muted-foreground">
              Every agreement you&apos;ve sent and its current signature status.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <Link to="/organization" className={buttonVariants({ variant: 'outline' })}>
              Organization info
            </Link>
            <Link to="/agreements/new" className={buttonVariants({ variant: 'default' })}>
              <PlusIcon data-icon="inline-start" />
              New agreement
            </Link>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-3">
          <StatTile title="Signed" subtitle="Fully executed" count={0} />
          <StatTile title="Awaiting signature" subtitle="In-progress" count={0} />
          <StatTile title="Drafts" subtitle="Not yet sent" count={all.length} />
        </div>

        <div className="flex gap-2" role="group" aria-label="Filter by status">
          {filters.map((filter) => (
            <button
              key={filter}
              type="button"
              aria-pressed={params.status === filter}
              onClick={() => setParam('status', filter)}
              className={cn(
                'rounded-full border px-3 py-1 text-sm',
                params.status === filter
                  ? 'border-foreground bg-foreground text-background'
                  : 'bg-card hover:bg-muted',
              )}
            >
              {filterLabels[filter]}
            </button>
          ))}
        </div>

        <div className="overflow-x-auto rounded-xl bg-card ring-1 ring-foreground/10">
          <table className="w-full text-sm">
            <thead className="text-left text-xs text-muted-foreground">
              <tr>
                {(
                  [
                    ['counterparty', 'Developer'],
                    ['title', 'Title'],
                    ['status', 'Status'],
                    ['created', 'Date created'],
                  ] as const
                ).map(([key, label]) => (
                  <th key={key} className="px-4 py-3 font-medium">
                    <button
                      type="button"
                      onClick={() => setParam('sort', key)}
                      className="inline-flex items-center gap-1 hover:text-foreground"
                    >
                      {label}
                      <ArrowUpDownIcon className="size-3" />
                    </button>
                  </th>
                ))}
                <th className="px-4 py-3">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((draft) => {
                const party = counterpartyLabel(draft)
                return (
                  <tr key={draft.id} className="border-t">
                    <td className="px-4 py-3">
                      <Link
                        to={`/agreements/${draft.id}/edit?step=${resumeStep(draft)}`}
                        className="font-medium hover:underline"
                      >
                        {party.primary}
                      </Link>
                      {party.secondary !== '' ? (
                        <p className="text-muted-foreground">{party.secondary}</p>
                      ) : null}
                    </td>
                    <td className="px-4 py-3">
                      {draft.steps.agreement
                        ? `SOW ${draft.steps.agreement.sowNumber} — ${draft.steps.agreement.title}`
                        : ''}
                    </td>
                    <td className="px-4 py-3">
                      <Badge variant="secondary">Draft</Badge>
                    </td>
                    <td className="px-4 py-3">{formatTimestamp(draft.createdAt)}</td>
                    <td className="px-4 py-3 text-right">
                      {documentReadiness(draft, organization).ready ? (
                        <TemplateDeliveryDialog
                          draft={draft}
                          organization={organization}
                          trigger={
                            <Button type="button" variant="ghost" size="sm">
                              Get Word document
                            </Button>
                          }
                        />
                      ) : null}
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => setPendingDelete(draft)}
                      >
                        Delete
                      </Button>
                    </td>
                  </tr>
                )
              })}
              {sorted.length === 0 ? (
                <tr className="border-t">
                  <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
                    Nothing here yet.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </main>

      <Dialog
        open={pendingDelete !== null}
        onOpenChange={(open) => !open && setPendingDelete(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete this draft?</DialogTitle>
            <DialogDescription>
              {pendingDelete
                ? `"${pendingDelete.steps.agreement?.title ?? 'Untitled'}" and every value entered in it will be removed permanently.`
                : ''}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setPendingDelete(null)}>
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={() => {
                if (pendingDelete) deleteDraft(pendingDelete.id)
                setPendingDelete(null)
              }}
            >
              Delete draft
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
