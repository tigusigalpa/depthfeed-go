package depthfeed

import (
	"encoding/json"
	"strings"
)

// rawSnapshot is an alias type used to decode Snapshot's known fields
// without triggering the custom UnmarshalJSON.
type rawSnapshot struct {
	ID            string     `json:"id"`
	Time          string     `json:"time"`
	MarketID      string     `json:"market_id"`
	PriceUp       float64    `json:"price_up"`
	PriceDown     float64    `json:"price_down"`
	OrderbookUp   *OrderBook `json:"orderbook_up,omitempty"`
	OrderbookDown *OrderBook `json:"orderbook_down,omitempty"`
}

// UnmarshalJSON stores the full raw object (so callers can access the
// dynamic per-coin price key, e.g. btc_price) and decodes the known
// fields. Unknown fields are preserved, not dropped.
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	s.Raw = append(s.Raw[:0], data...)
	var r rawSnapshot
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	s.ID = r.ID
	s.Time = r.Time
	s.MarketID = r.MarketID
	s.PriceUp = r.PriceUp
	s.PriceDown = r.PriceDown
	s.OrderbookUp = r.OrderbookUp
	s.OrderbookDown = r.OrderbookDown
	// Extract the dynamic {coin}_price field (e.g. btc_price), if present.
	var generic map[string]any
	if json.Unmarshal(data, &generic) == nil {
		for k, v := range generic {
			if strings.HasSuffix(k, "_price") && k != "price_up" && k != "price_down" {
				if f, ok := v.(float64); ok {
					s.CoinPrice = f
					break
				}
			}
		}
	}
	return nil
}

// rawMarket aliases Market for decoding known fields.
type rawMarket struct {
	MarketID       string   `json:"market_id"`
	EventID        string   `json:"event_id"`
	Slug           string   `json:"slug"`
	MarketType     string   `json:"market_type"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	ConditionID    string   `json:"condition_id"`
	ClobTokenUp    string   `json:"clob_token_up"`
	ClobTokenDown  string   `json:"clob_token_down"`
	Winner         *string  `json:"winner"`
	FinalVolume    *float64 `json:"final_volume"`
	FinalLiquidity *float64 `json:"final_liquidity"`
	ResolvedAt     *string  `json:"resolved_at"`
}

// UnmarshalJSON stores the full raw object (preserving dynamic
// {coin}_price_start / {coin}_price_end fields) and decodes known fields.
func (m *Market) UnmarshalJSON(data []byte) error {
	m.Raw = append(m.Raw[:0], data...)
	var r rawMarket
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	m.MarketID = r.MarketID
	m.EventID = r.EventID
	m.Slug = r.Slug
	m.MarketType = r.MarketType
	m.StartTime = r.StartTime
	m.EndTime = r.EndTime
	m.ConditionID = r.ConditionID
	m.ClobTokenUp = r.ClobTokenUp
	m.ClobTokenDown = r.ClobTokenDown
	m.Winner = r.Winner
	m.FinalVolume = r.FinalVolume
	m.FinalLiquidity = r.FinalLiquidity
	m.ResolvedAt = r.ResolvedAt
	return nil
}

// preserveRaw unmarshals data into v and also stores a copy in *raw.
func preserveRaw(data []byte, v any, raw *json.RawMessage) error {
	*raw = append((*raw)[:0], data...)
	return json.Unmarshal(data, v)
}

// UnmarshalJSON preserves the raw object for PolymarketVenueMarket.
func (m *PolymarketVenueMarket) UnmarshalJSON(data []byte) error {
	type alias struct {
		AssetID     string   `json:"asset_id"`
		ConditionID string   `json:"condition_id"`
		Slug        string   `json:"slug"`
		BaseAsset   string   `json:"base_asset"`
		MarketType  string   `json:"market_type"`
		Outcomes    []string `json:"outcomes"`
		Event       string   `json:"event"`
	}
	var a alias
	if err := preserveRaw(data, &a, &m.Raw); err != nil {
		return err
	}
	m.AssetID = a.AssetID
	m.ConditionID = a.ConditionID
	m.Slug = a.Slug
	m.BaseAsset = a.BaseAsset
	m.MarketType = a.MarketType
	m.Outcomes = a.Outcomes
	m.Event = a.Event
	return nil
}

// UnmarshalJSON preserves the raw object for BinancePredictionMarket.
func (m *BinancePredictionMarket) UnmarshalJSON(data []byte) error {
	type alias struct {
		MarketID       string   `json:"market_id"`
		TopicID        string   `json:"topic_id"`
		ConditionID    string   `json:"condition_id"`
		BaseAsset      string   `json:"base_asset"`
		MarketType     string   `json:"market_type"`
		Status         string   `json:"status"`
		WinningOutcome *string  `json:"winning_outcome"`
		Volume         *float64 `json:"volume"`
		Liquidity      *float64 `json:"liquidity"`
		Participants   *int     `json:"participants"`
		Chain          string   `json:"chain"`
	}
	var a alias
	if err := preserveRaw(data, &a, &m.Raw); err != nil {
		return err
	}
	m.MarketID = a.MarketID
	m.TopicID = a.TopicID
	m.ConditionID = a.ConditionID
	m.BaseAsset = a.BaseAsset
	m.MarketType = a.MarketType
	m.Status = a.Status
	m.WinningOutcome = a.WinningOutcome
	m.Volume = a.Volume
	m.Liquidity = a.Liquidity
	m.Participants = a.Participants
	m.Chain = a.Chain
	return nil
}

// UnmarshalJSON preserves the raw object for ScreenerDrilldown.
func (s *ScreenerDrilldown) UnmarshalJSON(data []byte) error {
	type alias struct {
		Asset  string        `json:"asset"`
		Window string        `json:"window"`
		Books  ScreenerBooks `json:"books"`
	}
	var a alias
	if err := preserveRaw(data, &a, &s.Raw); err != nil {
		return err
	}
	s.Asset = a.Asset
	s.Window = a.Window
	s.Books = a.Books
	return nil
}

// UnmarshalJSON preserves the raw object for Overview.
func (o *Overview) UnmarshalJSON(data []byte) error {
	type alias struct {
		Assets []json.RawMessage `json:"assets"`
	}
	var a alias
	if err := preserveRaw(data, &a, &o.Raw); err != nil {
		return err
	}
	o.Assets = a.Assets
	return nil
}

// UnmarshalJSON preserves the raw object for SportsOverview.
func (o *SportsOverview) UnmarshalJSON(data []byte) error {
	type alias struct {
		Leagues []json.RawMessage `json:"leagues"`
	}
	var a alias
	if err := preserveRaw(data, &a, &o.Raw); err != nil {
		return err
	}
	o.Leagues = a.Leagues
	return nil
}

// UnmarshalJSON preserves the raw object for SportsGame.
func (g *SportsGame) UnmarshalJSON(data []byte) error {
	type alias struct {
		GameSlug string `json:"game_slug"`
		League   string `json:"league"`
	}
	var a alias
	if err := preserveRaw(data, &a, &g.Raw); err != nil {
		return err
	}
	g.GameSlug = a.GameSlug
	g.League = a.League
	return nil
}

// UnmarshalJSON preserves the raw object for SportsMarketSingle.
func (m *SportsMarketSingle) UnmarshalJSON(data []byte) error {
	type alias struct {
		ConditionID string   `json:"condition_id"`
		Outcomes    []string `json:"outcomes"`
		Tokens      []string `json:"tokens"`
	}
	var a alias
	if err := preserveRaw(data, &a, &m.Raw); err != nil {
		return err
	}
	m.ConditionID = a.ConditionID
	m.Outcomes = a.Outcomes
	m.Tokens = a.Tokens
	return nil
}

// UnmarshalJSON preserves the raw object for SportsPlayerProfile.
func (p *SportsPlayerProfile) UnmarshalJSON(data []byte) error {
	type alias struct {
		AthleteID string `json:"athlete_id"`
	}
	var a alias
	if err := preserveRaw(data, &a, &p.Raw); err != nil {
		return err
	}
	p.AthleteID = a.AthleteID
	return nil
}

// UnmarshalJSON preserves the raw object for SportsOddsLine.
func (l *SportsOddsLine) UnmarshalJSON(data []byte) error {
	return preserveRaw(data, &struct{}{}, &l.Raw)
}

// UnmarshalJSON preserves the raw object for SportsOddsHistory.
func (h *SportsOddsHistory) UnmarshalJSON(data []byte) error {
	return preserveRaw(data, &struct{}{}, &h.Raw)
}

// UnmarshalJSON preserves the raw object for BinancePredictionTimeSeries.
func (t *BinancePredictionTimeSeries) UnmarshalJSON(data []byte) error {
	type alias struct {
		MarketID string            `json:"market_id"`
		Metric   string            `json:"metric"`
		Points   []TimeSeriesPoint `json:"points"`
	}
	var a alias
	if err := preserveRaw(data, &a, &t.Raw); err != nil {
		return err
	}
	t.MarketID = a.MarketID
	t.Metric = a.Metric
	t.Points = a.Points
	return nil
}
