# DepthFeed Golang Client/SDK/Library

An idiomatic Go client for the [DepthFeed v3 API](https://depthfeed.com/docs) — REST, WebSocket, and SSE coverage for prediction-market order books, price bars, sports data, and Paper Trading.

![DepthFeed Golang API SDK](https://i.postimg.cc/SKtvb0Rg/depthfeed-golang-api.jpg)

[![CI](https://github.com/tigusigalpa/depthfeed-go/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/depthfeed-go/actions/workflows/ci.yml)
[![Tests](https://github.com/tigusigalpa/depthfeed-go/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/depthfeed-go/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green?style=flat-square)](LICENSE)
[![CodeQL](https://github.com/tigusigalpa/depthfeed-go/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/tigusigalpa/depthfeed-go/actions/workflows/codeql.yml)
[![Codecov](https://codecov.io/gh/tigusigalpa/depthfeed-go/graph/badge.svg)](https://codecov.io/gh/tigusigalpa/depthfeed-go)
[![GitHub Release](https://img.shields.io/github/v/release/tigusigalpa/depthfeed-go?style=flat-square)](https://github.com/tigusigalpa/depthfeed-go/releases)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue?style=flat-square&logo=go)](https://pkg.go.dev/github.com/tigusigalpa/depthfeed-go)

## Features

- **Complete typed REST coverage** — every documented v3 route: identity, markets, snapshots, ticks, Binance, bars, Kalshi, Limitless, Binance Prediction, screener, batch, sports, sports webhooks, and Paper Trading.
- **Live WebSocket stream** — subscribe/unsubscribe, ping/pong, typed frames, jittered exponential reconnect with subscription replay, and documented close-code handling.
- **Sports odds SSE** — correct SSE parsing (multiline data, comments, id/retry), context cancellation, and optional reconnect.
- **Webhook verification** — HMAC-SHA256 over raw request bytes with constant-time comparison for Sports injury and Paper Trading fill callbacks.
- **Safe authentication** — `Authorization: Bearer` by default, `X-API-Key` alternative, no keys in URLs or logs.
- **Cursor pagination** — opaque cursors passed back unchanged, `has_more` termination, `include_count` opt-in.
- **Typed errors** — `*APIError` with HTTP status, stable error code, message, details, request ID, rate-limit headers, and transport cause. Sentinel errors for `errors.Is` (`ErrNotFound`, `ErrRateLimited`, …).
- **Opt-in retry** — capped exponential backoff with jitter, `Retry-After` header and `details.retry_after` honored, only retryable codes/statuses retried, non-idempotent calls never retried automatically.
- **Timestamp helpers** — accept epoch-ms or ISO-8601, preserve millisecond precision, no lossy conversions.
- **Raw fallbacks** — unknown fields and dynamic per-coin price keys preserved as `json.RawMessage` where the contract requires it.
- **Context-aware** — every REST method takes `context.Context` for cancellation and deadlines.
- **Testable** — mock HTTP server, fake credentials, no real API calls in tests.

## Installation

```bash
go get github.com/tigusigalpa/depthfeed-go
```

Requires Go 1.22+.

Before running an authenticated example, create an API key in DepthFeed and
export it in your shell. Never commit the key or put it in a URL.

```powershell
$env:DEPTHFEED_API_KEY = "df_..."
```

```bash
export DEPTHFEED_API_KEY='df_...'
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    depthfeed "github.com/tigusigalpa/depthfeed-go"
)

func main() {
    client, err := depthfeed.NewClient(depthfeed.Config{
        APIKey: os.Getenv("DEPTHFEED_API_KEY"),
        Retry: depthfeed.RetryConfig{Enabled: true},
    })
    if err != nil {
        log.Fatal(err)
    }

    who, _, err := client.Identity.WhoAmI(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Plan: %s\n", who.Plan)

    markets, pg, _, err := client.Markets.List(context.Background(), depthfeed.CoinBTC, &depthfeed.MarketListOptions{
        ListOptions: depthfeed.ListOptions{Limit: 100},
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("%d markets, has_more=%v\n", len(markets), pg.HasMorePages())
}
```

`NewClient` may be used without a key when you only need public endpoints
(`Health` and `Overview`). Any authenticated call made without a key fails
locally with `depthfeed.ErrMissingAPIKey`, before an HTTP request is sent.

## Configuration

| Option | Description | Default |
|--------|-------------|---------|
| `WithBaseURL(url)` | Override REST base URL | `https://api.depthfeed.com/v3` |
| `WithWebSocketURL(url)` | Override WebSocket URL | `wss://api.depthfeed.com/v3/stream` |
| `WithHTTPClient(c)` | Custom `*http.Client` | 30s timeout |
| `WithTimeout(d)` | HTTP timeout | 30s |
| `WithAuthStrategy(s)` | `AuthBearer` or `AuthHeader` | `AuthBearer` |
| `WithUserAgent(ua)` | Custom User-Agent | `depthfeed-go/1.0.0` |
| `WithRetry(r)` | Enable/configure retry | disabled |
| `WithLogger(l)` | Diagnostic logger | nop |

### From environment

```go
client, err := depthfeed.NewClientFromEnv()
// Reads DEPTHFEED_API_KEY, DEPTHFEED_BASE_URL, DEPTHFEED_WS_URL,
// DEPTHFEED_AUTH_STRATEGY, DEPTHFEED_USER_AGENT.
```

## Authentication

All routes except `/v3/health` and `/v3/overview` require an API key (`df_...`). The SDK sends it as `Authorization: Bearer df_...` by default. To use `X-API-Key` instead:

```go
client, _ := depthfeed.NewClient(depthfeed.Config{
    APIKey: "df_...",
}, depthfeed.WithAuthStrategy(depthfeed.AuthHeader))
```

Keys are never placed in URLs, query parameters, or log output.

## API coverage

| Service | Routes |
|---------|--------|
| `Identity` | `GET /health`, `GET /whoami`, `GET /overview` |
| `Markets` | `GET /{coin}/markets`, `GET /{coin}/markets/{id}`, `GET /{coin}/markets/by-slug/{slug}`, `GET /polymarket/markets` |
| `Snapshots` | `GET /{coin}/markets/{id}/snapshots`, `GET .../at/{ts}`, `GET /polymarket/{asset_id}/snapshots` |
| `Ticks` | `GET /{coin}/markets/{id}/ticks`, `GET /polymarket/{asset_id}/ticks`, `GET /kalshi/{ticker}/ticks` |
| `Binance` | `GET /{coin}/{venue}/latest`, `.../snapshots`, `.../trades`, `.../trades/latest` |
| `Bars` | `GET /bars` |
| `Kalshi` | `GET /kalshi/markets`, `.../{ticker}/orderbook/latest`, `.../snapshots`, `.../candles` |
| `Limitless` | `GET /limitless/markets`, `.../{slug}/orderbook/latest`, `.../snapshots` |
| `BinancePrediction` | `GET /binance-prediction/markets`, `.../{id}/orderbook/latest`, `.../snapshots`, `.../timeseries`, `.../trades` |
| `Screener` | `GET /screener`, `GET /screener/{asset}/{window}` |
| `Batch` | `POST /batch` |
| `Sports` | 20+ sports routes (overview, leagues, markets, books, games, players, injuries, odds) |
| `SportsWebhooks` | `POST/GET /sports/webhooks`, `POST .../{id}/test`, `DELETE .../{id}` |
| `Paper` | `POST /paper/hook/{token}` |

Full route map: see the [official docs](https://depthfeed.com/docs).

## Pagination

DepthFeed uses keyset (cursor) pagination. Pass `ListOptions{Cursor: pg.NextCursor}` to fetch the next page. Cursors are opaque — pass them back unchanged.

```go
opts := &depthfeed.MarketListOptions{ListOptions: depthfeed.ListOptions{Limit: 100}}
for {
    markets, pg, _, err := client.Markets.List(ctx, depthfeed.CoinBTC, opts)
    if err != nil { return err }
    // process markets...
    if pg == nil || !pg.HasMore { break }
    opts.Cursor = pg.NextCursor
}
```

For historical snapshots, ticks, and bars, keep concurrent page requests
bounded. Prefer `limit=1000` where the endpoint supports it, a narrow time
window, and the returned cursor. DepthFeed enforces separate historical-data
concurrency limits, so aggressive parallel pagination is likely to be slower
than a small worker pool.

## Error handling

```go
markets, _, _, err := client.Markets.List(ctx, depthfeed.CoinBTC, nil)
if err != nil {
    var apiErr *depthfeed.APIError
    if errors.As(err, &apiErr) {
        fmt.Println(apiErr.Code)        // "INVALID_COIN"
        fmt.Println(apiErr.HTTPStatus)  // 400
        fmt.Println(apiErr.RequestID)   // "req_..."
    }
    if errors.Is(err, depthfeed.ErrRateLimited) {
        // back off
    }
}
```

### Error codes

| Code | HTTP | Retryable | Description |
|------|------|-----------|-------------|
| `AUTH_MISSING` / `AUTH_INVALID` | 401 | No | Missing or invalid key |
| `COIN_NOT_IN_PLAN` | 402 | No | Asset not in plan |
| `SYMBOL_LIMIT_EXCEEDED` | 402 | No | Too many symbols in /bars |
| `HISTORY_LIMIT_EXCEEDED` | 403 | No | Window older than plan allows |
| `INVALID_COIN` / `INVALID_CURSOR` | 400 | No | Bad parameter |
| `RATE_LIMIT_BURST` / `RATE_LIMIT_SUSTAINED` | 429 | Yes | Rate limited |
| `HISTORY_CONCURRENCY_LIMIT` | 429 | Yes | Account concurrency |
| `HISTORY_POOL_BUSY` | 429 | Yes | Shared pool busy |
| `BARS_BUSY` | 503 | Yes | Bars engine busy (has `details.retry_after`) |
| `UPSTREAM_UNAVAILABLE` | 503 | Yes | Upstream venue down |

## Retry

Retry is **opt-in**. When enabled, only retryable codes (rate limits, `BARS_BUSY`, `UPSTREAM_UNAVAILABLE`, `AUTH_UNAVAILABLE`, `INTERNAL_ERROR`) and transient network errors are retried with capped exponential backoff and jitter. `Retry-After` headers and `details.retry_after` are honored. Non-idempotent calls (POST/DELETE) are never retried automatically except `POST /batch` (GET-only subrequests).

```go
client, _ := depthfeed.NewClient(depthfeed.Config{
    APIKey: "df_...",
    Retry: depthfeed.RetryConfig{
        Enabled:      true,
        MaxAttempts:  4,
        InitialDelay: 250 * time.Millisecond,
        MaxDelay:     5 * time.Second,
        OnRetry: func(attempt int, delay time.Duration, err error) {
            log.Printf("retry %d in %v: %v", attempt, delay, err)
        },
    },
})
```

## WebSocket streaming

```go
sc := client.NewStreamClient(stream.WithReconnect(true))
sc.OnEvent(func(e stream.Event) {
    if e.Kind == stream.EventReconnected {
        log.Println("reconnected")
    }
})
sc.Connect(ctx)
sc.Subscribe([]string{stream.BookPolymarket("btc", marketID)})

for {
    frame, err := sc.Receive(ctx)
    if err != nil { break }
    if frame.Op == "book" {
        var snap depthfeed.Snapshot
        json.Unmarshal(frame.Data, &snap)
    }
}
```

Channel builders: `BookPolymarket`, `BookKalshi`, `BookLimitless`, `BookBinance`, `BookSportsPolymarket`, `BookSportsKalshi`, `Price`.

Close codes: `4001` (auth failed), `4003` (plan limit), `4029` (op flooding), `1013` (auth unavailable). Fatal codes (`4001`, `4003`) do not trigger reconnect.

Always give `Connect` and `Receive` a context with a deadline or cancellation
path, and call `Close` during application shutdown. The stream client restores
active subscriptions after an unexpected reconnect; it does not reconnect
after an explicit `Close`.

## Sports odds SSE

The sports odds feed is a long-lived Server-Sent Events connection. It lives
in the small `sse` subpackage so applications that only need REST do not need
to manage an always-on stream.

```go
import (
    "os"

    "github.com/tigusigalpa/depthfeed-go/sse"
)

odds := sse.NewClient(sse.Config{
    URL:       "https://api.depthfeed.com/v3/sports/odds/stream",
    APIKey:    os.Getenv("DEPTHFEED_API_KEY"),
    Reconnect: true,
})

err := odds.Stream(ctx, func(event sse.Event) {
    if event.Event != "odds" || !event.HasData() {
        return
    }
    // event.Data is the exact JSON payload; event.ID is retained for reconnects.
})
```

The client sends `Last-Event-ID` when it reconnects, parses multiline `data:`
fields correctly, and never accumulates an unbounded event buffer. Call
`odds.Close()` to stop it explicitly.

## Webhooks

Verify Sports injury webhook deliveries using HMAC-SHA256 over the raw request body:

```go
ok, err := webhooks.VerifySportsInjury(rawBody, sigHeader, secret)
if !ok {
    http.Error(w, "invalid signature", 401)
    return
}
```

See `examples/verify-webhook/` for a complete net/http receiver.

Read the HTTP request body once, verify those exact bytes, then decode JSON.
The signing secret returned while creating a subscription is shown only once;
treat it like a password and do not log it. Paper Trading calls are not retried
automatically: use a stable `PaperSignal.ClientID` if your application retries
them after an ambiguous network failure.

## Examples

- [`examples/markets/`](examples/markets) — list markets and page through snapshots
- [`examples/stream/`](examples/stream) — subscribe to the live WebSocket book stream
- [`examples/verify-webhook/`](examples/verify-webhook) — verify a Sports injury webhook delivery

## Testing

```bash
go test ./...
go vet ./...
gofmt -l .
```

Tests use a mock HTTP server and fake credentials (`df_test_example_not_a_real_key`). No real API calls are made.

For a local quality check matching CI, run:

```bash
go mod verify
go vet ./...
go test -race ./...
```

## Versioning

This SDK follows [SemVer 2.0.0](https://semver.org). See [CHANGELOG.md](CHANGELOG.md).

## Author

Igor Sazonov — [@tigusigalpa](https://github.com/tigusigalpa) — sovletig@gmail.com

## License

MIT — see [LICENSE](LICENSE).
