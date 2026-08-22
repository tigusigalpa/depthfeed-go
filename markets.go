package depthfeed

import (
	"context"
	"net/url"
)

// MarketsService provides access to Polymarket market discovery.
// {coin} is one of btc, eth, sol, xrp, doge, bnb, hype.
//
// Docs: https://depthfeed.com/docs#ep-markets
type MarketsService struct {
	client *Client
}

// MarketListOptions filters /v3/{coin}/markets.
type MarketListOptions struct {
	ListOptions
	// Type filters by window (5m/15m/1h/4h/24h).
	Type string
	// Resolved filters by resolved state (tri-state: nil = omit).
	Resolved *bool
}

// ApplyToValues writes market-specific + common options.
func (o *MarketListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "type", o.Type)
	applyBoolFilter(q, "resolved", o.Resolved)
	o.ListOptions.ApplyToValues(q)
}

// List returns Polymarket up/down markets for a coin.
// Path: GET /v3/{coin}/markets
func (s *MarketsService) List(ctx context.Context, coin Coin, opts *MarketListOptions) ([]Market, *Pagination, *ResponseMetadata, error) {
	if !coin.IsValid() {
		return nil, nil, nil, ErrInvalidCoin
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []Market
	var pg *Pagination
	meta, err := s.client.get(ctx, "/"+string(coin)+"/markets", q, &out, &pg)
	return out, pg, meta, err
}

// Get retrieves a single market by id.
// Path: GET /v3/{coin}/markets/{market_id}
func (s *MarketsService) Get(ctx context.Context, coin Coin, marketID string) (*Market, *ResponseMetadata, error) {
	if !coin.IsValid() {
		return nil, nil, ErrInvalidCoin
	}
	esc, err := pathEscape(marketID)
	if err != nil {
		return nil, nil, err
	}
	var out Market
	meta, err := s.client.getSingle(ctx, "/"+string(coin)+"/markets/"+esc, nil, &out)
	return &out, meta, err
}

// GetBySlug retrieves a single market by slug.
// Path: GET /v3/{coin}/markets/by-slug/{slug}
func (s *MarketsService) GetBySlug(ctx context.Context, coin Coin, slug string) (*Market, *ResponseMetadata, error) {
	if !coin.IsValid() {
		return nil, nil, ErrInvalidCoin
	}
	esc, err := pathEscape(slug)
	if err != nil {
		return nil, nil, err
	}
	var out Market
	meta, err := s.client.getSingle(ctx, "/"+string(coin)+"/markets/by-slug/"+esc, nil, &out)
	return &out, meta, err
}

// PolymarketVenueListOptions filters the whole-venue catalogue.
type PolymarketVenueListOptions struct {
	ListOptions
	// Q is a slug search query.
	Q string
	// Type filters by market family; "event" selects everything that isn't
	// an up/down ladder.
	Type string
	// Coin filters by base asset.
	Coin string
	// Event filters by event id.
	Event string
	// Outcome filters by outcome.
	Outcome string
}

// ApplyToValues writes venue-specific + common options.
func (o *PolymarketVenueListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "q", o.Q)
	applyStr(q, "type", o.Type)
	applyStr(q, "coin", o.Coin)
	applyStr(q, "event", o.Event)
	applyStr(q, "outcome", o.Outcome)
	o.ListOptions.ApplyToValues(q)
}

// PolymarketVenue returns the whole Polymarket venue catalogue (paid plan).
// Markets here are keyed by CLOB asset_id (token id); pass that to
// /v3/polymarket/{asset_id}/snapshots or /ticks. Non-crypto markets carry
// an empty base_asset.
// Path: GET /v3/polymarket/markets
func (s *MarketsService) PolymarketVenue(ctx context.Context, opts *PolymarketVenueListOptions) ([]PolymarketVenueMarket, *Pagination, *ResponseMetadata, error) {
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []PolymarketVenueMarket
	var pg *Pagination
	meta, err := s.client.get(ctx, "/polymarket/markets", q, &out, &pg)
	return out, pg, meta, err
}
