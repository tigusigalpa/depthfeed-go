package depthfeed

import (
	"context"
	"net/url"
)

// LimitlessService provides access to Limitless Exchange — a CLOB on Base
// running recurring up/down crypto markets at 5m and 15m. Paid plans only.
// {slug} looks like btc-up-or-down-15-min-1780798512077.
//
// Docs: https://depthfeed.com/docs#ep-limitless
type LimitlessService struct {
	client *Client
}

// LimitlessMarketListOptions filters /v3/limitless/markets.
type LimitlessMarketListOptions struct {
	ListOptions
	// Coin filters by base asset.
	Coin string
	// Type filters by window (5m/15m).
	Type string
}

// ApplyToValues writes limitless-market + common options.
func (o *LimitlessMarketListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "coin", o.Coin)
	applyStr(q, "type", o.Type)
	o.ListOptions.ApplyToValues(q)
}

// Markets lists currently-open Limitless slots (paid plan).
// Path: GET /v3/limitless/markets
func (s *LimitlessService) Markets(ctx context.Context, opts *LimitlessMarketListOptions) ([]LimitlessMarket, *Pagination, *ResponseMetadata, error) {
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []LimitlessMarket
	var pg *Pagination
	meta, err := s.client.get(ctx, "/limitless/markets", q, &out, &pg)
	return out, pg, meta, err
}

// LatestBook returns the newest L2 depth for a slug (bids descending,
// asks ascending) (paid plan).
// Path: GET /v3/limitless/{slug}/orderbook/latest
func (s *LimitlessService) LatestBook(ctx context.Context, slug string) (*LimitlessLatestBook, *ResponseMetadata, error) {
	esc, err := pathEscape(slug)
	if err != nil {
		return nil, nil, err
	}
	var out LimitlessLatestBook
	meta, err := s.client.getSingle(ctx, "/limitless/"+esc+"/orderbook/latest", nil, &out)
	return &out, meta, err
}

// Snapshots returns the historical depth series for one slug (paid plan).
// Path: GET /v3/limitless/{slug}/snapshots
func (s *LimitlessService) Snapshots(ctx context.Context, slug string, opts *ListOptions) ([]LimitlessSnapshot, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(slug)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []LimitlessSnapshot
	var pg *Pagination
	meta, err := s.client.get(ctx, "/limitless/"+esc+"/snapshots", q, &out, &pg)
	return out, pg, meta, err
}
