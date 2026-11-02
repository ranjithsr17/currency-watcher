// Package rates serves exchange rates through a cache of complete rate tables.
package rates

import (
	"context"
	"sync"
	"time"

	"currencywatcher/internal/cache"
	"currencywatcher/internal/frankfurter"
)

// Fetcher retrieves the complete rate table for a base currency.
type Fetcher interface {
	Rates(ctx context.Context, base string) ([]frankfurter.Rate, error)
}

// Quote is the rate for one target currency and the date it was published.
type Quote struct {
	Rate float64
	Date string
}

// Table maps target currency codes to quotes. Tables are shared between requests
// and must not be modified.
type Table map[string]Quote

// Result is a rate table and where it came from.
type Result struct {
	Base      string
	Rates     Table
	FetchedAt time.Time
	ExpiresAt time.Time
	Cached    bool
}

type call struct {
	done   chan struct{}
	result Result
	err    error
}

// Service returns rate tables, fetching one only when it is not cached.
type Service struct {
	fetcher Fetcher
	cache   *cache.Cache[Table]

	mu       sync.Mutex
	inflight map[string]*call
}

// NewService returns a Service backed by fetcher and tables.
func NewService(fetcher Fetcher, tables *cache.Cache[Table]) *Service {
	return &Service{
		fetcher:  fetcher,
		cache:    tables,
		inflight: make(map[string]*call),
	}
}

// Get returns the complete rate table for base.
//
// Concurrent misses for the same base share one upstream request. That request
// runs without the caller's cancellation, so one client disconnecting cannot
// fail the others waiting on it. Errors are never cached.
func (s *Service) Get(ctx context.Context, base string) (Result, error) {
	if entry, ok := s.cache.Get(base); ok {
		return cachedResult(base, entry), nil
	}

	s.mu.Lock()
	c, ok := s.inflight[base]
	if !ok {
		// A fetch may have finished between the cache miss and taking the lock.
		if entry, ok := s.cache.Get(base); ok {
			s.mu.Unlock()
			return cachedResult(base, entry), nil
		}
		c = &call{done: make(chan struct{})}
		s.inflight[base] = c
		go s.fetch(context.WithoutCancel(ctx), base, c)
	}
	s.mu.Unlock()

	select {
	case <-c.done:
		return c.result, c.err
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
}

func (s *Service) fetch(ctx context.Context, base string, c *call) {
	defer close(c.done)

	rows, err := s.fetcher.Rates(ctx, base)
	if err != nil {
		c.err = err
	} else {
		table := make(Table, len(rows))
		for _, row := range rows {
			table[row.Quote] = Quote{Rate: row.Rate, Date: row.Date}
		}
		entry := s.cache.Set(base, table)
		c.result = Result{
			Base:      base,
			Rates:     table,
			FetchedAt: entry.StoredAt,
			ExpiresAt: entry.ExpiresAt,
		}
	}

	s.mu.Lock()
	delete(s.inflight, base)
	s.mu.Unlock()
}

func cachedResult(base string, entry cache.Entry[Table]) Result {
	return Result{
		Base:      base,
		Rates:     entry.Value,
		FetchedAt: entry.StoredAt,
		ExpiresAt: entry.ExpiresAt,
		Cached:    true,
	}
}
