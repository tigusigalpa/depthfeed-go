// Package stream implements the DepthFeed v3 live WebSocket stream
// (wss://api.depthfeed.com/v3/stream). It pushes live order books and
// prices as they change; frame payloads are the same JSON objects the REST
// snapshot endpoints return, so the frame Data field is exposed as
// encoding/json.RawMessage and can be decoded into the root depthfeed
// package types (e.g. depthfeed.Snapshot).
//
// The package is self-contained and does not import the root depthfeed
// package, so it can be used independently or via Client.NewStreamClient.
//
// Docs: https://depthfeed.com/docs#ep-stream
package stream

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Op is a WebSocket operation.
type Op string

// Documented operations.
const (
	OpSubscribe   Op = "subscribe"
	OpUnsubscribe Op = "unsubscribe"
	OpPing        Op = "ping"
	OpPong        Op = "pong"
	OpError       Op = "error"
	OpSubscribed  Op = "subscribed"
)

// CloseCode is a WebSocket close code.
type CloseCode int

// Documented close codes.
const (
	CloseAuthFailed      CloseCode = 4001
	ClosePlanLimit       CloseCode = 4003
	CloseOpFlooding      CloseCode = 4029
	CloseAuthUnavailable CloseCode = 1013
)

// Frame is a decoded incoming WebSocket frame. Use the Channel + Op fields
// to dispatch; Data carries the raw payload (same shape as REST).
type Frame struct {
	// Op is the operation type ("subscribed", "error", "price", "book", …).
	Op Op `json:"op"`
	// Channel is the channel name for data frames (e.g. "price:btc").
	Channel string `json:"channel,omitempty"`
	// Data is the raw payload. For book/price frames it is the same JSON
	// object the REST snapshot endpoints return.
	Data json.RawMessage `json:"data,omitempty"`
	// Channels is the echoed channel list for subscribe/unsubscribe/subscribed acks.
	Channels []string `json:"channels,omitempty"`
	// Active is the active subscription count after an op.
	Active int `json:"active,omitempty"`
	// Code is the error code on an error frame.
	Code string `json:"code,omitempty"`
	// Message is the error message on an error frame.
	Message string `json:"message,omitempty"`
	// TS is the server timestamp (epoch-ms) on data frames.
	TS int64 `json:"ts,omitempty"`
}

// IsError reports whether the frame is an op-level error.
func (f *Frame) IsError() bool { return f.Op == OpError }

// IsSubscribed reports whether the frame is a subscribe ack.
func (f *Frame) IsSubscribed() bool { return f.Op == OpSubscribed }

// --- Channel builders ---

// BookPolymarket builds book:polymarket:<coin>:<market_id>.
func BookPolymarket(coin, marketID string) string {
	return fmt.Sprintf("book:polymarket:%s:%s", coin, marketID)
}

// BookKalshi builds book:kalshi:<coin>:<ticker>.
func BookKalshi(coin, ticker string) string {
	return fmt.Sprintf("book:kalshi:%s:%s", coin, ticker)
}

// BookLimitless builds book:limitless:<coin>:<slug>.
func BookLimitless(coin, slug string) string {
	return fmt.Sprintf("book:limitless:%s:%s", coin, slug)
}

// BookBinance builds book:binance:<coin>:<spot|futures>.
func BookBinance(coin, venue string) string {
	return fmt.Sprintf("book:binance:%s:%s", coin, venue)
}

// BookSportsPolymarket builds book:sports:polymarket:<token_id>.
func BookSportsPolymarket(tokenID string) string {
	return fmt.Sprintf("book:sports:polymarket:%s", tokenID)
}

// BookSportsKalshi builds book:sports:kalshi:<ticker>.
func BookSportsKalshi(ticker string) string {
	return fmt.Sprintf("book:sports:kalshi:%s", ticker)
}

// Price builds price:<coin>.
func Price(coin string) string {
	return fmt.Sprintf("price:%s", coin)
}

// ValidateChannel reports whether a channel name is well-formed
// (non-empty, known prefix). It does NOT check plan entitlements.
func ValidateChannel(ch string) bool {
	if ch == "" {
		return false
	}
	switch {
	case strings.HasPrefix(ch, "book:polymarket:"),
		strings.HasPrefix(ch, "book:kalshi:"),
		strings.HasPrefix(ch, "book:limitless:"),
		strings.HasPrefix(ch, "book:binance:"),
		strings.HasPrefix(ch, "book:sports:polymarket:"),
		strings.HasPrefix(ch, "book:sports:kalshi:"),
		strings.HasPrefix(ch, "price:"):
		return true
	}
	return false
}

// SubscribeMessage builds a subscribe op JSON message.
func SubscribeMessage(channels []string) ([]byte, error) {
	return json.Marshal(Frame{Op: OpSubscribe, Channels: channels})
}

// UnsubscribeMessage builds an unsubscribe op JSON message.
func UnsubscribeMessage(channels []string) ([]byte, error) {
	return json.Marshal(Frame{Op: OpUnsubscribe, Channels: channels})
}

// PingMessage builds a ping op JSON message.
func PingMessage() ([]byte, error) {
	return json.Marshal(Frame{Op: OpPing})
}
