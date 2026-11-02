# Currency Watcher

A dashboard for tracking exchange rates between currency pairs. A React frontend talks to a Go API, which gets real rates from the [Frankfurter API v2](https://frankfurter.dev) and caches them in memory for one hour.

```
React + Vite (localhost:5173) → Go API (localhost:8080) → In-memory cache → Frankfurter API v2
```

AWS deployment is not set up yet, so `terraform/` and `.github/workflows/` are empty placeholders.

## Run locally

You need [Go 1.27+](https://go.dev/dl/) and [Node.js 20.19+](https://nodejs.org). Use two terminals.

**Terminal 1: backend**

```sh
cd backend
go run ./cmd/server
```

**Terminal 2: frontend**

```sh
cd frontend
cp .env.example .env    # PowerShell: Copy-Item .env.example .env
npm install
npm run dev
```

Open <http://localhost:5173>. It must be this exact address, because the backend's CORS allowlist names it.

## API

### `GET /api/health`

Returns `{"status":"ok"}`. It doesn't call Frankfurter, so an upstream outage doesn't make the service look unhealthy.

### `GET /api/rates?base=USD&targets=EUR,SGD`

| Parameter | Required | Description |
|---|---|---|
| `base` | Yes | 3-letter currency code |
| `targets` | No | Comma-separated codes, up to 50. Omit it to get every available rate. |

```json
{
  "base": "USD",
  "rates": {
    "EUR": { "rate": 0.89471, "date": "2026-11-02" },
    "SGD": { "rate": 1.2915, "date": "2026-11-02" }
  },
  "fetchedAt": "2026-11-02T10:00:00Z",
  "expiresAt": "2026-11-02T11:00:00Z",
  "cached": true
}
```

Each rate has its own date because Frankfurter's sources publish on different schedules. The `X-Cache` response header is `HIT` or `MISS`.

Errors use the format `{"error":{"code":"...","message":"..."}}`.

| Status | Code | Meaning |
|---|---|---|
| 400 | `invalid_base`, `invalid_targets` | The parameter is missing or malformed |
| 400 | `unsupported_base`, `unsupported_target` | The code is well formed, but Frankfurter doesn't list it |
| 502 | `upstream_unavailable` | Frankfurter failed or returned an unusable response |
| 504 | `upstream_timeout` | Frankfurter didn't respond within 10 seconds |

## Caching

- **Storage:** a Go map guarded by `sync.RWMutex`, in `internal/cache`.
- **What is cached:** the complete rate table for each base currency. `USD→EUR` and `USD→SGD` therefore share one entry and one upstream call.
- **Expiry:** one hour. Expired entries are ignored on read and replaced by the next fetch.
- **Failures:** errors are never cached, so the next request tries again. Expired data is never served.
- **Concurrent misses:** requests for the same base share one upstream call, and a client disconnecting doesn't cancel it for the others.
- **Limits:** the cache belongs to one process, so a restart empties it and each instance keeps its own copy. Only successful responses for real currencies are stored, which keeps the number of entries small without a background cleanup.

The Refresh button always reaches the backend, but returns the same cached rates until the hour is up. The dashboard therefore shows both when it last updated and when the data was fetched from Frankfurter.

## Configuration

**Backend** (environment variables, all optional)

| Variable | Default |
|---|---|
| `PORT` | `8080` |
| `ALLOWED_ORIGINS` | `http://localhost:5173` (comma-separated) |
| `FRANKFURTER_BASE_URL` | `https://api.frankfurter.dev/v2` |

**Frontend** (`frontend/.env`)

| Variable | Default |
|---|---|
| `VITE_API_BASE_URL` | `http://localhost:8080` |

## Project structure

```
backend/
  cmd/server/          Startup, configuration and graceful shutdown
  internal/api/        Routes, validation, error responses, CORS, logging
  internal/cache/      Generic TTL cache
  internal/frankfurter/  Frankfurter API client
  internal/rates/      Combines the cache and client, and shares concurrent fetches
frontend/src/
  api/                 Backend client
  components/          Reusable UI
  hooks/               Tracked pairs, rate loading and the currency list
  utils/               Pair storage and formatting
```

## Tests

```sh
cd backend
gofmt -l .
go vet ./...
go test ./...
go test -race ./...   # needs CGO_ENABLED=1 and a C compiler such as gcc
```

On a machine with little free memory, `go test ./...` can crash the Go compiler. Limit the parallel builds with `go test -p 1 ./...`, and set the `GOMAXPROCS` environment variable to `2`.

On Windows, antivirus can briefly lock the test binary and cause an `unlinkat ... being used by another process` error after the tests have passed. Run the command again.
