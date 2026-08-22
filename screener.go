package depthfeed

import "context"

// ScreenerService provides access to the cross-venue screener — start
// here for anything cross-venue. It answers how the same "up or down"
// market is priced on Polymarket, Kalshi, and Limitless right now in a
// single request.
//
// Docs: https://depthfeed.com/docs#ep-screener
type ScreenerService struct {
	client *Client
}

// Grid returns the cross-venue grid: one cell per asset with each venue's
// latest "up" probability, best bid/ask, and settlement reference. A
// placeholder/empty book reports null rather than a meaningless 0.50.
// Path: GET /v3/screener
func (s *ScreenerService) Grid(ctx context.Context) (*ScreenerGrid, *ResponseMetadata, error) {
	var out ScreenerGrid
	meta, err := s.client.getSingle(ctx, "/screener", nil, &out)
	return &out, meta, err
}

// Drilldown returns the full L2 book on every venue side by side for one
// (asset, window), plus the underlying spot series and the settlement
// price_to_beat. asset is one of the seven coins; window is 5m/15m/1h/4h/24h.
// Path: GET /v3/screener/{asset}/{window}
func (s *ScreenerService) Drilldown(ctx context.Context, asset Coin, window MarketWindow) (*ScreenerDrilldown, *ResponseMetadata, error) {
	if !asset.IsValid() {
		return nil, nil, ErrInvalidCoin
	}
	escWindow, err := pathEscape(string(window))
	if err != nil {
		return nil, nil, err
	}
	var out ScreenerDrilldown
	meta, err := s.client.getSingle(ctx, "/screener/"+string(asset)+"/"+escWindow, nil, &out)
	return &out, meta, err
}
