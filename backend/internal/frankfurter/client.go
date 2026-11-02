// Package frankfurter is a client for the Frankfurter exchange rate API (v2).
package frankfurter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// maxResponseBytes limits how much of a response is read. A full rate table for
// one base currency is around 10 KB.
const maxResponseBytes = 1 << 20

var (
	// ErrUnknownCurrency means Frankfurter does not recognise the base currency.
	ErrUnknownCurrency = errors.New("unknown currency")
	// ErrTimeout means Frankfurter did not respond in time.
	ErrTimeout = errors.New("frankfurter timed out")
	// ErrUnavailable means the request failed or the response was unusable.
	ErrUnavailable = errors.New("frankfurter unavailable")
)

// Rate is one row of a Frankfurter rates response. Rows in the same response can
// have different dates because currencies are published on different schedules.
type Rate struct {
	Date  string  `json:"date"`
	Base  string  `json:"base"`
	Quote string  `json:"quote"`
	Rate  float64 `json:"rate"`
}

// Client calls the Frankfurter API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a Client for the API at baseURL, such as
// https://api.frankfurter.dev/v2. The httpClient should have a timeout set.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    httpClient,
	}
}

// Rates returns the latest rates for every currency quoted against base.
func (c *Client) Rates(ctx context.Context, base string) ([]Rate, error) {
	endpoint := c.baseURL + "/rates?" + url.Values{"base": {base}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %w", ErrUnavailable, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "currency-watcher/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, classify(err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusUnprocessableEntity:
		return nil, fmt.Errorf("%w: %s", ErrUnknownCurrency, base)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%w: unexpected status %d", ErrUnavailable, resp.StatusCode)
	}

	return decodeRates(resp.Body)
}

func decodeRates(r io.Reader) ([]Rate, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxResponseBytes+1))
	if err != nil {
		return nil, classify(err)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("%w: response larger than %d bytes", ErrUnavailable, maxResponseBytes)
	}

	var rates []Rate
	if err := json.Unmarshal(body, &rates); err != nil {
		return nil, fmt.Errorf("%w: decode response: %w", ErrUnavailable, err)
	}
	if len(rates) == 0 {
		return nil, fmt.Errorf("%w: response contained no rates", ErrUnavailable)
	}
	for _, rate := range rates {
		if rate.Quote == "" || rate.Rate <= 0 {
			return nil, fmt.Errorf("%w: invalid rate for %q", ErrUnavailable, rate.Quote)
		}
	}
	return rates, nil
}

func classify(err error) error {
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return fmt.Errorf("%w: %w", ErrTimeout, err)
	}
	return fmt.Errorf("%w: %w", ErrUnavailable, err)
}
