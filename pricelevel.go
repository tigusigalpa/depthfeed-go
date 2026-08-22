package depthfeed

import (
	"encoding/json"
	"fmt"
)

// PriceLevel is a single order-book ladder level. The API encodes a level
// as a two-item array [price, size]; this type maps safely to and from
// that tuple while exposing named fields. Prices are probability prices
// (0–1 dollars) or asset prices, and sizes are quantities — both are kept
// as float64 exactly as the API returns them. Do not locale-format these
// values at serialization boundaries.
type PriceLevel struct {
	Price float64 `json:"-"`
	Size  float64 `json:"-"`
}

// MarshalJSON encodes the level as [price, size].
func (l PriceLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]float64{l.Price, l.Size})
}

// UnmarshalJSON decodes a [price, size] array.
func (l *PriceLevel) UnmarshalJSON(data []byte) error {
	var arr [2]float64
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("depthfeed: price level must be [price, size]: %w", err)
	}
	l.Price = arr[0]
	l.Size = arr[1]
	return nil
}

// OrderBook is a two-sided ladder of [price, size] levels.
type OrderBook struct {
	Bids []PriceLevel `json:"bids"`
	Asks []PriceLevel `json:"asks"`
}

// YesNoBook is a Kalshi yes/no book, each side a slice of [price, size].
type YesNoBook struct {
	Yes []PriceLevel `json:"yes"`
	No  []PriceLevel `json:"no"`
}
