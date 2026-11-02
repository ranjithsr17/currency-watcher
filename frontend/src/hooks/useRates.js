import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { fetchRates } from '../api/rates.js'
import { groupByBase } from '../utils/pairs.js'

// useRates loads rates for the tracked pairs, with one request per base currency.
// A failed base keeps its last known rates and doesn't affect the other bases.
export function useRates(pairs) {
  const [quotes, setQuotes] = useState({})
  const [errors, setErrors] = useState({})
  const [loading, setLoading] = useState(pairs.length > 0)
  const [lastUpdated, setLastUpdated] = useState(null)
  const controllerRef = useRef(null)

  const groups = useMemo(() => groupByBase(pairs), [pairs])
  const groupsKey = JSON.stringify(groups)

  const refresh = useCallback(async () => {
    // Only the newest request may update state, so a slow response can't
    // overwrite the result of a later one.
    controllerRef.current?.abort()
    const controller = new AbortController()
    controllerRef.current = controller

    const currentGroups = JSON.parse(groupsKey)
    if (currentGroups.length === 0) {
      setErrors({})
      setLoading(false)
      return
    }

    setLoading(true)
    const results = await Promise.allSettled(
      currentGroups.map(({ base, targets }) => fetchRates(base, targets, controller.signal)),
    )
    if (controller.signal.aborted) {
      return
    }

    const nextQuotes = {}
    const nextErrors = {}
    results.forEach((result, index) => {
      const { base } = currentGroups[index]
      if (result.status === 'fulfilled') {
        nextQuotes[base] = result.value
      } else {
        nextErrors[base] = result.reason.message
      }
    })

    setQuotes((current) => ({ ...current, ...nextQuotes }))
    setErrors(nextErrors)
    if (Object.keys(nextQuotes).length > 0) {
      setLastUpdated(new Date())
    }
    setLoading(false)
  }, [groupsKey])

  useEffect(() => {
    refresh()
    return () => controllerRef.current?.abort()
  }, [refresh])

  return { quotes, errors, loading, lastUpdated, refresh }
}
