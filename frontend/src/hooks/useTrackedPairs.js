import { useCallback, useEffect, useState } from 'react'
import { loadPairs, pairId, savePairs } from '../utils/pairs.js'

export function useTrackedPairs() {
  const [pairs, setPairs] = useState(loadPairs)

  useEffect(() => {
    savePairs(pairs)
  }, [pairs])

  const addPair = useCallback((pair) => {
    setPairs((current) =>
      current.some((existing) => pairId(existing) === pairId(pair)) ? current : [...current, pair],
    )
  }, [])

  const removePair = useCallback((pair) => {
    setPairs((current) => current.filter((existing) => pairId(existing) !== pairId(pair)))
  }, [])

  return { pairs, addPair, removePair }
}
