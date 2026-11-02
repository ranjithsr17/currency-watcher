const veryLargeRate = new Intl.NumberFormat(undefined, {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})
const largeRate = new Intl.NumberFormat(undefined, {
  minimumFractionDigits: 4,
  maximumFractionDigits: 4,
})
const smallRate = new Intl.NumberFormat(undefined, { maximumSignificantDigits: 5 })
const time = new Intl.DateTimeFormat(undefined, { timeStyle: 'medium' })
const shortTime = new Intl.DateTimeFormat(undefined, { timeStyle: 'short' })
const utcDate = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeZone: 'UTC' })
const currencyNames = new Intl.DisplayNames(undefined, { type: 'currency', fallback: 'code' })

export function formatRate(rate) {
  if (rate >= 1000) {
    return veryLargeRate.format(rate)
  }
  return (rate >= 1 ? largeRate : smallRate).format(rate)
}

export function formatTime(date) {
  return time.format(date)
}

export function formatShortTime(date) {
  return shortTime.format(date)
}

// formatRateDate formats a YYYY-MM-DD string. It parses and prints in UTC so that
// the day does not shift in time zones behind UTC.
export function formatRateDate(isoDate) {
  return utcDate.format(new Date(`${isoDate}T00:00:00Z`))
}

export function currencyName(code) {
  return currencyNames.of(code)
}
