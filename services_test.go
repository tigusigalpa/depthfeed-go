package depthfeed

import (
	"context"
	"net/http"
	"testing"

	"github.com/tigusigalpa/depthfeed-go/internal/testutil"
)

// endpointCase is one (service call → expected method+path) assertion.
type endpointCase struct {
	name   string
	method string
	path   string // path prefix (no query) the server should see
	call   func(c *Client)
}

// endpointCases enumerizes every documented REST route so path drift is
// easy to spot. Each case registers a handler on the expected path and
// invokes the service method; the mock records the actual method+path.
func endpointCases() []endpointCase {
	return []endpointCase{
		{"health", "GET", "/health", func(c *Client) { _, _, _ = c.Identity.Health(context.Background()) }},
		{"whoami", "GET", "/whoami", func(c *Client) { _, _, _ = c.Identity.WhoAmI(context.Background()) }},
		{"overview", "GET", "/overview", func(c *Client) { _, _, _ = c.Identity.Overview(context.Background(), "btc") }},
		{"markets.list", "GET", "/btc/markets", func(c *Client) { _, _, _, _ = c.Markets.List(context.Background(), "btc", nil) }},
		{"markets.get", "GET", "/btc/markets/123", func(c *Client) { _, _, _ = c.Markets.Get(context.Background(), "btc", "123") }},
		{"markets.by-slug", "GET", "/btc/markets/by-slug/slug", func(c *Client) { _, _, _ = c.Markets.GetBySlug(context.Background(), "btc", "slug") }},
		{"polymarket.venue", "GET", "/polymarket/markets", func(c *Client) { _, _, _, _ = c.Markets.PolymarketVenue(context.Background(), nil) }},
		{"snapshots.list", "GET", "/btc/markets/123/snapshots", func(c *Client) { _, _, _, _ = c.Snapshots.List(context.Background(), "btc", "123", nil) }},
		{"snapshots.at", "GET", "/btc/markets/123/snapshots/at/1780774485780", func(c *Client) {
			_, _, _ = c.Snapshots.AtTimestamp(context.Background(), "btc", "123", NewTimestampFromMillis(1780774485780))
		}},
		{"polymarket.asset.snapshots", "GET", "/polymarket/asset1/snapshots", func(c *Client) {
			_, _, _, _ = c.Snapshots.PolymarketAssetSnapshots(context.Background(), "asset1", nil)
		}},
		{"ticks.polymarket", "GET", "/btc/markets/123/ticks", func(c *Client) { _, _, _, _ = c.Ticks.Polymarket(context.Background(), "btc", "123", nil) }},
		{"ticks.polymarket.asset", "GET", "/polymarket/asset1/ticks", func(c *Client) { _, _, _, _ = c.Ticks.PolymarketAssetTicks(context.Background(), "asset1", nil) }},
		{"ticks.kalshi", "GET", "/kalshi/KXBTC-25T/ticks", func(c *Client) { _, _, _, _ = c.Ticks.Kalshi(context.Background(), "KXBTC-25T", nil) }},
		{"binance.latest", "GET", "/btc/spot/latest", func(c *Client) { _, _, _ = c.Binance.Latest(context.Background(), "btc", "spot") }},
		{"binance.snapshots", "GET", "/btc/spot/snapshots", func(c *Client) { _, _, _, _ = c.Binance.Snapshots(context.Background(), "btc", "spot", nil) }},
		{"binance.trades", "GET", "/btc/spot/trades", func(c *Client) { _, _, _, _ = c.Binance.Trades(context.Background(), "btc", "spot", nil) }},
		{"binance.trades.latest", "GET", "/btc/spot/trades/latest", func(c *Client) { _, _, _ = c.Binance.LatestTrade(context.Background(), "btc", "spot") }},
		{"bars", "GET", "/bars", func(c *Client) { _, _, _, _ = c.Bars.Get(context.Background(), &BarsOptions{Symbols: "BTCUSDT"}) }},
		{"kalshi.markets", "GET", "/kalshi/markets", func(c *Client) { _, _, _, _ = c.Kalshi.Markets(context.Background(), nil) }},
		{"kalshi.latest", "GET", "/kalshi/KXBTC-25T/orderbook/latest", func(c *Client) { _, _, _ = c.Kalshi.LatestBook(context.Background(), "KXBTC-25T") }},
		{"kalshi.snapshots", "GET", "/kalshi/KXBTC-25T/snapshots", func(c *Client) { _, _, _, _ = c.Kalshi.Snapshots(context.Background(), "KXBTC-25T", nil) }},
		{"kalshi.candles", "GET", "/kalshi/KXBTC-25T/candles", func(c *Client) { _, _, _, _ = c.Kalshi.Candles(context.Background(), "KXBTC-25T", nil) }},
		{"limitless.markets", "GET", "/limitless/markets", func(c *Client) { _, _, _, _ = c.Limitless.Markets(context.Background(), nil) }},
		{"limitless.latest", "GET", "/limitless/slug/orderbook/latest", func(c *Client) { _, _, _ = c.Limitless.LatestBook(context.Background(), "slug") }},
		{"limitless.snapshots", "GET", "/limitless/slug/snapshots", func(c *Client) { _, _, _, _ = c.Limitless.Snapshots(context.Background(), "slug", nil) }},
		{"binance-pred.markets", "GET", "/binance-prediction/markets", func(c *Client) { _, _, _, _ = c.BinancePrediction.Markets(context.Background(), nil) }},
		{"binance-pred.latest", "GET", "/binance-prediction/m1/orderbook/latest", func(c *Client) { _, _, _ = c.BinancePrediction.LatestBook(context.Background(), "m1") }},
		{"binance-pred.snapshots", "GET", "/binance-prediction/m1/snapshots", func(c *Client) { _, _, _, _ = c.BinancePrediction.Snapshots(context.Background(), "m1", nil) }},
		{"binance-pred.timeseries", "GET", "/binance-prediction/m1/timeseries", func(c *Client) {
			_, _, _, _ = c.BinancePrediction.TimeSeries(context.Background(), "m1", MetricChance, nil)
		}},
		{"binance-pred.trades", "GET", "/binance-prediction/m1/trades", func(c *Client) { _, _, _, _ = c.BinancePrediction.Trades(context.Background(), "m1", nil) }},
		{"screener.grid", "GET", "/screener", func(c *Client) { _, _, _ = c.Screener.Grid(context.Background()) }},
		{"screener.drilldown", "GET", "/screener/btc/5m", func(c *Client) { _, _, _ = c.Screener.Drilldown(context.Background(), "btc", "5m") }},
		{"batch", "POST", "/batch", func(c *Client) {
			req, _ := NewBatchRequest(BatchSubrequest{ID: "a", Method: "GET", Path: "/v3/btc/markets"})
			_, _, _ = c.Batch.Run(context.Background(), req)
		}},
		{"sports.overview", "GET", "/sports/overview", func(c *Client) { _, _, _ = c.Sports.Overview(context.Background()) }},
		{"sports.leagues", "GET", "/sports/leagues", func(c *Client) { _, _, _ = c.Sports.Leagues(context.Background()) }},
		{"sports.markets", "GET", "/sports/markets", func(c *Client) { _, _, _, _ = c.Sports.Markets(context.Background(), nil) }},
		{"sports.book.latest", "GET", "/sports/books/asset1/latest", func(c *Client) { _, _, _ = c.Sports.LatestBook(context.Background(), "asset1") }},
		{"sports.book.history", "GET", "/sports/books/asset1/history", func(c *Client) { _, _, _, _ = c.Sports.BookHistory(context.Background(), "asset1", nil) }},
		{"sports.games", "GET", "/sports/nba/games", func(c *Client) { _, _, _ = c.Sports.Games(context.Background(), "nba") }},
		{"sports.game", "GET", "/sports/games/g1", func(c *Client) { _, _, _ = c.Sports.Game(context.Background(), "g1") }},
		{"sports.game.markets", "GET", "/sports/games/g1/markets", func(c *Client) { _, _, _, _ = c.Sports.GameMarkets(context.Background(), "g1") }},
		{"sports.game.lines", "GET", "/sports/games/g1/lines", func(c *Client) { _, _, _, _ = c.Sports.GameLines(context.Background(), "g1") }},
		{"sports.game.box", "GET", "/sports/games/g1/box", func(c *Client) { _, _, _ = c.Sports.GameBox(context.Background(), "g1") }},
		{"sports.game.plays", "GET", "/sports/games/g1/plays", func(c *Client) { _, _, _, _ = c.Sports.GamePlays(context.Background(), "g1") }},
		{"sports.game.winprob", "GET", "/sports/games/g1/winprob", func(c *Client) { _, _, _, _ = c.Sports.GameWinProb(context.Background(), "g1") }},
		{"sports.market", "GET", "/sports/markets/c1", func(c *Client) { _, _, _ = c.Sports.Market(context.Background(), "c1") }},
		{"sports.players", "GET", "/sports/nba/players", func(c *Client) { _, _, _ = c.Sports.Players(context.Background(), "nba") }},
		{"sports.player", "GET", "/sports/players/a1", func(c *Client) { _, _, _ = c.Sports.Player(context.Background(), "a1") }},
		{"sports.player.injuries", "GET", "/sports/players/a1/injuries", func(c *Client) { _, _, _, _ = c.Sports.PlayerInjuries(context.Background(), "a1") }},
		{"sports.player.box", "GET", "/sports/players/a1/box", func(c *Client) { _, _, _ = c.Sports.PlayerBox(context.Background(), "a1") }},
		{"sports.player.markets", "GET", "/sports/players/a1/markets", func(c *Client) { _, _, _, _ = c.Sports.PlayerMarkets(context.Background(), "a1") }},
		{"sports.league.injuries", "GET", "/sports/nba/injuries", func(c *Client) { _, _, _, _ = c.Sports.LeagueInjuries(context.Background(), "nba") }},
		{"sports.odds.lines", "GET", "/sports/odds/lines", func(c *Client) { _, _, _, _ = c.Sports.OddsLines(context.Background()) }},
		{"sports.odds.books", "GET", "/sports/odds/books", func(c *Client) { _, _, _ = c.Sports.OddsBooks(context.Background()) }},
		{"sports.odds.history", "GET", "/sports/nba/odds-history", func(c *Client) { _, _, _, _ = c.Sports.OddsHistory(context.Background(), "nba", nil) }},
		{"sports.webhooks.create", "POST", "/sports/webhooks", func(c *Client) {
			_, _, _ = c.SportsWebhooks.Create(context.Background(), &SportsWebhookCreate{URL: "https://x", Leagues: []string{"nba"}})
		}},
		{"sports.webhooks.list", "GET", "/sports/webhooks", func(c *Client) { _, _, _ = c.SportsWebhooks.List(context.Background()) }},
		{"sports.webhooks.test", "POST", "/sports/webhooks/sub1/test", func(c *Client) { _, _ = c.SportsWebhooks.Test(context.Background(), "sub1") }},
		{"sports.webhooks.delete", "DELETE", "/sports/webhooks/sub1", func(c *Client) { _, _ = c.SportsWebhooks.Delete(context.Background(), "sub1") }},
	}
}

func TestEndpointCoverage(t *testing.T) {
	for _, tc := range endpointCases() {
		t.Run(tc.name, func(t *testing.T) {
			srv := testutil.New()
			defer srv.Close()
			srv.Handle(tc.method, tc.path, func(w http.ResponseWriter, r *http.Request) {
				// Return a generic 200 envelope with empty data.
				srv.DataResponse(w, []any{}, nil)
			})
			c := newTestClient(t, srv.Start())
			c.paperBaseURL = srv.Start()

			tc.call(c)

			reqs := srv.Requests()
			if len(reqs) == 0 {
				t.Fatalf("no request recorded for %s (expected %s %s)", tc.name, tc.method, tc.path)
			}
			last := reqs[len(reqs)-1]
			if last.Method != tc.method {
				t.Errorf("method = %q, want %q", last.Method, tc.method)
			}
			if last.Path != tc.path {
				t.Errorf("path = %q, want %q", last.Path, tc.path)
			}
		})
	}
}

// TestEndpointCount ensures the coverage table stays in sync with the
// documented route set (a guard against silent drift).
func TestEndpointCount(t *testing.T) {
	got := len(endpointCases())
	// 3 identity + 4 markets + 3 snapshots + 3 ticks + 4 binance + 1 bars +
	// 4 kalshi + 3 limitless + 5 binance-pred + 2 screener + 1 batch +
	// 22 sports + 4 sports-webhooks = 59
	const want = 59
	if got != want {
		t.Errorf("endpointCases count = %d, want %d (add new routes to the table)", got, want)
	}
}
