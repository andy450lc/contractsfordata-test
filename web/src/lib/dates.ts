// API timestamps are Unix milliseconds. Display
// formatting for the whole app goes through this module.

const dateTimeFormat = new Intl.DateTimeFormat('en-US', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'UTC',
})

const longDateFormat = new Intl.DateTimeFormat('en-US', {
  dateStyle: 'long',
  timeZone: 'UTC',
})

// formatTimestamp renders a Unix-millisecond
// timestamp as a date and time in UTC.
export function formatTimestamp(millis: number): string {
  return dateTimeFormat.format(new Date(millis))
}

// formatLongDate renders an ISO calendar date such as 2026-09-06 as
// "September 6, 2026". The date is read as UTC. An empty value renders
// as an empty string.
export function formatLongDate(isoDate: string): string {
  if (isoDate === '') return ''
  const date = new Date(`${isoDate}T00:00:00Z`)
  if (Number.isNaN(date.getTime())) return ''
  return longDateFormat.format(date)
}
