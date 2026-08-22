package depthfeed

import (
	"context"
	"net/url"
)

// TicksService provides access to the Polymarket and Kalshi tick tapes
// (every individual order-book change). Enterprise only; other plans
// receive 402 TICKS_NOT_IN_PLAN.
//
// Docs: https://depthfeed.com/docs#ep-ticks
type TicksService struct {
	client *Client
}

// TickListOptions filters the tick tapes.
type TickListOptions struct {
	ListOptions
	// Outcome narrows Polymarket ticks to "up" or "down".
	Outcome string
	// Side narrows Kalshi ticks to "yes" or "no".
	Side string
}

// ApplyToValues writes tick-specific + common options.
func (o *TickListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "outcome", o.Outcome)
	applyStr(q, "side", o.Side)
	o.ListOptions.ApplyToValues(q)
}

// Polymarket returns the Polymarket tick tape for a market (enterprise).
// Path: GET /v3/{coin}/markets/{market_id}/ticks
func (s *TicksService) Polymarket(ctx context.Context, coin Coin, marketID string, opts *TickListOptions) ([]PolymarketTick, *Pagination, *ResponseMetadata, error) {
	if !coin.IsValid() {
		return nil, nil, nil, ErrInvalidCoin
	}
	esc, err := pathEscape(marketID)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []PolymarketTick
	var pg *Pagination
	meta, err := s.client.get(ctx, "/"+string(coin)+"/markets/"+esc+"/ticks", q, &out, &pg)
	return out, pg, meta, err
}

// PolymarketAssetTicks returns the tick tape for a whole-venue market
// keyed by CLOB asset_id.
// Path: GET /v3/polymarket/{asset_id}/ticks
func (s *TicksService) PolymarketAssetTicks(ctx context.Context, assetID string, opts *TickListOptions) ([]PolymarketTick, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(assetID)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []PolymarketTick
	var pg *Pagination
	meta, err := s.client.get(ctx, "/polymarket/"+esc+"/ticks", q, &out, &pg)
	return out, pg, meta, err
}

// Kalshi returns the Kalshi tick tape for a ticker (enterprise). Kalshi
// tick capture began 2026-08-07; there is no Kalshi tick history before
// that date. Never pass a series (e.g. KXBTC15M) where an exact discovered
// market ticker is required.
// Path: GET /v3/kalshi/{ticker}/ticks
func (s *TicksService) Kalshi(ctx context.Context, ticker string, opts *TickListOptions) ([]KalshiTick, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(ticker)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []KalshiTick
	var pg *Pagination
	meta, err := s.client.get(ctx, "/kalshi/"+esc+"/ticks", q, &out, &pg)
	return out, pg, meta, err
}
