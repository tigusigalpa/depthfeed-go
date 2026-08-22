package depthfeed

import (
	"context"
	"net/url"
)

// SnapshotsService provides access to Polymarket order-book snapshots.
// The order-book depth time-series for a market, with the underlying coin
// price stamped on each snapshot.
//
// Docs: https://depthfeed.com/docs#ep-snapshots
type SnapshotsService struct {
	client *Client
}

// SnapshotListOptions filters /v3/{coin}/markets/{market_id}/snapshots.
type SnapshotListOptions struct {
	ListOptions
	// IncludeOrderbook returns the full ladder (omitted by default for a
	// lighter payload).
	IncludeOrderbook bool
}

// ApplyToValues writes snapshot-specific + common options.
func (o *SnapshotListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	if o.IncludeOrderbook {
		q.Set("include_orderbook", "true")
	}
	o.ListOptions.ApplyToValues(q)
}

// List returns historical snapshots for a market.
// Path: GET /v3/{coin}/markets/{market_id}/snapshots
func (s *SnapshotsService) List(ctx context.Context, coin Coin, marketID string, opts *SnapshotListOptions) ([]Snapshot, *Pagination, *ResponseMetadata, error) {
	if !coin.IsValid() {
		return nil, nil, nil, ErrInvalidCoin
	}
	esc, err := pathEscape(marketID)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []Snapshot
	var pg *Pagination
	meta, err := s.client.get(ctx, "/"+string(coin)+"/markets/"+esc+"/snapshots", q, &out, &pg)
	return out, pg, meta, err
}

// AtTimestamp returns the single snapshot closest to a timestamp (within
// ±2s). Unlike the list endpoint, data is a single object with no
// pagination block; no match returns 404 SNAPSHOT_NOT_FOUND.
// Path: GET /v3/{coin}/markets/{market_id}/snapshots/at/{timestamp}
func (s *SnapshotsService) AtTimestamp(ctx context.Context, coin Coin, marketID string, ts Timestamp) (*Snapshot, *ResponseMetadata, error) {
	if !coin.IsValid() {
		return nil, nil, ErrInvalidCoin
	}
	esc, err := pathEscape(marketID)
	if err != nil {
		return nil, nil, err
	}
	if !ts.IsSet() {
		return nil, nil, ErrEmptyPathSegment
	}
	tsEsc, err := pathEscape(ts.String())
	if err != nil {
		return nil, nil, err
	}
	var out Snapshot
	meta, err := s.client.getSingle(ctx, "/"+string(coin)+"/markets/"+esc+"/snapshots/at/"+tsEsc, nil, &out)
	return &out, meta, err
}

// PolymarketAssetSnapshots returns depth history for a whole-venue market
// keyed by CLOB asset_id (token id). Use this for non-crypto Polymarket
// markets discovered via /v3/polymarket/markets.
// Path: GET /v3/polymarket/{asset_id}/snapshots
func (s *SnapshotsService) PolymarketAssetSnapshots(ctx context.Context, assetID string, opts *SnapshotListOptions) ([]Snapshot, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(assetID)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []Snapshot
	var pg *Pagination
	meta, err := s.client.get(ctx, "/polymarket/"+esc+"/snapshots", q, &out, &pg)
	return out, pg, meta, err
}
