package frankfurter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(server.URL+"/v2/", &http.Client{Timeout: time.Second})
}

func TestRatesSuccess(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/rates" {
			t.Errorf("path = %q, want /v2/rates", r.URL.Path)
		}
		if got := r.URL.Query().Get("base"); got != "USD" {
			t.Errorf("base = %q, want USD", got)
		}
		w.Write([]byte(`[
			{"date":"2026-11-02","base":"USD","quote":"EUR","rate":0.89471},
			{"date":"2026-11-01","base":"USD","quote":"AFN","rate":64.93}
		]`))
	})

	rates, err := client.Rates(context.Background(), "USD")
	if err != nil {
		t.Fatalf("Rates returned error: %v", err)
	}
	if len(rates) != 2 {
		t.Fatalf("got %d rates, want 2", len(rates))
	}
	want := Rate{Date: "2026-11-02", Base: "USD", Quote: "EUR", Rate: 0.89471}
	if rates[0] != want {
		t.Errorf("rates[0] = %+v, want %+v", rates[0], want)
	}
}

func TestRatesErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"unknown currency", http.StatusUnprocessableEntity, `{"status":422,"message":"invalid currency: ZZZ"}`, ErrUnknownCurrency},
		{"wrong path is not an unknown currency", http.StatusNotFound, `{"status":404,"message":"not found"}`, ErrUnavailable},
		{"rate limited", http.StatusTooManyRequests, ``, ErrUnavailable},
		{"server error", http.StatusInternalServerError, `oops`, ErrUnavailable},
		{"malformed JSON", http.StatusOK, `{not json`, ErrUnavailable},
		{"unexpected shape", http.StatusOK, `{"rates":{}}`, ErrUnavailable},
		{"no rates", http.StatusOK, `[]`, ErrUnavailable},
		{"zero rate", http.StatusOK, `[{"date":"2026-11-02","base":"USD","quote":"EUR","rate":0}]`, ErrUnavailable},
		{"missing quote", http.StatusOK, `[{"date":"2026-11-02","base":"USD","rate":1.5}]`, ErrUnavailable},
		{"oversized response", http.StatusOK, strings.Repeat(" ", maxResponseBytes+1), ErrUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			})

			_, err := client.Rates(context.Background(), "USD")
			if !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestRatesTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(release) })

	client := NewClient(server.URL, &http.Client{Timeout: 20 * time.Millisecond})

	_, err := client.Rates(context.Background(), "USD")
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("err = %v, want %v", err, ErrTimeout)
	}
}

func TestRatesConnectionFailure(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	client := NewClient(server.URL, &http.Client{Timeout: time.Second})

	_, err := client.Rates(context.Background(), "USD")
	if !errors.Is(err, ErrUnavailable) {
		t.Errorf("err = %v, want %v", err, ErrUnavailable)
	}
}
