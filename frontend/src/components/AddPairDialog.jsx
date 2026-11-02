import { useEffect, useRef, useState } from 'react'
import { useCurrencies } from '../hooks/useCurrencies.js'
import { currencyName } from '../utils/format.js'
import { pairId } from '../utils/pairs.js'
import { CloseIcon, SwapIcon } from './Icons.jsx'

function CurrencySelect({ id, label, value, codes, onChange }) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <select id={id} value={value} onChange={(event) => onChange(event.target.value)}>
        {codes.map((code) => (
          <option key={code} value={code}>
            {code} – {currencyName(code)}
          </option>
        ))}
      </select>
    </div>
  )
}

function validationMessage(pair, pairs) {
  if (pair.base === pair.quote) {
    return 'Choose two different currencies.'
  }
  if (pairs.some((existing) => pairId(existing) === pairId(pair))) {
    return `You're already tracking ${pair.base} / ${pair.quote}.`
  }
  return null
}

export default function AddPairDialog({ pairs, onAdd, onClose }) {
  const dialogRef = useRef(null)
  const { codes, loading, error, retry } = useCurrencies()
  const [base, setBase] = useState('USD')
  const [quote, setQuote] = useState('EUR')

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog.open) {
      dialog.showModal()
    }
    return () => dialog.close()
  }, [])

  const pair = { base, quote }
  const message = validationMessage(pair, pairs)

  function handleSubmit(event) {
    event.preventDefault()
    if (!message) {
      onAdd(pair)
      onClose()
    }
  }

  function handleSwap() {
    setBase(quote)
    setQuote(base)
  }

  return (
    <dialog
      ref={dialogRef}
      className="dialog"
      aria-labelledby="add-pair-title"
      onClose={() => {
        // Close events are queued. StrictMode reopens the dialog after its effect
        // cleanup, so an event that arrives while it's open is out of date.
        if (!dialogRef.current?.open) {
          onClose()
        }
      }}
      onClick={(event) => {
        // A click on the dialog element itself is a click on the backdrop.
        if (event.target === dialogRef.current) {
          onClose()
        }
      }}
    >
      <form className="dialog-body" onSubmit={handleSubmit}>
        <div className="dialog-header">
          <h2 id="add-pair-title">Add currency pair</h2>
          <button type="button" className="icon-button" onClick={onClose} aria-label="Close">
            <CloseIcon />
          </button>
        </div>

        {loading && <p className="dialog-note">Loading currencies…</p>}

        {error && (
          <div className="dialog-error" role="alert">
            <p>{error}</p>
            <button type="button" className="button button-secondary" onClick={retry}>
              Try again
            </button>
          </div>
        )}

        {!loading && !error && (
          <>
            <div className="dialog-fields">
              <CurrencySelect id="pair-base" label="From" value={base} codes={codes} onChange={setBase} />
              <button
                type="button"
                className="icon-button swap-button"
                onClick={handleSwap}
                aria-label="Swap currencies"
                title="Swap currencies"
              >
                <SwapIcon />
              </button>
              <CurrencySelect id="pair-quote" label="To" value={quote} codes={codes} onChange={setQuote} />
            </div>
            <p className={message ? 'dialog-validation' : 'dialog-note'} role="status">
              {message ?? `Tracks 1 ${base} in ${quote}.`}
            </p>
          </>
        )}

        <div className="dialog-actions">
          <button type="button" className="button button-secondary" onClick={onClose}>
            Cancel
          </button>
          <button
            type="submit"
            className="button button-primary"
            disabled={loading || Boolean(error) || Boolean(message)}
          >
            Add pair
          </button>
        </div>
      </form>
    </dialog>
  )
}
