import { ChartIcon, PlusIcon, RefreshIcon } from './Icons.jsx'

export default function Header({ loading, canRefresh, onRefresh, onAdd }) {
  return (
    <header className="header">
      <div className="header-inner">
        <div className="brand">
          <span className="brand-mark">
            <ChartIcon />
          </span>
          Currency Watcher
        </div>
        <div className="header-actions">
          <button
            type="button"
            className="button button-secondary"
            onClick={onRefresh}
            disabled={loading || !canRefresh}
          >
            <RefreshIcon className={loading ? 'spin' : undefined} />
            {loading ? 'Refreshing' : 'Refresh'}
          </button>
          <button type="button" className="button button-primary" onClick={onAdd}>
            <PlusIcon />
            Add pair
          </button>
        </div>
      </div>
    </header>
  )
}
