package depthfeed

import (
	"context"
	"net/url"
)

// KalshiService provides access to Kalshi crypto contracts — full yes/no
// depth. Paid plans only. 21 series for all 7 assets. Never coerce a
// Kalshi series into a ticker: discovery must precede a ticker-specific
// call.
//
// Docs: https://depthfeed.com/docs#ep-kalshi
type KalshiService struct {
	client *Client
}

// KalshiMarketListOptions filters /v3/kalshi/markets.
type KalshiMarketListOptions struct {
	ListOptions
	// Coin filters by base asset.
	Coin string
	// Type filters by window.
	Type string
	// Series is an exact series filter (e.g. KXBTC15M).
	Series string
}

// ApplyToValues writes kalshi-market + common options.
func (o *KalshiMarketListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "coin", o.Coin)
	applyStr(q, "type", o.Type)
	applyStr(q, "series", o.Series)
	o.ListOptions.ApplyToValues(q)
}

// Markets lists Kalshi markets. For historical downloads, pass
// start_time/end_time here first and use the returned venue-native
// tickers. Do not generate Kalshi ticker ids from UTC timestamps.
// Path: GET /v3/kalshi/markets
func (s *KalshiService) Markets(ctx context.Context, opts *KalshiMarketListOptions) ([]KalshiMarket, *Pagination, *ResponseMetadata, error) {
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []KalshiMarket
	var pg *Pagination
	meta, err := s.client.get(ctx, "/kalshi/markets", q, &out, &pg)
	return out, pg, meta, err
}

// LatestBook returns the newest full yes/no book for a ticker (paid plan).
// yes/no are [price, size] arrays (prices in 0–1 dollars).
// Path: GET /v3/kalshi/{ticker}/orderbook/latest
func (s *KalshiService) LatestBook(ctx context.Context, ticker string) (*KalshiLatestBook, *ResponseMetadata, error) {
	esc, err := pathEscape(ticker)
	if err != nil {
		return nil, nil, err
	}
	var out KalshiLatestBook
	meta, err := s.client.getSingle(ctx, "/kalshi/"+esc+"/orderbook/latest", nil, &out)
	return &out, meta, err
}

// Snapshots returns the historical yes/no depth series for one exact
// discovered market ticker. Passing a series (e.g. KXBTC15M) or an
// invented historical ticker returns an explicit discovery error instead
// of a misleading empty page.
// Path: GET /v3/kalshi/{ticker}/snapshots
func (s *KalshiService) Snapshots(ctx context.Context, ticker string, opts *ListOptions) ([]KalshiSnapshot, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(ticker)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []KalshiSnapshot
	var pg *Pagination
	meta, err := s.client.get(ctx, "/kalshi/"+esc+"/snapshots", q, &out, &pg)
	return out, pg, meta, err
}

// Candles returns hourly OHLC of the yes bid, yes ask, and traded price,
// plus volume and open_interest. Only 1h is recorded; any other interval
// returns an explicit error. Candles are stored as recorded and never
// resampled. price is null when nothing traded in the period.
// Path: GET /v3/kalshi/{ticker}/candles
func (s *KalshiService) Candles(ctx context.Context, ticker string, opts *ListOptions) ([]KalshiCandle, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(ticker)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []KalshiCandle
	var pg *Pagination
	meta, err := s.client.get(ctx, "/kalshi/"+esc+"/candles", q, &out, &pg)
	return out, pg, meta, err
}
