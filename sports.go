package depthfeed

import (
	"context"
	"encoding/json"
	"net/url"
)

// SportsService provides access to realtime sports prediction-market data
// — live order-book depth on Polymarket and Kalshi, alongside sportsbook
// odds, live scores, and an injury feed. Both venues are included on
// every tier; plans change the shared history window, throughput, and
// live-book concurrency.
//
// Discovery funnel: /v3/sports/leagues → /v3/sports/markets?league=… →
// take a market's tokens[] (each is one outcome's asset_id) →
// /v3/sports/books/{asset_id}/latest.
//
// Docs: https://depthfeed.com/docs#ep-sports
type SportsService struct {
	client *Client
}

// SportsMarketListOptions filters /v3/sports/markets.
type SportsMarketListOptions struct {
	ListOptions
	// League filters by league (e.g. nba, mlb, soccer).
	League string
}

// ApplyToValues writes sports-market + common options.
func (o *SportsMarketListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "league", o.League)
	o.ListOptions.ApplyToValues(q)
}

// Overview is the live cross-sport board — every tracked league with its
// in-progress and upcoming games, current prices, and best bid/ask per
// venue, in a single request.
// Path: GET /v3/sports/overview
func (s *SportsService) Overview(ctx context.Context) (*SportsOverview, *ResponseMetadata, error) {
	var out SportsOverview
	meta, err := s.client.getSingle(ctx, "/sports/overview", nil, &out)
	return &out, meta, err
}

// Leagues returns the leagues DepthFeed tracks, each with its game and
// market counts — the top of the discovery funnel.
// Path: GET /v3/sports/leagues
func (s *SportsService) Leagues(ctx context.Context) ([]SportsLeague, *ResponseMetadata, error) {
	var out []SportsLeague
	meta, err := s.client.getSingle(ctx, "/sports/leagues", nil, &out)
	return out, meta, err
}

// Markets lists sports markets across Polymarket and Kalshi. Each market
// carries its condition_id, market_slug, outcomes, and tokens (the
// per-outcome asset_ids you hand to the book endpoint).
// Path: GET /v3/sports/markets
func (s *SportsService) Markets(ctx context.Context, opts *SportsMarketListOptions) ([]SportsMarket, *Pagination, *ResponseMetadata, error) {
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []SportsMarket
	var pg *Pagination
	meta, err := s.client.get(ctx, "/sports/markets", q, &out, &pg)
	return out, pg, meta, err
}

// LatestBook returns the newest recorded full order book for one outcome
// — the asset_id is a token from the market's tokens[]. Same [price, size]
// bid/ask ladder shape as the crypto book endpoints.
// Path: GET /v3/sports/books/{asset_id}/latest
func (s *SportsService) LatestBook(ctx context.Context, assetID string) (*SportsBook, *ResponseMetadata, error) {
	esc, err := pathEscape(assetID)
	if err != nil {
		return nil, nil, err
	}
	var out SportsBook
	meta, err := s.client.getSingle(ctx, "/sports/books/"+esc+"/latest", nil, &out)
	return &out, meta, err
}

// SportsBookHistoryOptions filters /v3/sports/books/{asset_id}/history.
// The shorter from/to aliases are also accepted in addition to
// start_time/end_time.
type SportsBookHistoryOptions struct {
	ListOptions
	// From is the shorter alias for start_time.
	From string
	// To is the shorter alias for end_time.
	To string
}

// ApplyToValues writes sports-book-history + common options.
func (o *SportsBookHistoryOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "from", o.From)
	applyStr(q, "to", o.To)
	o.ListOptions.ApplyToValues(q)
}

// BookHistory returns historical full-ladder snapshots for the outcome.
// Page with limit/cursor; bound time with start_time/end_time (or the
// shorter from/to aliases). Add interval to downsample.
// Path: GET /v3/sports/books/{asset_id}/history
func (s *SportsService) BookHistory(ctx context.Context, assetID string, opts *SportsBookHistoryOptions) ([]SportsBook, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(assetID)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []SportsBook
	var pg *Pagination
	meta, err := s.client.get(ctx, "/sports/books/"+esc+"/history", q, &out, &pg)
	return out, pg, meta, err
}

// Games returns live games and scores for a league (e.g. nba, nfl,
// soccer/eng.1), each with its linked Polymarket & Kalshi markets.
// Path: GET /v3/sports/{league}/games
func (s *SportsService) Games(ctx context.Context, league string) ([]SportsGame, *ResponseMetadata, error) {
	esc, err := pathEscape(league)
	if err != nil {
		return nil, nil, err
	}
	var out []SportsGame
	meta, err := s.client.getSingle(ctx, "/sports/"+esc+"/games", nil, &out)
	return out, meta, err
}

// Game returns the joined game document: schedule and final score, linked
// prediction markets and settlements, sportsbook lines, and injury context.
// Path: GET /v3/sports/games/{game_slug}
func (s *SportsService) Game(ctx context.Context, gameSlug string) (*SportsGame, *ResponseMetadata, error) {
	esc, err := pathEscape(gameSlug)
	if err != nil {
		return nil, nil, err
	}
	var out SportsGame
	meta, err := s.client.getSingle(ctx, "/sports/games/"+esc, nil, &out)
	return &out, meta, err
}

// rawCollection fetches a list-style game/player sub-resource into a slice
// of json.RawMessage, preserving unknown fields.
func (s *SportsService) rawCollection(ctx context.Context, path string) ([]json.RawMessage, *Pagination, *ResponseMetadata, error) {
	var out []json.RawMessage
	var pg *Pagination
	meta, err := s.client.get(ctx, path, nil, &out, &pg)
	return out, pg, meta, err
}

// rawSingle fetches a single-object sub-resource into json.RawMessage.
func (s *SportsService) rawSingle(ctx context.Context, path string) (json.RawMessage, *ResponseMetadata, error) {
	var out json.RawMessage
	meta, err := s.client.getSingle(ctx, path, nil, &out)
	return out, meta, err
}

// GameMarkets returns game markets.
// Path: GET /v3/sports/games/{game_slug}/markets
func (s *SportsService) GameMarkets(ctx context.Context, gameSlug string) ([]json.RawMessage, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(gameSlug)
	if err != nil {
		return nil, nil, nil, err
	}
	return s.rawCollection(ctx, "/sports/games/"+esc+"/markets")
}

// GameLines returns game line movement.
// Path: GET /v3/sports/games/{game_slug}/lines
func (s *SportsService) GameLines(ctx context.Context, gameSlug string) ([]json.RawMessage, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(gameSlug)
	if err != nil {
		return nil, nil, nil, err
	}
	return s.rawCollection(ctx, "/sports/games/"+esc+"/lines")
}

// GameBox returns the game box score.
// Path: GET /v3/sports/games/{game_slug}/box
func (s *SportsService) GameBox(ctx context.Context, gameSlug string) (json.RawMessage, *ResponseMetadata, error) {
	esc, err := pathEscape(gameSlug)
	if err != nil {
		return nil, nil, err
	}
	return s.rawSingle(ctx, "/sports/games/"+esc+"/box")
}

// GamePlays returns play-by-play.
// Path: GET /v3/sports/games/{game_slug}/plays
func (s *SportsService) GamePlays(ctx context.Context, gameSlug string) ([]json.RawMessage, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(gameSlug)
	if err != nil {
		return nil, nil, nil, err
	}
	return s.rawCollection(ctx, "/sports/games/"+esc+"/plays")
}

// GameWinProb returns per-play win probability.
// Path: GET /v3/sports/games/{game_slug}/winprob
func (s *SportsService) GameWinProb(ctx context.Context, gameSlug string) ([]json.RawMessage, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(gameSlug)
	if err != nil {
		return nil, nil, nil, err
	}
	return s.rawCollection(ctx, "/sports/games/"+esc+"/winprob")
}

// Market returns one normalized market with venue identifiers, outcomes,
// token IDs, game join, and settlement state.
// Path: GET /v3/sports/markets/{condition_id}
func (s *SportsService) Market(ctx context.Context, conditionID string) (*SportsMarketSingle, *ResponseMetadata, error) {
	esc, err := pathEscape(conditionID)
	if err != nil {
		return nil, nil, err
	}
	var out SportsMarketSingle
	meta, err := s.client.getSingle(ctx, "/sports/markets/"+esc, nil, &out)
	return &out, meta, err
}

// Players searches the league athlete directory and retrieves stable
// athlete IDs used by player-prop joins.
// Path: GET /v3/sports/{league}/players
func (s *SportsService) Players(ctx context.Context, league string) ([]SportsPlayer, *ResponseMetadata, error) {
	esc, err := pathEscape(league)
	if err != nil {
		return nil, nil, err
	}
	var out []SportsPlayer
	meta, err := s.client.getSingle(ctx, "/sports/"+esc+"/players", nil, &out)
	return out, meta, err
}

// Player returns a player profile.
// Path: GET /v3/sports/players/{athlete_id}
func (s *SportsService) Player(ctx context.Context, athleteID string) (*SportsPlayerProfile, *ResponseMetadata, error) {
	esc, err := pathEscape(athleteID)
	if err != nil {
		return nil, nil, err
	}
	var out SportsPlayerProfile
	meta, err := s.client.getSingle(ctx, "/sports/players/"+esc, nil, &out)
	return &out, meta, err
}

// PlayerInjuries returns the player injury change-log.
// Path: GET /v3/sports/players/{athlete_id}/injuries
func (s *SportsService) PlayerInjuries(ctx context.Context, athleteID string) ([]json.RawMessage, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(athleteID)
	if err != nil {
		return nil, nil, nil, err
	}
	return s.rawCollection(ctx, "/sports/players/"+esc+"/injuries")
}

// PlayerBox returns the player box-score game log.
// Path: GET /v3/sports/players/{athlete_id}/box
func (s *SportsService) PlayerBox(ctx context.Context, athleteID string) (json.RawMessage, *ResponseMetadata, error) {
	esc, err := pathEscape(athleteID)
	if err != nil {
		return nil, nil, err
	}
	return s.rawSingle(ctx, "/sports/players/"+esc+"/box")
}

// PlayerMarkets returns entity-resolved prediction-market props.
// Path: GET /v3/sports/players/{athlete_id}/markets
func (s *SportsService) PlayerMarkets(ctx context.Context, athleteID string) ([]json.RawMessage, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(athleteID)
	if err != nil {
		return nil, nil, nil, err
	}
	return s.rawCollection(ctx, "/sports/players/"+esc+"/markets")
}

// LeagueInjuries returns the live injury feed per league — the pre-game
// status changes that move the market.
// Path: GET /v3/sports/{league}/injuries
func (s *SportsService) LeagueInjuries(ctx context.Context, league string) ([]json.RawMessage, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(league)
	if err != nil {
		return nil, nil, nil, err
	}
	return s.rawCollection(ctx, "/sports/"+esc+"/injuries")
}

// OddsLines returns live moneyline, spread, total, and prop lines across
// books, normalized for comparison with the market price.
// Path: GET /v3/sports/odds/lines
func (s *SportsService) OddsLines(ctx context.Context) ([]SportsOddsLine, *Pagination, *ResponseMetadata, error) {
	var out []SportsOddsLine
	var pg *Pagination
	meta, err := s.client.get(ctx, "/sports/odds/lines", nil, &out, &pg)
	return out, pg, meta, err
}

// OddsBooks returns the active sportsbook catalog and source freshness.
// Path: GET /v3/sports/odds/books
func (s *SportsService) OddsBooks(ctx context.Context) ([]SportsBookCatalogEntry, *ResponseMetadata, error) {
	var out []SportsBookCatalogEntry
	meta, err := s.client.getSingle(ctx, "/sports/odds/books", nil, &out)
	return out, meta, err
}

// OddsHistoryOptions filters /v3/sports/{league}/odds-history. Bound NFL
// requests with season, team, from, or to. Soccer requests require a
// football-data season code (e.g. 2324) and/or team.
type OddsHistoryOptions struct {
	ListOptions
	Season string
	Team   string
	// From is a bound alias (start).
	From string
	// To is a bound alias (end).
	To string
}

// ApplyToValues writes odds-history + common options.
func (o *OddsHistoryOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	applyStr(q, "season", o.Season)
	applyStr(q, "team", o.Team)
	applyStr(q, "from", o.From)
	applyStr(q, "to", o.To)
	o.ListOptions.ApplyToValues(q)
}

// OddsHistory returns historical sportsbook lines, including NFL history
// to 1999 and European soccer to the 2000/01 season.
// Path: GET /v3/sports/{league}/odds-history
func (s *SportsService) OddsHistory(ctx context.Context, league string, opts *OddsHistoryOptions) ([]SportsOddsHistory, *Pagination, *ResponseMetadata, error) {
	esc, err := pathEscape(league)
	if err != nil {
		return nil, nil, nil, err
	}
	q := url.Values{}
	opts.ApplyToValues(q)
	var out []SportsOddsHistory
	var pg *Pagination
	meta, err := s.client.get(ctx, "/sports/"+esc+"/odds-history", q, &out, &pg)
	return out, pg, meta, err
}
