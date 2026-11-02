import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchRates } from '../api/rates.js'

// Any base returns the full list of quoted currencies, and the backend caches it.
const REFERENCE_BASE = 'USD'

// useCurrencies loads the currency codes that can be used to build a pair.
export function useCurrencies() {
  const [codes, setCodes] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const controllerRef = useRef(null)

  const load = useCallback(async () => {
    controllerRef.current?.abort()
    const controller = new AbortController()
    controllerRef.current = controller

    setLoading(true)
    setError(null)
    try {
      const { rates } = await fetchRates(REFERENCE_BASE, null, controller.signal)
      setCodes(Object.keys(rates).sort())
      setLoading(false)
    } catch (err) {
      if (err.name !== 'AbortError') {
        setError(err.message)
        setLoading(false)
      }
    }
  }, [])

  useEffect(() => {
    load()
    return () => controllerRef.current?.abort()
  }, [load])

  return { codes, loading, error, retry: load }
}
