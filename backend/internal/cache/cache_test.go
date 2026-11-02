package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestGetMissingKey(t *testing.T) {
	c := New[string](time.Hour, newFakeClock().Now)

	if _, ok := c.Get("USD"); ok {
		t.Fatal("Get returned a value for a key that was never set")
	}
}

func TestSetRecordsFreshnessTimes(t *testing.T) {
	clock := newFakeClock()
	start := clock.Now()
	c := New[string](time.Hour, clock.Now)

	c.Set("USD", "rates")

	entry, ok := c.Get("USD")
	if !ok {
		t.Fatal("Get missed a fresh entry")
	}
	if entry.Value != "rates" {
		t.Errorf("Value = %q, want %q", entry.Value, "rates")
	}
	if !entry.StoredAt.Equal(start) {
		t.Errorf("StoredAt = %v, want %v", entry.StoredAt, start)
	}
	if want := start.Add(time.Hour); !entry.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", entry.ExpiresAt, want)
	}
}

func TestExpiryBoundary(t *testing.T) {
	clock := newFakeClock()
	c := New[string](time.Hour, clock.Now)
	c.Set("USD", "rates")

	clock.Advance(time.Hour - time.Nanosecond)
	if _, ok := c.Get("USD"); !ok {
		t.Fatal("entry expired before its TTL elapsed")
	}

	clock.Advance(time.Nanosecond)
	if _, ok := c.Get("USD"); ok {
		t.Fatal("entry still returned once its TTL elapsed")
	}
}

func TestSetReplacesEntryAndRestartsTTL(t *testing.T) {
	clock := newFakeClock()
	c := New[string](time.Hour, clock.Now)
	c.Set("USD", "old")

	clock.Advance(59 * time.Minute)
	c.Set("USD", "new")
	clock.Advance(59 * time.Minute)

	entry, ok := c.Get("USD")
	if !ok {
		t.Fatal("replaced entry expired on the original TTL")
	}
	if entry.Value != "new" {
		t.Errorf("Value = %q, want %q", entry.Value, "new")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	c := New[string](time.Hour, newFakeClock().Now)
	c.Set("USD", "dollars")
	c.Set("EUR", "euros")

	entry, ok := c.Get("EUR")
	if !ok || entry.Value != "euros" {
		t.Errorf("Get(EUR) = %q, %v; want euros, true", entry.Value, ok)
	}
}

// TestConcurrentAccess is most useful under `go test -race`.
func TestConcurrentAccess(t *testing.T) {
	c := New[int](time.Hour, newFakeClock().Now)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i%5)
			for j := range 100 {
				c.Set(key, j)
				c.Get(key)
			}
		}()
	}
	wg.Wait()

	for i := range 5 {
		if _, ok := c.Get(fmt.Sprintf("key-%d", i)); !ok {
			t.Errorf("key-%d missing after concurrent writes", i)
		}
	}
}
