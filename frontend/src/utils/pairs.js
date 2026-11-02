const STORAGE_KEY = 'currency-watcher:pairs'
const CODE_PATTERN = /^[A-Z]{3}$/

const DEFAULT_PAIRS = [
  { base: 'USD', quote: 'EUR' },
  { base: 'USD', quote: 'SGD' },
  { base: 'EUR', quote: 'GBP' },
]

export function pairId({ base, quote }) {
  return `${base}/${quote}`
}

function isValidPair(pair) {
  return (
    CODE_PATTERN.test(pair?.base) &&
    CODE_PATTERN.test(pair?.quote) &&
    pair.base !== pair.quote
  )
}

// loadPairs returns the saved pairs. First-time visitors get a starter set, while
// someone who removed every pair keeps their empty list.
export function loadPairs() {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === null) {
      return DEFAULT_PAIRS
    }

    const parsed = JSON.parse(stored)
    if (!Array.isArray(parsed)) {
      return DEFAULT_PAIRS
    }

    const unique = new Map()
    for (const pair of parsed) {
      if (isValidPair(pair)) {
        unique.set(pairId(pair), { base: pair.base, quote: pair.quote })
      }
    }
    return [...unique.values()]
  } catch {
    return DEFAULT_PAIRS
  }
}

export function savePairs(pairs) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(pairs))
  } catch {
    // Storage can be blocked or full. The list still works for this session.
  }
}

// groupByBase returns one { base, targets } per base so that each base needs only
// one request.
export function groupByBase(pairs) {
  const groups = new Map()
  for (const { base, quote } of pairs) {
    groups.set(base, [...(groups.get(base) ?? []), quote])
  }
  return [...groups].map(([base, targets]) => ({ base, targets }))
}
