import { currencyName, formatRate, formatRateDate } from '../utils/format.js'
import { CloseIcon } from './Icons.jsx'

function CardBody({ pair, quote, loading, failed }) {
  if (quote) {
    return (
      <>
        <p className="rate">{formatRate(quote.rate)}</p>
        <p className="inverse">
          1 {pair.quote} = {formatRate(1 / quote.rate)} {pair.base}
        </p>
        <p className="rate-date">
          {failed ? <span className="tag">Not updated</span> : null}
          Rate date {formatRateDate(quote.date)}
        </p>
      </>
    )
  }

  if (loading) {
    return (
      <div className="skeleton-group" aria-label="Loading rate">
        <div className="skeleton skeleton-rate" />
        <div className="skeleton skeleton-line" />
        <div className="skeleton skeleton-line short" />
      </div>
    )
  }

  return <p className="card-unavailable">{failed ? 'Rate unavailable' : 'No rate returned'}</p>
}

export default function PairCard({ pair, quote, loading, failed, onRemove }) {
  const label = `${pair.base} to ${pair.quote}`

  return (
    <article className="card">
      <div className="card-top">
        <div>
          <h2 className="pair-code">
            {pair.base}
            <span className="pair-separator"> / </span>
            {pair.quote}
          </h2>
          <p className="pair-names">
            {currencyName(pair.base)} to {currencyName(pair.quote)}
          </p>
        </div>
        <button
          type="button"
          className="icon-button"
          onClick={() => onRemove(pair)}
          aria-label={`Remove ${label}`}
          title={`Remove ${label}`}
        >
          <CloseIcon />
        </button>
      </div>
      <CardBody pair={pair} quote={quote} loading={loading} failed={failed} />
    </article>
  )
}
