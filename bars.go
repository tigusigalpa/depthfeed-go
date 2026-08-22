package depthfeed

import (
	"context"
	"net/url"
	"strings"
)

// BarsService provides access to /v3/bars OHLCV price bars for US-listed
// equities and ETFs, Binance spot pairs, and perpetual futures. One
// minute is the finest resolution; coarser frames are rolled up from it.
//
// Limits: history window (403), symbols per request (402
// SYMBOL_LIMIT_EXCEEDED: 3/8/16/32 by plan), per-request span (400
// WINDOW_TOO_WIDE), and 503 BARS_BUSY (retryable, carries
// details.retry_after). Page with limit and cursor to walk a wider range.
//
// Docs: https://depthfeed.com/docs#ep-bars
type BarsService struct {
	client *Client
}

// BarsOptions filters /v3/bars.
type BarsOptions struct {
	// Symbols is a comma-separated list (e.g. "AAPL,BTCUSDT"). Required.
	Symbols string
	// Timeframe is one of 1Min, 5Min, 15Min, 1Hour, 1Day (default 1Min).
	Timeframe BarsTimeframe
	// Venue optionally selects the market outright: us-equities, binance,
	// or binance-futures. Leave off for auto-resolution (equities first,
	// then Binance spot).
	Venue BarsVenue
	// StartTime bounds the window (ms-epoch or ISO-8601).
	StartTime Timestamp
	// EndTime bounds the window (ms-epoch or ISO-8601).
	EndTime Timestamp
	// Limit is bars per symbol. A request without one is not paged at all.
	Limit int
	// Cursor is the opaque keyset cursor from the prior page.
	Cursor string
}

// ApplyToValues writes bars options.
func (o *BarsOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "symbols", o.Symbols)
	applyStr(q, "timeframe", string(o.Timeframe))
	applyStr(q, "venue", string(o.Venue))
	if o.StartTime.IsSet() {
		q.Set("start_time", o.StartTime.String())
	}
	if o.EndTime.IsSet() {
		q.Set("end_time", o.EndTime.String())
	}
	applyInt(q, "limit", o.Limit)
	applyStr(q, "cursor", o.Cursor)
}

// Get returns multi-series OHLCV bars. The attribution string and each
// series' source must travel with the data if displayed. When paging,
// every series is cut at the same instant and next_cursor resumes exactly
// there: no overlap, no gap.
// Path: GET /v3/bars
func (s *BarsService) Get(ctx context.Context, opts *BarsOptions) (*BarsResponse, *Pagination, *ResponseMetadata, error) {
	if opts == nil || strings.TrimSpace(opts.Symbols) == "" {
		return nil, nil, nil, ErrEmptyPathSegment
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out BarsResponse
	var pg *Pagination
	meta, err := s.client.get(ctx, "/bars", q, &out, &pg)
	return &out, pg, meta, err
}
