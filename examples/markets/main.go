package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	depthfeed "github.com/tigusigalpa/depthfeed-go"
)

// Example: list BTC Polymarket markets and page through snapshots.
func main() {
	client, err := depthfeed.NewClient(depthfeed.Config{
		APIKey: os.Getenv("DEPTHFEED_API_KEY"),
		Retry: depthfeed.RetryConfig{
			Enabled:      true,
			MaxAttempts:  4,
			InitialDelay: 250 * time.Millisecond,
			MaxDelay:     5 * time.Second,
		},
	})
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	ctx := context.Background()

	// Confirm the key works.
	who, _, err := client.Identity.WhoAmI(ctx)
	if err != nil {
		log.Fatalf("WhoAmI: %v", err)
	}
	fmt.Printf("Plan: %s, RPS: %d, History: %d days\n", who.Plan, who.RPS, who.HistoryDays)

	// List BTC markets.
	markets, pg, _, err := client.Markets.List(ctx, depthfeed.CoinBTC, &depthfeed.MarketListOptions{
		ListOptions: depthfeed.ListOptions{Limit: 100},
	})
	if err != nil {
		log.Fatalf("Markets.List: %v", err)
	}
	fmt.Printf("Got %d BTC markets (has_more=%v)\n", len(markets), pg.HasMorePages())

	if len(markets) == 0 {
		return
	}

	// Fetch the first market's snapshots with a bounded window.
	start := depthfeed.NewTimestampFromTime(time.Now().Add(-1 * time.Hour))
	end := depthfeed.NewTimestampFromTime(time.Now())
	snapshots, _, _, err := client.Snapshots.List(ctx, depthfeed.CoinBTC, markets[0].MarketID, &depthfeed.SnapshotListOptions{
		ListOptions: depthfeed.ListOptions{
			Limit:     100,
			StartTime: start,
			EndTime:   end,
		},
		IncludeOrderbook: true,
	})
	if err != nil {
		log.Fatalf("Snapshots.List: %v", err)
	}
	fmt.Printf("Got %d snapshots for market %s\n", len(snapshots), markets[0].MarketID)
}
