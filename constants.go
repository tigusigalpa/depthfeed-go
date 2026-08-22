package depthfeed

// Code generated constants for the DepthFeed v3 API. These mirror the
// documented finite value sets at https://depthfeed.com/docs and are
// exposed as typed string values so callers can branch safely while still
// passing a free-form string for forward-compatible, undocumented values.

// Coin is a supported base asset. DepthFeed covers seven assets across
// every venue that lists them: btc, eth, sol, xrp, doge, bnb, hype.
type Coin string

// Documented coins. See https://depthfeed.com/docs#ep-markets.
const (
	CoinBTC  Coin = "btc"
	CoinETH  Coin = "eth"
	CoinSOL  Coin = "sol"
	CoinXRP  Coin = "xrp"
	CoinDOGE Coin = "doge"
	CoinBNB  Coin = "bnb"
	CoinHYPE Coin = "hype"
)

// supportedCoins is the set of coins the API accepts in {coin} path
// segments. It is used for local validation before a network round-trip.
var supportedCoins = map[Coin]bool{
	CoinBTC:  true,
	CoinETH:  true,
	CoinSOL:  true,
	CoinXRP:  true,
	CoinDOGE: true,
	CoinBNB:  true,
	CoinHYPE: true,
}

// IsValid reports whether c is one of the documented coins.
func (c Coin) IsValid() bool { return supportedCoins[c] }

// MarketWindow is a recurring up/down market duration. The set varies by
// venue (Polymarket: 5m/15m/1h/4h/24h; Limitless: 5m/15m; Kalshi: 15m/1h/24h/1w).
type MarketWindow string

// Documented market windows.
const (
	Window5m  MarketWindow = "5m"
	Window15m MarketWindow = "15m"
	Window1h  MarketWindow = "1h"
	Window4h  MarketWindow = "4h"
	Window24h MarketWindow = "24h"
	Window1w  MarketWindow = "1w"
)

// Outcome is a binary-market outcome side.
type Outcome string

// Documented outcomes.
const (
	OutcomeUp   Outcome = "up"
	OutcomeDown Outcome = "down"
)

// Side is a Kalshi yes/no side or an order side.
type Side string

// Documented sides.
const (
	SideYes  Outcome = "yes"
	SideNo   Outcome = "no"
	SideBuy  Side    = "BUY"
	SideSell Side    = "SELL"
)

// Venue is a Binance underlying venue: spot or futures.
type Venue string

// Documented Binance venues. {venue} is restricted to spot or futures.
const (
	VenueSpot    Venue = "spot"
	VenueFutures Venue = "futures"
)

// IsValid reports whether v is a documented Binance venue.
func (v Venue) IsValid() bool { return v == VenueSpot || v == VenueFutures }

// BarsTimeframe is a /v3/bars timeframe. The finest resolution is 1Min;
// coarser frames are rolled up from it.
type BarsTimeframe string

// Documented bars timeframes.
const (
	Timeframe1Min  BarsTimeframe = "1Min"
	Timeframe5Min  BarsTimeframe = "5Min"
	Timeframe15Min BarsTimeframe = "15Min"
	Timeframe1Hour BarsTimeframe = "1Hour"
	Timeframe1Day  BarsTimeframe = "1Day"
)

// BarsVenue selects the market for /v3/bars outright.
type BarsVenue string

// Documented bars venues.
const (
	BarsVenueUSEquities     BarsVenue = "us-equities"
	BarsVenueBinance        BarsVenue = "binance"
	BarsVenueBinanceFutures BarsVenue = "binance-futures"
)

// PredictionMetric is a Binance-prediction timeseries metric.
type PredictionMetric string

// Documented Binance-prediction timeseries metrics.
const (
	MetricChance    PredictionMetric = "chance"
	MetricPrice     PredictionMetric = "price"
	MetricLastTrade PredictionMetric = "last_trade"
)

// AuthStrategy selects which header carries the API key.
type AuthStrategy string

// Documented auth strategies. Both are accepted by the API.
const (
	// AuthBearer sends Authorization: Bearer df_... (recommended).
	AuthBearer AuthStrategy = "bearer"
	// AuthHeader sends X-API-Key: df_...
	AuthHeader AuthStrategy = "header"
)

// ErrorCode is a stable DepthFeed error code from the error envelope.
// Branch on these rather than string-matching human-readable messages.
// See https://depthfeed.com/docs#errors.
type ErrorCode string

// Documented error codes.
const (
	ErrCodeAuthMissing             ErrorCode = "AUTH_MISSING"
	ErrCodeAuthInvalid             ErrorCode = "AUTH_INVALID"
	ErrCodeCoinNotInPlan           ErrorCode = "COIN_NOT_IN_PLAN"
	ErrCodeEndpointNotInPlan       ErrorCode = "ENDPOINT_NOT_IN_PLAN"
	ErrCodeSymbolLimitExceeded     ErrorCode = "SYMBOL_LIMIT_EXCEEDED"
	ErrCodeWindowTooWide           ErrorCode = "WINDOW_TOO_WIDE"
	ErrCodeHistoryLimitExceeded    ErrorCode = "HISTORY_LIMIT_EXCEEDED"
	ErrCodeInvalidCoin             ErrorCode = "INVALID_COIN"
	ErrCodeInvalidCursor           ErrorCode = "INVALID_CURSOR"
	ErrCodeInvalidTimestamp        ErrorCode = "INVALID_TIMESTAMP"
	ErrCodeInvalidInterval         ErrorCode = "INVALID_INTERVAL"
	ErrCodeInvalidSlug             ErrorCode = "INVALID_SLUG"
	ErrCodeInvalidTicker           ErrorCode = "INVALID_TICKER"
	ErrCodeMarketTickerRequired    ErrorCode = "MARKET_TICKER_REQUIRED"
	ErrCodeBatchTooLarge           ErrorCode = "BATCH_TOO_LARGE"
	ErrCodeBatchInvalidPath        ErrorCode = "BATCH_INVALID_PATH"
	ErrCodeMarketNotFound          ErrorCode = "MARKET_NOT_FOUND"
	ErrCodeSnapshotNotFound        ErrorCode = "SNAPSHOT_NOT_FOUND"
	ErrCodeEndpointNotFound        ErrorCode = "ENDPOINT_NOT_FOUND"
	ErrCodeRateLimitBurst          ErrorCode = "RATE_LIMIT_BURST"
	ErrCodeRateLimitSustained      ErrorCode = "RATE_LIMIT_SUSTAINED"
	ErrCodeHistoryConcurrencyLimit ErrorCode = "HISTORY_CONCURRENCY_LIMIT"
	ErrCodeHistoryPoolBusy         ErrorCode = "HISTORY_POOL_BUSY"
	ErrCodeTicksNotInPlan          ErrorCode = "TICKS_NOT_IN_PLAN"
	ErrCodeInternalError           ErrorCode = "INTERNAL_ERROR"
	ErrCodeAuthUnavailable         ErrorCode = "AUTH_UNAVAILABLE"
	ErrCodeUpstreamUnavailable     ErrorCode = "UPSTREAM_UNAVAILABLE"
	ErrCodeBarsBusy                ErrorCode = "BARS_BUSY"
)

// retryableErrorCodes is the set of error codes that may be retried with
// capped exponential backoff and jitter, subject to the retry policy.
var retryableErrorCodes = map[ErrorCode]bool{
	ErrCodeRateLimitBurst:          true,
	ErrCodeRateLimitSustained:      true,
	ErrCodeHistoryConcurrencyLimit: true,
	ErrCodeHistoryPoolBusy:         true,
	ErrCodeInternalError:           true,
	ErrCodeAuthUnavailable:         true,
	ErrCodeUpstreamUnavailable:     true,
	ErrCodeBarsBusy:                true,
}

// IsRetryable reports whether the error code is documented as retryable.
func (c ErrorCode) IsRetryable() bool { return retryableErrorCodes[c] }

// PaperAction is a Paper Trading signal action.
type PaperAction string

// Documented paper trading actions.
const (
	PaperBuy   PaperAction = "buy"
	PaperSell  PaperAction = "sell"
	PaperClose PaperAction = "close"
)

// PaperVenue is a Paper Trading market venue.
type PaperVenue string

// Documented paper trading venues.
const (
	PaperVenuePolymarket PaperVenue = "polymarket"
	PaperVenueKalshi     PaperVenue = "kalshi"
)
