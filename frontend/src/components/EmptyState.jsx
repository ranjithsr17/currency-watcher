import { ChartIcon, PlusIcon } from './Icons.jsx'

export default function EmptyState({ onAdd }) {
  return (
    <section className="empty">
      <span className="empty-icon">
        <ChartIcon />
      </span>
      <h2>No currency pairs yet</h2>
      <p>Add a pair to start watching its exchange rate.</p>
      <button type="button" className="button button-primary" onClick={onAdd}>
        <PlusIcon />
        Add your first pair
      </button>
    </section>
  )
}
