package depthfeed

import (
	"context"
	"net/url"
)

// IdentityService provides access to health, whoami, and overview.
// /v3/health and /v3/overview are public (no key required); /v3/whoami
// requires a key.
//
// Docs: https://depthfeed.com/docs#ep-identity
type IdentityService struct {
	client *Client
}

// Health performs a public liveness probe. No API key is sent.
// Path: GET /v3/health
func (s *IdentityService) Health(ctx context.Context) (*Health, *ResponseMetadata, error) {
	var out Health
	meta, err := s.client.getNoAuth(ctx, "/health", nil, &out)
	return &out, meta, err
}

// WhoAmI returns the authenticated identity, plan, rate limits, history
// window, asset/sports access, and bars limits. The fastest way to
// confirm a key works.
// Path: GET /v3/whoami
func (s *IdentityService) WhoAmI(ctx context.Context) (*WhoAmI, *ResponseMetadata, error) {
	var out WhoAmI
	meta, err := s.client.getSingle(ctx, "/whoami", nil, &out)
	return &out, meta, err
}

// Overview returns the public per-asset, cross-venue time-series overview
// (what powers the dashboard overview chart). No key required. Pass an
// empty asset to select all.
// Path: GET /v3/overview
func (s *IdentityService) Overview(ctx context.Context, asset string) (*Overview, *ResponseMetadata, error) {
	q := url.Values{}
	applyStr(q, "asset", asset)
	var out Overview
	meta, err := s.client.getNoAuth(ctx, "/overview", q, &out)
	return &out, meta, err
}
