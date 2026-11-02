package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"currencywatcher/internal/cache"
	"currencywatcher/internal/frankfurter"
	"currencywatcher/internal/rates"
)

const allowedOrigin = "http://localhost:5173"

var (
	fetchedAt = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	testTable = rates.Table{
		"EUR": {Rate: 0.89471, Date: "2026-11-02"},
		"SGD": {Rate: 1.2915, Date: "2026-11-02"},
		"AFN": {Rate: 64.93, Date: "2026-11-01"},
	}
)

type fakeProvider struct {
	result rates.Result
	err    error
	calls  int
}

func (p *fakeProvider) Get(_ context.Context, base string) (rates.Result, error) {
	p.calls++
	if p.err != nil {
		return rates.Result{}, p.err
	}
	result := p.result
	result.Base = base
	return result, nil
}

func successProvider() *fakeProvider {
	return &fakeProvider{result: rates.Result{
		Rates:     testTable,
		FetchedAt: fetchedAt,
		ExpiresAt: fetchedAt.Add(time.Hour),
	}}
}

func newTestHandler(provider RatesProvider) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(provider, logger, []string{allowedOrigin})
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var body T
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestHealth(t *testing.T) {
	provider := successProvider()
	rec := get(t, newTestHandler(provider), "/api/health")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := decode[healthResponse](t, rec); got.Status != "ok" {
		t.Errorf("status field = %q, want ok", got.Status)
	}
	if provider.calls != 0 {
		t.Error("health check called the exchange rate provider")
	}
}

func TestRatesFiltersToRequestedTargets(t *testing.T) {
	rec := get(t, newTestHandler(successProvider()), "/api/rates?base=USD&targets=EUR,SGD")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}

	body := decode[ratesResponse](t, rec)
	if body.Base != "USD" {
		t.Errorf("base = %q, want USD", body.Base)
	}
	if len(body.Rates) != 2 {
		t.Fatalf("got %d rates, want 2: %+v", len(body.Rates), body.Rates)
	}
	if want := (rateResponse{Rate: 0.89471, Date: "2026-11-02"}); body.Rates["EUR"] != want {
		t.Errorf("EUR = %+v, want %+v", body.Rates["EUR"], want)
	}
	if !body.FetchedAt.Equal(fetchedAt) || !body.ExpiresAt.Equal(fetchedAt.Add(time.Hour)) {
		t.Errorf("fetchedAt = %v, expiresAt = %v", body.FetchedAt, body.ExpiresAt)
	}
}

func TestRatesTimestampsAreUTCSeconds(t *testing.T) {
	provider := successProvider()
	provider.result.FetchedAt = time.Date(2026, 11, 2, 15, 30, 0, 123, time.FixedZone("IST", 5*60*60+30*60))
	rec := get(t, newTestHandler(provider), "/api/rates?base=USD&targets=EUR")

	if want := `"fetchedAt":"2026-11-02T10:00:00Z"`; !strings.Contains(rec.Body.String(), want) {
		t.Errorf("body = %s, want it to contain %s", rec.Body, want)
	}
}

func TestRatesWithoutTargetsReturnsAllRates(t *testing.T) {
	rec := get(t, newTestHandler(successProvider()), "/api/rates?base=USD")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if body := decode[ratesResponse](t, rec); len(body.Rates) != len(testTable) {
		t.Errorf("got %d rates, want %d", len(body.Rates), len(testTable))
	}
}

func TestRatesNormalizesCodesAndRemovesDuplicates(t *testing.T) {
	rec := get(t, newTestHandler(successProvider()), "/api/rates?base=usd&targets=eur,%20EUR,sgd")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	body := decode[ratesResponse](t, rec)
	if body.Base != "USD" {
		t.Errorf("base = %q, want USD", body.Base)
	}
	if len(body.Rates) != 2 {
		t.Errorf("got %d rates, want 2: %+v", len(body.Rates), body.Rates)
	}
}

func TestRatesValidation(t *testing.T) {
	tooMany := strings.TrimSuffix(strings.Repeat("EUR,", maxTargets+1), ",")

	tests := []struct {
		name     string
		query    string
		wantCode string
	}{
		{"missing base", "targets=EUR", "invalid_base"},
		{"empty base", "base=&targets=EUR", "invalid_base"},
		{"short base", "base=US&targets=EUR", "invalid_base"},
		{"long base", "base=USDX&targets=EUR", "invalid_base"},
		{"numeric base", "base=U5D&targets=EUR", "invalid_base"},
		{"injection in base", "base=USD%26quotes%3DEUR", "invalid_base"},
		{"empty targets", "base=USD&targets=", "invalid_targets"},
		{"invalid target", "base=USD&targets=EUR,12", "invalid_targets"},
		{"blank target in list", "base=USD&targets=EUR,,SGD", "invalid_targets"},
		{"too many targets", "base=USD&targets=" + tooMany, "invalid_targets"},
		{"target not in table", "base=USD&targets=EUR,ZZZ", "unsupported_target"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, newTestHandler(successProvider()), "/api/rates?"+tt.query)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body)
			}
			if got := decode[errorResponse](t, rec); got.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestInvalidInputDoesNotReachProvider(t *testing.T) {
	provider := successProvider()
	get(t, newTestHandler(provider), "/api/rates?base=nope")

	if provider.calls != 0 {
		t.Errorf("provider called %d times, want 0", provider.calls)
	}
}

func TestRatesUpstreamErrors(t *testing.T) {
	const secret = "dial tcp 10.0.0.1:443"

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"unknown base", frankfurter.ErrUnknownCurrency, http.StatusBadRequest, "unsupported_base"},
		{"timeout", errors.Join(frankfurter.ErrTimeout, errors.New(secret)), http.StatusGatewayTimeout, "upstream_timeout"},
		{"unavailable", errors.Join(frankfurter.ErrUnavailable, errors.New(secret)), http.StatusBadGateway, "upstream_unavailable"},
		{"unexpected error", errors.New(secret), http.StatusBadGateway, "upstream_unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := get(t, newTestHandler(&fakeProvider{err: tt.err}), "/api/rates?base=USD&targets=EUR")

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := decode[errorResponse](t, rec); got.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if strings.Contains(rec.Body.String(), secret) {
				t.Errorf("response leaks internal error: %s", rec.Body)
			}
		})
	}
}

func TestWrongMethodIsRejected(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestHandler(successProvider()).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/rates?base=USD", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   string
	}{
		{"allowed origin", allowedOrigin, allowedOrigin},
		{"other origin", "https://evil.example", ""},
		{"other port", "http://localhost:3000", ""},
		{"no origin", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			newTestHandler(successProvider()).ServeHTTP(rec, req)

			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.want)
			}
			if got := rec.Header().Get("Vary"); got != "Origin" {
				t.Errorf("Vary = %q, want Origin", got)
			}
		})
	}
}

func TestCORSPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/rates", nil)
	req.Header.Set("Origin", allowedOrigin)
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()
	newTestHandler(successProvider()).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != allowedOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodGet) {
		t.Errorf("Access-Control-Allow-Methods = %q", got)
	}
}

// TestRatesCachingThroughFullStack wires the real service, cache and client to a
// fake upstream to check that different target lists share one cached table.
func TestRatesCachingThroughFullStack(t *testing.T) {
	var upstreamCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		if strings.Contains(r.URL.RawQuery, "quotes") {
			t.Errorf("upstream request was filtered: %s", r.URL.RawQuery)
		}
		w.Write([]byte(`[
			{"date":"2026-11-02","base":"USD","quote":"EUR","rate":0.89471},
			{"date":"2026-11-02","base":"USD","quote":"SGD","rate":1.2915}
		]`))
	}))
	defer upstream.Close()

	client := frankfurter.NewClient(upstream.URL, &http.Client{Timeout: time.Second})
	service := rates.NewService(client, cache.New[rates.Table](time.Hour, time.Now))
	h := newTestHandler(service)

	first := get(t, h, "/api/rates?base=USD&targets=EUR")
	if got := first.Header().Get("X-Cache"); got != "MISS" {
		t.Errorf("first X-Cache = %q, want MISS", got)
	}
	if decode[ratesResponse](t, first).Cached {
		t.Error("first response reported as cached")
	}

	second := get(t, h, "/api/rates?base=USD&targets=SGD")
	if got := second.Header().Get("X-Cache"); got != "HIT" {
		t.Errorf("second X-Cache = %q, want HIT", got)
	}
	body := decode[ratesResponse](t, second)
	if !body.Cached {
		t.Error("second response not reported as cached")
	}
	if body.Rates["SGD"].Rate != 1.2915 {
		t.Errorf("SGD rate = %v, want 1.2915", body.Rates["SGD"].Rate)
	}
	if got := upstreamCalls.Load(); got != 1 {
		t.Errorf("upstream called %d times, want 1", got)
	}
}
