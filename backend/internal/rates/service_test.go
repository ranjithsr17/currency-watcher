package rates

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"currencywatcher/internal/cache"
	"currencywatcher/internal/frankfurter"
)

type fakeFetcher struct {
	calls atomic.Int32
	fn    func(ctx context.Context, base string) ([]frankfurter.Rate, error)
}

func (f *fakeFetcher) Rates(ctx context.Context, base string) ([]frankfurter.Rate, error) {
	f.calls.Add(1)
	if f.fn != nil {
		return f.fn(ctx, base)
	}
	return []frankfurter.Rate{{Date: "2026-11-02", Base: base, Quote: "EUR", Rate: 0.9}}, nil
}

func newTestService(fetcher Fetcher) (*Service, *time.Time) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	return NewService(fetcher, cache.New[Table](time.Hour, func() time.Time { return now })), &now
}

func TestGetCachesCompleteTable(t *testing.T) {
	fetcher := &fakeFetcher{}
	service, _ := newTestService(fetcher)

	first, err := service.Get(context.Background(), "USD")
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if first.Cached {
		t.Error("first result reported as cached")
	}
	if want := (Quote{Rate: 0.9, Date: "2026-11-02"}); first.Rates["EUR"] != want {
		t.Errorf("EUR = %+v, want %+v", first.Rates["EUR"], want)
	}

	second, err := service.Get(context.Background(), "USD")
	if err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if !second.Cached {
		t.Error("second result not reported as cached")
	}
	if !second.FetchedAt.Equal(first.FetchedAt) || !second.ExpiresAt.Equal(first.ExpiresAt) {
		t.Error("cached result has different freshness times from the original")
	}
	if got := fetcher.calls.Load(); got != 1 {
		t.Errorf("fetcher called %d times, want 1", got)
	}
}

func TestGetCachesEachBaseSeparately(t *testing.T) {
	fetcher := &fakeFetcher{}
	service, _ := newTestService(fetcher)

	for _, base := range []string{"USD", "EUR", "USD", "EUR"} {
		if _, err := service.Get(context.Background(), base); err != nil {
			t.Fatalf("Get(%s): %v", base, err)
		}
	}
	if got := fetcher.calls.Load(); got != 2 {
		t.Errorf("fetcher called %d times, want 2", got)
	}
}

func TestGetRefetchesAfterTTL(t *testing.T) {
	fetcher := &fakeFetcher{}
	service, now := newTestService(fetcher)

	if _, err := service.Get(context.Background(), "USD"); err != nil {
		t.Fatalf("first Get: %v", err)
	}

	*now = now.Add(time.Hour)

	result, err := service.Get(context.Background(), "USD")
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if result.Cached {
		t.Error("result after expiry reported as cached")
	}
	if got := fetcher.calls.Load(); got != 2 {
		t.Errorf("fetcher called %d times, want 2", got)
	}
}

func TestGetDoesNotCacheErrors(t *testing.T) {
	failing := true
	fetcher := &fakeFetcher{}
	fetcher.fn = func(_ context.Context, base string) ([]frankfurter.Rate, error) {
		if failing {
			return nil, frankfurter.ErrUnavailable
		}
		return []frankfurter.Rate{{Date: "2026-11-02", Base: base, Quote: "EUR", Rate: 0.9}}, nil
	}
	service, _ := newTestService(fetcher)

	if _, err := service.Get(context.Background(), "USD"); !errors.Is(err, frankfurter.ErrUnavailable) {
		t.Fatalf("err = %v, want %v", err, frankfurter.ErrUnavailable)
	}

	failing = false

	if _, err := service.Get(context.Background(), "USD"); err != nil {
		t.Fatalf("Get after recovery: %v", err)
	}
	if got := fetcher.calls.Load(); got != 2 {
		t.Errorf("fetcher called %d times, want 2", got)
	}
}

func TestConcurrentMissesShareOneFetch(t *testing.T) {
	release := make(chan struct{})
	fetcher := &fakeFetcher{}
	fetcher.fn = func(_ context.Context, base string) ([]frankfurter.Rate, error) {
		<-release
		return []frankfurter.Rate{{Date: "2026-11-02", Base: base, Quote: "EUR", Rate: 0.9}}, nil
	}
	service, _ := newTestService(fetcher)

	const callers = 50
	errs := make(chan error, callers)
	var started, finished sync.WaitGroup
	for range callers {
		started.Add(1)
		finished.Add(1)
		go func() {
			defer finished.Done()
			started.Done()
			_, err := service.Get(context.Background(), "USD")
			errs <- err
		}()
	}
	started.Wait()
	time.Sleep(20 * time.Millisecond)
	close(release)
	finished.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Errorf("Get: %v", err)
		}
	}
	if got := fetcher.calls.Load(); got != 1 {
		t.Errorf("fetcher called %d times, want 1", got)
	}
}

func TestCancelledCallerDoesNotFailOtherWaiters(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	fetcher := &fakeFetcher{}
	fetcher.fn = func(ctx context.Context, base string) ([]frankfurter.Rate, error) {
		close(entered)
		<-release
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return []frankfurter.Rate{{Date: "2026-11-02", Base: base, Quote: "EUR", Rate: 0.9}}, nil
	}
	service, _ := newTestService(fetcher)

	ctx, cancel := context.WithCancel(context.Background())
	firstErr := make(chan error, 1)
	go func() {
		_, err := service.Get(ctx, "USD")
		firstErr <- err
	}()
	<-entered

	secondErr := make(chan error, 1)
	go func() {
		_, err := service.Get(context.Background(), "USD")
		secondErr <- err
	}()

	cancel()
	if err := <-firstErr; !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled caller err = %v, want %v", err, context.Canceled)
	}

	close(release)
	if err := <-secondErr; err != nil {
		t.Errorf("waiting caller err = %v, want nil", err)
	}
	if got := fetcher.calls.Load(); got != 1 {
		t.Errorf("fetcher called %d times, want 1", got)
	}
}
