import { AlertIcon } from './Icons.jsx'

export default function ErrorBanner({ messages, loading, onRetry }) {
  return (
    <div className="banner" role="alert">
      <AlertIcon />
      <div className="banner-text">
        <strong>Some rates couldn't be updated</strong>
        {messages.map((message) => (
          <span key={message}>{message}</span>
        ))}
      </div>
      <button type="button" className="button button-secondary" onClick={onRetry} disabled={loading}>
        Try again
      </button>
    </div>
  )
}
