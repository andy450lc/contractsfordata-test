import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, useWatch } from 'react-hook-form'
import { Link, useNavigate } from 'react-router-dom'

import { Button, buttonVariants } from '@/components/ui/button'

import { AppHeader } from '../components/app-header'
import { TextField } from '../components/field'
import { emptyOrganization } from '../defaults'
import { organizationSchema, type Organization } from '../schemas'
import { useAgreementsStore } from '../store'

// OrganizationPage edits the sender's own party details, which every
// draft's review step reads.
export function OrganizationPage() {
  const navigate = useNavigate()
  const organization = useAgreementsStore((state) => state.organization)
  const setOrganization = useAgreementsStore((state) => state.setOrganization)
  const form = useForm<Organization>({
    resolver: zodResolver(organizationSchema),
    defaultValues: organization ?? emptyOrganization,
  })
  const { control } = form
  const legalName = useWatch({ control, name: 'legalName' })
  const incorporationPlace = useWatch({ control, name: 'incorporationPlace' })

  function fillDerived() {
    if (form.getValues('shortName') === '' && legalName.trim() !== '') {
      form.setValue('shortName', legalName.trim().split(/\s+/)[0] ?? '')
    }
    if (form.getValues('governingLaw') === '' && incorporationPlace.trim() !== '') {
      form.setValue('governingLaw', incorporationPlace.trim())
    }
  }

  function onSubmit(values: Organization) {
    setOrganization(values)
    void navigate('/agreements')
  }

  return (
    <div className="min-h-svh bg-muted/40">
      <AppHeader />
      <main className="px-4 py-8">
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          onBlur={fillDerived}
          noValidate
          className="mx-auto flex w-full max-w-2xl flex-col gap-5 rounded-xl bg-card p-6 shadow-sm ring-1 ring-foreground/10 sm:p-8"
        >
          <div className="flex flex-col items-center gap-2 text-center">
            <h1 className="text-2xl font-semibold tracking-tight">Organization info</h1>
            <p className="max-w-md text-sm text-muted-foreground">
              Your organization is the Customer on every agreement you create. These
              details identify you in the document and its signature block. Agreements you
              have already sent keep the details they were sent with.
            </p>
          </div>
          <TextField
            control={control}
            name="legalName"
            label="Organization legal name"
            required
            help="Your company's full legal name, exactly as registered."
            placeholder="Acme Vision, Inc."
          />
          <div className="grid gap-5 sm:grid-cols-2">
            <TextField
              control={control}
              name="entityType"
              label="Entity type"
              required
              help="What kind of entity you are, for example 'corporation' or 'limited liability company'."
              placeholder="corporation"
            />
            <TextField
              control={control}
              name="incorporationPlace"
              label="Place of incorporation"
              required
              help="The state or country where your company is registered."
              placeholder="Delaware"
            />
          </div>
          <div className="grid gap-5 sm:grid-cols-2">
            <TextField
              control={control}
              name="governingLaw"
              label="Governing law jurisdiction"
              required
              help="Whose law applies to the agreement and where any dispute is heard. Usually the same as your place of incorporation."
              placeholder="Delaware"
            />
            <TextField
              control={control}
              name="shortName"
              label="Short name used in the document"
              required
              help="A short name to refer to you throughout the document, usually the first word of your legal name."
              placeholder="Acme"
            />
          </div>
          <TextField
            control={control}
            name="address"
            label="Organization address"
            required
            multiline
            rows={2}
            help="Your registered or mailing address. Formal notices are sent here."
            placeholder="100 Main Street, Suite 200, Wilmington, Delaware 19801"
          />
          <div className="grid gap-5 sm:grid-cols-2">
            <TextField
              control={control}
              name="signerName"
              label="Signer name"
              required
              help="The person who signs agreements on your behalf."
              placeholder="Jane Doe"
            />
            <TextField
              control={control}
              name="signerTitle"
              label="Signer title"
              required
              help="That person's role, for example 'President' or 'CEO'."
              placeholder="President"
            />
          </div>
          <div className="grid gap-5 sm:grid-cols-2">
            <TextField
              control={control}
              name="signerEmail"
              label="Signer email"
              required
              type="email"
              help="Where we send your own signing link."
              placeholder="jane@example.com"
            />
            <TextField
              control={control}
              name="contactPhone"
              label="Contact phone"
              type="tel"
              help="Optional. Shown to the Developer in case they have questions about the agreement."
              placeholder="(415) 555-0123"
            />
          </div>
          <div className="mt-2 flex items-center justify-between">
            <Link to="/agreements" className={buttonVariants({ variant: 'ghost' })}>
              Cancel
            </Link>
            <Button type="submit" size="lg">
              Save changes
            </Button>
          </div>
        </form>
      </main>
    </div>
  )
}
