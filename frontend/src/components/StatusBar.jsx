import { formatShortTime, formatTime } from '../utils/format.js'

// StatusBar separates when the dashboard last heard from the backend from when the
// backend last fetched from Frankfurter. Within the one-hour cache window, a refresh
// changes only the first of these.
export default function StatusBar({ lastUpdated, quotes }) {
  const fetchTimes = quotes.map((quote) => new Date(quote.fetchedAt).getTime())
  const expiryTimes = quotes.map((quote) => new Date(quote.expiresAt).getTime())

  return (
    <p className="status" role="status">
      {lastUpdated ? (
        <span>
          Last updated <strong>{formatTime(lastUpdated)}</strong>
        </span>
      ) : (
        <span>Loading latest rates…</span>
      )}
      {quotes.length > 0 && (
        <span className="status-detail">
          Fetched from source at {formatShortTime(Math.min(...fetchTimes))}. New data available
          from {formatShortTime(Math.min(...expiryTimes))}.
        </span>
      )}
    </p>
  )
}
