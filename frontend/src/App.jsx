import { useState } from 'react'
import AddPairDialog from './components/AddPairDialog.jsx'
import EmptyState from './components/EmptyState.jsx'
import ErrorBanner from './components/ErrorBanner.jsx'
import Header from './components/Header.jsx'
import PairCard from './components/PairCard.jsx'
import StatusBar from './components/StatusBar.jsx'
import { useRates } from './hooks/useRates.js'
import { useTrackedPairs } from './hooks/useTrackedPairs.js'
import { pairId } from './utils/pairs.js'

export default function App() {
  const { pairs, addPair, removePair } = useTrackedPairs()
  const { quotes, errors, loading, lastUpdated, refresh } = useRates(pairs)
  const [dialogOpen, setDialogOpen] = useState(false)

  const errorMessages = [...new Set(Object.values(errors))]
  const trackedQuotes = [...new Set(pairs.map((pair) => pair.base))]
    .map((base) => quotes[base])
    .filter(Boolean)

  return (
    <div className="app">
      <Header
        loading={loading}
        canRefresh={pairs.length > 0}
        onRefresh={refresh}
        onAdd={() => setDialogOpen(true)}
      />

      <main className="content">
        <div className="page-heading">
          <h1>Tracked pairs</h1>
          {pairs.length > 0 && <StatusBar lastUpdated={lastUpdated} quotes={trackedQuotes} />}
        </div>

        {errorMessages.length > 0 && (
          <ErrorBanner messages={errorMessages} loading={loading} onRetry={refresh} />
        )}

        {pairs.length === 0 ? (
          <EmptyState onAdd={() => setDialogOpen(true)} />
        ) : (
          <ul className="pair-grid" aria-busy={loading}>
            {pairs.map((pair) => (
              <li key={pairId(pair)}>
                <PairCard
                  pair={pair}
                  quote={quotes[pair.base]?.rates[pair.quote]}
                  loading={loading}
                  failed={Boolean(errors[pair.base])}
                  onRemove={removePair}
                />
              </li>
            ))}
          </ul>
        )}
      </main>

      <footer className="footer">
        Exchange rates from the{' '}
        <a href="https://frankfurter.dev" target="_blank" rel="noreferrer">
          Frankfurter API
        </a>
        . For information only.
      </footer>

      {dialogOpen && (
        <AddPairDialog
          pairs={pairs}
          onAdd={addPair}
          onClose={() => setDialogOpen(false)}
        />
      )}
    </div>
  )
}
