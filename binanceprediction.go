package depthfeed

import (
	"context"
	"net/url"
)

// BinancePredictionService provides access to Binance Wallet's
// Predict.fun-backed crypto prediction markets, normalized into a
// dedicated DepthFeed namespace. Paid plans only.
//
// Docs: https://depthfeed.com/docs#ep-binance-prediction
type BinancePredictionService struct {
	client *Client
}

// BinancePredictionMarketListOptions filters /v3/binance-prediction/markets.
type BinancePredictionMarketListOptions struct {
	ListOptions
	// Coin filters by base asset (e.g. btc).
	Coin string
	// Type filters by window (e.g. 5m).
	Type string
	// Status filters by status (e.g. resolved).
	Status string
}

// ApplyToValues writes binance-prediction-market + common options.
func (o *BinancePredictionMarketListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "coin", o.Coin)
	applyStr(q, "type", o.Type)
	applyStr(q, "status", o.Status)
	o.ListOptions.ApplyToValues(q)
}

// Markets lists Binance prediction markets (paid plan).
// Path: GET /v3/binance-prediction/markets
func (s *BinancePredictionService) Markets(ctx context.Context, opts *BinancePredictionMarketListOptions) ([]BinancePredictionMarket, *Pagination, *ResponseMetadata, error) {
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []BinancePredictionMarket
	var pg *Pagination
	meta, err := s.client.get(ctx, "/binance-prediction/markets", q, &out, &pg)
	return out, pg, meta, err
}

// LatestBook returns the latest full L2 depth (paid plan).
// Path: GET /v3/binance-prediction/{market_id}/orderbook/latest
func (s *BinancePredictionService) LatestBook(ctx context.Context, marketID string) (*BinancePredictionBook, *ResponseMetadata, error) {
	esc, err := pathEscape(marketID)
	if err != nil {
		return nil, nil, err
	}
	var out BinancePredictionBook
	meta, err := s.client.getSingle(ctx, "/binance-prediction/"+esc+"/orderbook/latest", nil, &out)
	return &out, meta, err
}

// Snapshots returns historical full L2 depth (paid plan).
// Path: GET /v3/binance-prediction/{market_id}/snapshots
func (s *BinancePredictionService) Snapshots(ctx context.Context, marketID string, opts *ListOptions) ([]BinancePredictionBook, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(marketID)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []BinancePredictionBook
	var pg *Pagination
	meta, err := s.client.get(ctx, "/binance-prediction/"+esc+"/snapshots", q, &out, &pg)
	return out, pg, meta, err
}

// TimeSeries returns a typed probability/price/last-trade time series
// (paid plan). metric is one of chance, price, last_trade.
// Path: GET /v3/binance-prediction/{market_id}/timeseries?metric={metric}
func (s *BinancePredictionService) TimeSeries(ctx context.Context, marketID string, metric PredictionMetric, opts *ListOptions) (*BinancePredictionTimeSeries, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(marketID)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	q.Set("metric", string(metric))
	opts.ApplyToValues(q)
	var out BinancePredictionTimeSeries
	var pg *Pagination
	meta, err := s.client.get(ctx, "/binance-prediction/"+esc+"/timeseries", q, &out, &pg)
	return &out, pg, meta, err
}

// Trades returns the typed trades collection (paid plan). Reserved while
// historical fills are reconstructed; currently returns an empty dataset.
// Path: GET /v3/binance-prediction/{market_id}/trades
func (s *BinancePredictionService) Trades(ctx context.Context, marketID string, opts *ListOptions) ([]BinancePredictionTrade, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(marketID)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []BinancePredictionTrade
	var pg *Pagination
	meta, err := s.client.get(ctx, "/binance-prediction/"+esc+"/trades", q, &out, &pg)
	return out, pg, meta, err
}
