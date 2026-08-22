package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	depthfeed "github.com/tigusigalpa/depthfeed-go"
	"github.com/tigusigalpa/depthfeed-go/stream"
)

// Example: subscribe to the live BTC Polymarket book stream and print
// frames as they arrive. Press Ctrl+C to stop.
func main() {
	client, err := depthfeed.NewClient(depthfeed.Config{
		APIKey: os.Getenv("DEPTHFEED_API_KEY"),
	})
	if err != nil {
		log.Fatalf("NewClient: %v", err)
	}

	// Discover the first BTC market to subscribe to.
	markets, _, _, err := client.Markets.List(context.Background(), depthfeed.CoinBTC, &depthfeed.MarketListOptions{
		ListOptions: depthfeed.ListOptions{Limit: 1},
	})
	if err != nil {
		log.Fatalf("Markets.List: %v", err)
	}
	if len(markets) == 0 {
		log.Fatal("no BTC markets found")
	}
	marketID := markets[0].MarketID
	channel := stream.BookPolymarket("btc", marketID)
	fmt.Printf("Subscribing to %s\n", channel)

	// Build a stream client from the REST client.
	sc := client.NewStreamClient(
		stream.WithReconnect(true),
		stream.WithBackoff(250*time.Millisecond, 5*time.Second),
	)
	sc.OnEvent(func(e stream.Event) {
		switch e.Kind {
		case stream.EventConnected:
			fmt.Println("connected")
		case stream.EventDisconnected:
			fmt.Printf("disconnected: %v\n", e.Err)
		case stream.EventReconnecting:
			fmt.Printf("reconnecting (attempt %d)\n", e.Attempt)
		case stream.EventReconnected:
			fmt.Printf("reconnected (attempt %d)\n", e.Attempt)
		}
	})

	ctx := context.Background()
	if err := sc.Connect(ctx); err != nil {
		log.Fatalf("Connect: %v", err)
	}
	defer sc.Close()

	if err := sc.Subscribe([]string{channel}); err != nil {
		log.Fatalf("Subscribe: %v", err)
	}

	for {
		frame, err := sc.Receive(ctx)
		if err != nil {
			log.Fatalf("Receive: %v", err)
		}
		if frame.IsError() {
			fmt.Printf("error frame: code=%s msg=%s\n", frame.Code, frame.Message)
			continue
		}
		if frame.Op == "book" {
			var pretty map[string]any
			_ = json.Unmarshal(frame.Data, &pretty)
			out, _ := json.MarshalIndent(pretty, "", "  ")
			fmt.Printf("book @ %d:\n%s\n", frame.TS, out)
		}
	}
}
