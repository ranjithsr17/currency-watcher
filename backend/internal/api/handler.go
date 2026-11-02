// Package api implements the HTTP interface of the Currency Watcher backend.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"currencywatcher/internal/frankfurter"
	"currencywatcher/internal/rates"
)

// maxTargets limits the size of the targets query parameter.
const maxTargets = 50

var currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)

// RatesProvider returns the complete rate table for a base currency.
type RatesProvider interface {
	Get(ctx context.Context, base string) (rates.Result, error)
}

type handler struct {
	rates  RatesProvider
	logger *slog.Logger
}

// NewHandler returns the API's root handler. Browsers may only call it from
// allowedOrigins.
func NewHandler(provider RatesProvider, logger *slog.Logger, allowedOrigins []string) http.Handler {
	h := &handler{rates: provider, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", h.health)
	mux.HandleFunc("GET /api/rates", h.getRates)

	return logRequests(logger, cors(allowedOrigins, mux))
}

type healthResponse struct {
	Status string `json:"status"`
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

type rateResponse struct {
	Rate float64 `json:"rate"`
	Date string  `json:"date"`
}

type ratesResponse struct {
	Base      string                  `json:"base"`
	Rates     map[string]rateResponse `json:"rates"`
	FetchedAt time.Time               `json:"fetchedAt"`
	ExpiresAt time.Time               `json:"expiresAt"`
	Cached    bool                    `json:"cached"`
}

func (h *handler) getRates(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	base, err := parseBase(query.Get("base"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_base", err.Error())
		return
	}
	targets, err := parseTargets(query.Get("targets"), query.Has("targets"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_targets", err.Error())
		return
	}

	result, err := h.rates.Get(r.Context(), base)
	if err != nil {
		h.writeRatesError(w, r, base, err)
		return
	}

	selected, err := selectRates(result.Rates, targets)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unsupported_target", err.Error())
		return
	}

	if result.Cached {
		w.Header().Set("X-Cache", "HIT")
	} else {
		w.Header().Set("X-Cache", "MISS")
	}
	// The backend owns freshness, so a refresh must always reach it.
	w.Header().Set("Cache-Control", "no-store")

	writeJSON(w, http.StatusOK, ratesResponse{
		Base:      result.Base,
		Rates:     selected,
		FetchedAt: result.FetchedAt.UTC().Truncate(time.Second),
		ExpiresAt: result.ExpiresAt.UTC().Truncate(time.Second),
		Cached:    result.Cached,
	})
}

func (h *handler) writeRatesError(w http.ResponseWriter, r *http.Request, base string, err error) {
	switch {
	case errors.Is(err, frankfurter.ErrUnknownCurrency):
		writeError(w, http.StatusBadRequest, "unsupported_base", fmt.Sprintf("currency %s is not supported", base))
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		// The client has gone, so there is nobody to respond to.
	case errors.Is(err, frankfurter.ErrTimeout):
		h.logger.Error("exchange rate request timed out", "base", base, "error", err)
		writeError(w, http.StatusGatewayTimeout, "upstream_timeout", "the exchange rate provider took too long to respond")
	default:
		h.logger.Error("exchange rate request failed", "base", base, "error", err)
		writeError(w, http.StatusBadGateway, "upstream_unavailable", "the exchange rate provider is unavailable")
	}
}

func parseBase(raw string) (string, error) {
	base := normalize(raw)
	if base == "" {
		return "", errors.New("query parameter base is required")
	}
	if !currencyCode.MatchString(base) {
		return "", fmt.Errorf("base must be a 3-letter currency code, got %q", raw)
	}
	return base, nil
}

// parseTargets returns the distinct requested targets. A nil result means no
// targets parameter was sent, which selects every available rate.
func parseTargets(raw string, present bool) ([]string, error) {
	if !present {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	if len(parts) > maxTargets {
		return nil, fmt.Errorf("targets can contain at most %d currencies", maxTargets)
	}

	targets := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		target := normalize(part)
		if !currencyCode.MatchString(target) {
			return nil, fmt.Errorf("targets must be comma-separated 3-letter currency codes, got %q", part)
		}
		if !seen[target] {
			seen[target] = true
			targets = append(targets, target)
		}
	}
	return targets, nil
}

func normalize(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func selectRates(table rates.Table, targets []string) (map[string]rateResponse, error) {
	if targets == nil {
		selected := make(map[string]rateResponse, len(table))
		for code, quote := range table {
			selected[code] = rateResponse{Rate: quote.Rate, Date: quote.Date}
		}
		return selected, nil
	}

	selected := make(map[string]rateResponse, len(targets))
	for _, code := range targets {
		quote, ok := table[code]
		if !ok {
			return nil, fmt.Errorf("currency %s is not supported", code)
		}
		selected[code] = rateResponse{Rate: quote.Rate, Date: quote.Date}
	}
	return selected, nil
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorDetail{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// An encoding failure here means the client has gone or the status is already sent.
	_ = json.NewEncoder(w).Encode(body)
}
