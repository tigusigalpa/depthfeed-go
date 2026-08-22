# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial release of `depthfeed-go`, an idiomatic Go client for the DepthFeed v3 API.
- Complete typed REST coverage for all documented v3 routes:
  - Identity: `GET /health`, `GET /whoami`, `GET /overview`
  - Markets: `GET /{coin}/markets`, `GET /{coin}/markets/{id}`, `GET /{coin}/markets/by-slug/{slug}`, `GET /polymarket/markets`
  - Snapshots: `GET /{coin}/markets/{id}/snapshots`, `GET .../at/{ts}`, `GET /polymarket/{asset_id}/snapshots`
  - Ticks: `GET /{coin}/markets/{id}/ticks`, `GET /polymarket/{asset_id}/ticks`, `GET /kalshi/{ticker}/ticks`
  - Binance: `GET /{coin}/{venue}/latest`, `.../snapshots`, `.../trades`, `.../trades/latest`
  - Bars: `GET /bars`
  - Kalshi: `GET /kalshi/markets`, `.../{ticker}/orderbook/latest`, `.../snapshots`, `.../candles`
  - Limitless: `GET /limitless/markets`, `.../{slug}/orderbook/latest`, `.../snapshots`
  - Binance Prediction: `GET /binance-prediction/markets`, `.../{id}/orderbook/latest`, `.../snapshots`, `.../timeseries`, `.../trades`
  - Screener: `GET /screener`, `GET /screener/{asset}/{window}`
  - Batch: `POST /batch`
  - Sports: 20+ routes (overview, leagues, markets, books, games, players, injuries, odds)
  - Sports Webhooks: `POST/GET /sports/webhooks`, `POST .../{id}/test`, `DELETE .../{id}`
  - Paper Trading: `POST /paper/hook/{token}`
- Live WebSocket stream client (`stream` subpackage) with subscribe/unsubscribe, ping/pong, typed frames, jittered exponential reconnect with subscription replay, and documented close-code handling.
- Sports odds SSE client (`sse` subpackage) with correct SSE parsing, context cancellation, and optional reconnect.
- Webhook verification helpers (`webhooks` subpackage) for Sports injury and Paper Trading fill callbacks using HMAC-SHA256 with constant-time comparison.
- Typed `*APIError` with HTTP status, stable error code, message, details, request ID, rate-limit headers, and transport cause.
- Sentinel errors for `errors.Is` (`ErrNotFound`, `ErrRateLimited`, `ErrBadRequest`, `ErrUnauthorized`, `ErrPaymentRequired`, `ErrForbidden`, `ErrServerError`, `ErrTransportFailure`).
- Opt-in retry with capped exponential backoff, jitter, `Retry-After` header and `details.retry_after` honoring.
- Cursor pagination helpers with opaque cursor pass-through and `has_more` termination.
- Timestamp helpers accepting epoch-ms or ISO-8601 with millisecond precision preservation.
- `[price, size]` tuple `PriceLevel` type with safe JSON marshaling.
- Raw `json.RawMessage` fallbacks for unknown fields and dynamic per-coin price keys.
- Configurable auth strategy (`Authorization: Bearer` or `X-API-Key`).
- `NewClientFromEnv` for environment-based configuration.
- Comprehensive test suite with mock HTTP server, endpoint coverage tests, and fake credentials.
- Examples: markets listing, WebSocket streaming, webhook verification.
- CI workflow (Go 1.22–1.26, gofmt, go vet, tests with coverage).
- README, CHANGELOG, CONTRIBUTING, LICENSE (MIT).
