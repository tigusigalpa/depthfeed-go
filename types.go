package depthfeed

import "encoding/json"

// Health is the /v3/health liveness response.
type Health struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Time    string `json:"time"`
}

// WhoAmI is the /v3/whoami identity/plan response.
type WhoAmI struct {
	UserID      string   `json:"user_id"`
	Plan        string   `json:"plan"`
	RPS         int      `json:"rps"`
	RPM         int      `json:"rpm"`
	HistoryDays int      `json:"history_days"`
	Coins       string   `json:"coins"`
	Sports      bool     `json:"sports"`
	Bars        BarsPlan `json:"bars"`
}

// BarsPlan is the bars entitlement block inside WhoAmI.
type BarsPlan struct {
	MaxSymbolsPerRequest int `json:"max_symbols_per_request"`
}

// Market is a Polymarket up/down market per coin. Settlement fields are
// null until the market settles. Per-coin price keys are dynamic (e.g.
// btc_price_start); unknown fields are preserved in Raw.
type Market struct {
	MarketID       string   `json:"market_id"`
	EventID        string   `json:"event_id"`
	Slug           string   `json:"slug"`
	MarketType     string   `json:"market_type"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	ConditionID    string   `json:"condition_id"`
	ClobTokenUp    string   `json:"clob_token_up"`
	ClobTokenDown  string   `json:"clob_token_down"`
	Winner         *string  `json:"winner"`
	FinalVolume    *float64 `json:"final_volume"`
	FinalLiquidity *float64 `json:"final_liquidity"`
	ResolvedAt     *string  `json:"resolved_at"`
	// Raw preserves unknown/dynamic fields (e.g. btc_price_start/end).
	Raw json.RawMessage `json:"-"`
}

// PolymarketVenueMarket is a whole-venue Polymarket market keyed by CLOB
// asset_id. Non-crypto markets carry an empty base_asset.
type PolymarketVenueMarket struct {
	AssetID     string          `json:"asset_id"`
	ConditionID string          `json:"condition_id"`
	Slug        string          `json:"slug"`
	BaseAsset   string          `json:"base_asset"`
	MarketType  string          `json:"market_type"`
	Outcomes    []string        `json:"outcomes"`
	Event       string          `json:"event"`
	Raw         json.RawMessage `json:"-"`
}

// Snapshot is a Polymarket order-book snapshot with the underlying coin
// price stamped on it. Levels are [price, size].
type Snapshot struct {
	ID            string     `json:"id"`
	Time          string     `json:"time"`
	MarketID      string     `json:"market_id"`
	PriceUp       float64    `json:"price_up"`
	PriceDown     float64    `json:"price_down"`
	OrderbookUp   *OrderBook `json:"orderbook_up,omitempty"`
	OrderbookDown *OrderBook `json:"orderbook_down,omitempty"`
	// CoinPrice is the dynamic per-coin price key (e.g. btc_price). It is
	// decoded into the dynamic field via custom unmarshaling; see
	// SnapshotRaw for the full raw object.
	CoinPrice float64         `json:"-"`
	Raw       json.RawMessage `json:"-"`
}

// PolymarketTick is a single Polymarket book-change tick.
type PolymarketTick struct {
	ID         string  `json:"id"`
	Time       string  `json:"time"`
	ReceivedMs string  `json:"received_ms"`
	Outcome    string  `json:"outcome"`
	TokenID    string  `json:"token_id"`
	Side       string  `json:"side"`
	Price      float64 `json:"price"`
	Size       float64 `json:"size"`
}

// KalshiTick is a single Kalshi book-change tick.
type KalshiTick struct {
	ID         string  `json:"id"`
	Time       string  `json:"time"`
	ReceivedMs string  `json:"received_ms"`
	Seq        string  `json:"seq"`
	Kind       string  `json:"kind"`
	Side       string  `json:"side"`
	Price      float64 `json:"price"`
	Delta      float64 `json:"delta"`
	Size       float64 `json:"size"`
}

// BinanceBook is a Binance spot/futures latest book or snapshot row.
type BinanceBook struct {
	Price     float64      `json:"price"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
	Timestamp string       `json:"timestamp"`
	ID        string       `json:"id"`
}

// BinanceTrade is a 1-second OHLCV candle with buy/sell split.
type BinanceTrade struct {
	ID                   string  `json:"id"`
	Timestamp            string  `json:"timestamp"`
	PriceOpen            float64 `json:"price_open"`
	PriceHigh            float64 `json:"price_high"`
	PriceLow             float64 `json:"price_low"`
	PriceClose           float64 `json:"price_close"`
	TotalVolume          float64 `json:"total_volume"`
	AggressiveBuyVolume  float64 `json:"aggressive_buy_volume"`
	AggressiveSellVolume float64 `json:"aggressive_sell_volume"`
	NumTrades            int     `json:"num_trades"`
}

// BarsResponse is the /v3/bars multi-series response.
type BarsResponse struct {
	Timeframe    string      `json:"timeframe"`
	LicenseClass string      `json:"license_class"`
	Attribution  string      `json:"attribution"`
	Series       []BarSeries `json:"series"`
}

// BarSeries is one symbol's OHLCV series within a bars response. The
// attribution string and per-series source must travel with the data if
// displayed.
type BarSeries struct {
	Symbol     string    `json:"symbol"`
	Timeframe  string    `json:"timeframe"`
	AssetClass string    `json:"assetClass"`
	Source     string    `json:"source"`
	Derived    bool      `json:"derived"`
	T          []int64   `json:"t"`
	Open       []float64 `json:"open"`
	High       []float64 `json:"high"`
	Low        []float64 `json:"low"`
	Close      []float64 `json:"close"`
	Volume     []float64 `json:"volume"`
}

// KalshiMarket is a Kalshi crypto contract market.
type KalshiMarket struct {
	Ticker      string   `json:"ticker"`
	EventTicker string   `json:"event_ticker"`
	Series      string   `json:"series"`
	BaseAsset   string   `json:"base_asset"`
	MarketType  string   `json:"market_type"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Strike      *float64 `json:"strike"`
	OpenTime    string   `json:"open_time"`
	CloseTime   string   `json:"close_time"`
}

// KalshiLatestBook is the newest full yes/no book for a ticker.
type KalshiLatestBook struct {
	Ticker     string       `json:"ticker"`
	Series     string       `json:"series"`
	BaseAsset  string       `json:"base_asset"`
	MarketType string       `json:"market_type"`
	ID         string       `json:"id"`
	Timestamp  string       `json:"timestamp"`
	Yes        []PriceLevel `json:"yes"`
	No         []PriceLevel `json:"no"`
}

// KalshiSnapshot is a historical Kalshi yes/no depth row.
type KalshiSnapshot struct {
	Ticker     string       `json:"ticker"`
	Series     string       `json:"series"`
	BaseAsset  string       `json:"base_asset"`
	MarketType string       `json:"market_type"`
	ID         string       `json:"id"`
	Timestamp  string       `json:"timestamp"`
	Yes        []PriceLevel `json:"yes"`
	No         []PriceLevel `json:"no"`
}

// KalshiCandle is an hourly Kalshi OHLC candle of yes bid/ask and price.
type KalshiCandle struct {
	Ticker        string      `json:"ticker"`
	Series        string      `json:"series"`
	MarketType    string      `json:"market_type"`
	ID            string      `json:"id"`
	Timestamp     string      `json:"timestamp"`
	PeriodMinutes int         `json:"period_minutes"`
	YesBid        *CandleOHLC `json:"yes_bid"`
	YesAsk        *CandleOHLC `json:"yes_ask"`
	Price         *CandleOHLC `json:"price"`
	Volume        float64     `json:"volume"`
	OpenInterest  float64     `json:"open_interest"`
}

// CandleOHLC is an OHLC quad.
type CandleOHLC struct {
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

// LimitlessMarket is a Limitless Exchange up/down market.
type LimitlessMarket struct {
	Slug         string `json:"slug"`
	ConditionID  string `json:"condition_id"`
	BaseAsset    string `json:"base_asset"`
	MarketType   string `json:"market_type"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	ExpirationMs string `json:"expiration_ms"`
}

// LimitlessLatestBook is the newest L2 depth for a slug.
type LimitlessLatestBook struct {
	Slug        string       `json:"slug"`
	ConditionID string       `json:"condition_id"`
	BaseAsset   string       `json:"base_asset"`
	MarketType  string       `json:"market_type"`
	ID          string       `json:"id"`
	Timestamp   string       `json:"timestamp"`
	Midpoint    float64      `json:"midpoint"`
	Bids        []PriceLevel `json:"bids"`
	Asks        []PriceLevel `json:"asks"`
}

// LimitlessSnapshot is a historical Limitless depth row.
type LimitlessSnapshot struct {
	Slug        string       `json:"slug"`
	ConditionID string       `json:"condition_id"`
	BaseAsset   string       `json:"base_asset"`
	MarketType  string       `json:"market_type"`
	ID          string       `json:"id"`
	Timestamp   string       `json:"timestamp"`
	Midpoint    float64      `json:"midpoint"`
	Bids        []PriceLevel `json:"bids"`
	Asks        []PriceLevel `json:"asks"`
}

// BinancePredictionMarket is a Binance Wallet prediction market.
type BinancePredictionMarket struct {
	MarketID       string          `json:"market_id"`
	TopicID        string          `json:"topic_id"`
	ConditionID    string          `json:"condition_id"`
	BaseAsset      string          `json:"base_asset"`
	MarketType     string          `json:"market_type"`
	Status         string          `json:"status"`
	WinningOutcome *string         `json:"winning_outcome"`
	Volume         *float64        `json:"volume"`
	Liquidity      *float64        `json:"liquidity"`
	Participants   *int            `json:"participants"`
	Chain          string          `json:"chain"`
	Raw            json.RawMessage `json:"-"`
}

// BinancePredictionBook is a Binance prediction full L2 depth row.
type BinancePredictionBook struct {
	MarketID  string       `json:"market_id"`
	ID        string       `json:"id"`
	Timestamp string       `json:"timestamp"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
}

// BinancePredictionTimeSeries is a typed probability/price/last-trade series.
type BinancePredictionTimeSeries struct {
	MarketID string            `json:"market_id"`
	Metric   string            `json:"metric"`
	Points   []TimeSeriesPoint `json:"points"`
	Raw      json.RawMessage   `json:"-"`
}

// TimeSeriesPoint is a single (timestamp, value) sample.
type TimeSeriesPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

// BinancePredictionTrade is a single Binance prediction trade.
type BinancePredictionTrade struct {
	ID        string  `json:"id"`
	Timestamp string  `json:"timestamp"`
	Price     float64 `json:"price"`
	Size      float64 `json:"size"`
	Side      string  `json:"side"`
}

// ScreenerGrid is the /v3/screener cross-venue grid.
type ScreenerGrid struct {
	Assets  []string       `json:"assets"`
	Windows []string       `json:"windows"`
	Cells   []ScreenerCell `json:"cells"`
	Paid    bool           `json:"paid"`
}

// ScreenerCell is one asset's cross-venue up/down probabilities.
type ScreenerCell struct {
	Asset      string             `json:"asset"`
	Polymarket *ScreenerVenueCell `json:"polymarket,omitempty"`
	Kalshi     *ScreenerVenueCell `json:"kalshi,omitempty"`
	Limitless  *ScreenerVenueCell `json:"limitless,omitempty"`
}

// ScreenerVenueCell is one venue's latest up/down summary.
type ScreenerVenueCell struct {
	Up          float64  `json:"up"`
	Down        float64  `json:"down"`
	BestBid     *float64 `json:"best_bid"`
	BestAsk     *float64 `json:"best_ask"`
	PriceToBeat *float64 `json:"price_to_beat"`
	Slug        string   `json:"slug,omitempty"`
}

// ScreenerDrilldown is the /v3/screener/{asset}/{window} full-L2 drill-down.
type ScreenerDrilldown struct {
	Asset  string          `json:"asset"`
	Window string          `json:"window"`
	Books  ScreenerBooks   `json:"books"`
	Raw    json.RawMessage `json:"-"`
}

// ScreenerBooks holds the full L2 book on every venue side by side.
type ScreenerBooks struct {
	Polymarket json.RawMessage `json:"polymarket,omitempty"`
	Kalshi     json.RawMessage `json:"kalshi,omitempty"`
	Limitless  json.RawMessage `json:"limitless,omitempty"`
}

// Overview is the public /v3/overview per-asset time-series overview.
type Overview struct {
	Assets []json.RawMessage `json:"assets"`
	Raw    json.RawMessage   `json:"-"`
}

// BatchRequest is the POST /v3/batch body.
type BatchRequest struct {
	Requests []BatchSubrequest `json:"requests"`
}

// BatchSubrequest is one GET /v3/ sub-request.
type BatchSubrequest struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

// BatchSubresponse is one per-subresponse result.
type BatchSubresponse struct {
	ID     string          `json:"id"`
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

// BatchResponse is the outer batch response. The envelope's `data` field
// is the array of per-subresponse results.
type BatchResponse struct {
	Responses []BatchSubresponse `json:"responses"`
}

// --- Sports types ---

// SportsOverview is the live cross-sport board.
type SportsOverview struct {
	Leagues []json.RawMessage `json:"leagues"`
	Raw     json.RawMessage   `json:"-"`
}

// SportsLeague is a tracked league with counts.
type SportsLeague struct {
	League      string `json:"league"`
	Name        string `json:"name"`
	GameCount   int    `json:"game_count"`
	MarketCount int    `json:"market_count"`
}

// SportsMarket is a normalized sports market across Polymarket and Kalshi.
type SportsMarket struct {
	ConditionID string   `json:"condition_id"`
	MarketSlug  string   `json:"market_slug"`
	League      string   `json:"league"`
	Kind        string   `json:"kind"`
	Title       string   `json:"title"`
	Outcomes    []string `json:"outcomes"`
	Tokens      []string `json:"tokens"`
	GameStart   string   `json:"game_start"`
	Resolved    bool     `json:"resolved"`
}

// SportsBook is a sports outcome order book (latest or history row).
type SportsBook struct {
	AssetID   string       `json:"asset_id"`
	ID        string       `json:"id"`
	Timestamp string       `json:"timestamp"`
	Bids      []PriceLevel `json:"bids"`
	Asks      []PriceLevel `json:"asks"`
}

// SportsGame is a league game with linked markets.
type SportsGame struct {
	GameSlug string          `json:"game_slug"`
	League   string          `json:"league"`
	Raw      json.RawMessage `json:"-"`
}

// SportsMarketSingle is one normalized market with venue identifiers.
type SportsMarketSingle struct {
	ConditionID string          `json:"condition_id"`
	Outcomes    []string        `json:"outcomes"`
	Tokens      []string        `json:"tokens"`
	Raw         json.RawMessage `json:"-"`
}

// SportsPlayer is a league athlete directory entry.
type SportsPlayer struct {
	AthleteID string `json:"athlete_id"`
	Name      string `json:"name"`
	League    string `json:"league"`
	Team      string `json:"team"`
}

// SportsPlayerProfile is a player profile.
type SportsPlayerProfile struct {
	AthleteID string          `json:"athlete_id"`
	Raw       json.RawMessage `json:"-"`
}

// SportsOddsLine is a normalized live odds line.
type SportsOddsLine struct {
	Raw json.RawMessage `json:"-"`
}

// SportsBookCatalogEntry is a sportsbook catalogue/freshness entry.
type SportsBookCatalogEntry struct {
	Book       string `json:"book"`
	Fresh      bool   `json:"fresh"`
	LastUpdate string `json:"last_update"`
}

// SportsOddsHistory is a historical sportsbook line.
type SportsOddsHistory struct {
	Raw json.RawMessage `json:"-"`
}

// --- Sports webhook types ---

// SportsWebhookCreate is the POST /v3/sports/webhooks body.
type SportsWebhookCreate struct {
	URL      string   `json:"url"`
	Leagues  []string `json:"leagues"`
	Statuses []string `json:"statuses"`
}

// SportsWebhook is a subscription with delivery health.
type SportsWebhook struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Leagues   []string `json:"leagues"`
	Statuses  []string `json:"statuses"`
	CreatedAt string   `json:"created_at"`
	// SigningSecret is shown once at creation; never log or persist it
	// unless the application explicitly asks.
	SigningSecret string `json:"signing_secret,omitempty"`
}

// --- Paper Trading types ---

// PaperSignal is the POST /paper/hook/{token} body. Fills always price
// from the live book at signal time; a payload can never set its own
// price. Use ClientID for idempotent client-side retry handling.
type PaperSignal struct {
	Action   PaperAction `json:"action"`
	Venue    PaperVenue  `json:"venue"`
	Market   string      `json:"market"`
	Side     string      `json:"side,omitempty"`
	USD      *float64    `json:"usd,omitempty"`
	Shares   *float64    `json:"shares,omitempty"`
	ClientID string      `json:"client_id,omitempty"`
}

// PaperFill is the Paper Trading signal response.
type PaperFill struct {
	Filled     bool    `json:"filled"`
	Action     string  `json:"action"`
	Price      float64 `json:"price"`
	Shares     float64 `json:"shares"`
	CostCents  int     `json:"cost_cents"`
	CashCents  int     `json:"cash_cents"`
	PositionID string  `json:"position_id"`
}
