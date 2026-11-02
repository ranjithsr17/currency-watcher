const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080').replace(/\/+$/, '')

// fetchRates returns rates for base. Omitting targets returns every available rate.
export async function fetchRates(base, targets, signal) {
  const params = new URLSearchParams({ base })
  if (targets) {
    params.set('targets', targets.join(','))
  }

  let response
  try {
    response = await fetch(`${API_BASE_URL}/api/rates?${params}`, { signal })
  } catch (error) {
    if (error.name === 'AbortError') {
      throw error
    }
    throw new Error(`Could not reach the backend at ${API_BASE_URL}.`)
  }

  const body = await response.json().catch(() => null)
  if (!response.ok) {
    const message = body?.error?.message
    throw new Error(
      message
        ? message.charAt(0).toUpperCase() + message.slice(1)
        : `The backend returned an error (${response.status}).`,
    )
  }
  if (!body?.rates) {
    throw new Error('The backend returned an unexpected response.')
  }
  return body
}
