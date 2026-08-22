package depthfeed

import (
	"context"
	"net/url"
)

// BinanceService provides access to Binance spot & futures books and
// trades. {venue} is restricted to spot or futures. The book is 20 levels
// per side.
//
// Docs: https://depthfeed.com/docs#ep-binance
type BinanceService struct {
	client *Client
}

// coinVenuePath builds /{coin}/{venue} after validating both segments.
func coinVenuePath(coin Coin, venue Venue) (string, error) {
	if !coin.IsValid() {
		return "", ErrInvalidCoin
	}
	if !venue.IsValid() {
		return "", ErrInvalidVenue
	}
	return "/" + string(coin) + "/" + string(venue), nil
}

// Latest returns the current book.
// Path: GET /v3/{coin}/{venue}/latest
func (s *BinanceService) Latest(ctx context.Context, coin Coin, venue Venue) (*BinanceBook, *ResponseMetadata, error) {
	base, err := coinVenuePath(coin, venue)
	if err != nil {
		return nil, nil, err
	}
	var out BinanceBook
	meta, err := s.client.getSingle(ctx, base+"/latest", nil, &out)
	return &out, meta, err
}

// Snapshots returns paginated book history.
// Path: GET /v3/{coin}/{venue}/snapshots
func (s *BinanceService) Snapshots(ctx context.Context, coin Coin, venue Venue, opts *ListOptions) ([]BinanceBook, *Pagination, *ResponseMetadata, error) {
	base, err := coinVenuePath(coin, venue)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []BinanceBook
	var pg *Pagination
	meta, err := s.client.get(ctx, base+"/snapshots", q, &out, &pg)
	return out, pg, meta, err
}

// Trades returns the paginated 1-second OHLCV tape.
// Path: GET /v3/{coin}/{venue}/trades
func (s *BinanceService) Trades(ctx context.Context, coin Coin, venue Venue, opts *ListOptions) ([]BinanceTrade, *Pagination, *ResponseMetadata, error) {
	base, err := coinVenuePath(coin, venue)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []BinanceTrade
	var pg *Pagination
	meta, err := s.client.get(ctx, base+"/trades", q, &out, &pg)
	return out, pg, meta, err
}

// LatestTrade returns the latest 1-second OHLCV data.
// Path: GET /v3/{coin}/{venue}/trades/latest
func (s *BinanceService) LatestTrade(ctx context.Context, coin Coin, venue Venue) (*BinanceTrade, *ResponseMetadata, error) {
	base, err := coinVenuePath(coin, venue)
	if err != nil {
		return nil, nil, err
	}
	var out BinanceTrade
	meta, err := s.client.getSingle(ctx, base+"/trades/latest", nil, &out)
	return &out, meta, err
}
