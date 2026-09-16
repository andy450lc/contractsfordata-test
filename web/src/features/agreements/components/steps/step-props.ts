// StepProps is the contract every wizard step form shares. Next hands
// back validated values. Back hands back whatever is typed so the page
// can keep it for the session.
export interface StepProps<T> {
  step: number
  defaultValues: T
  onNext: (values: T) => void
  onBack: (values: T) => void
  backTo?: string
}
