package depthfeed

import "context"

// SportsWebhooksService provides access to sports injury webhook
// subscription management. Paid plan. The signing secret is shown once —
// never log or persist it unless the application explicitly asks.
//
// Deliveries include X-DepthFeed-Event and X-DepthFeed-Signature; verify
// the HMAC-SHA256 signature over the exact raw request body before
// parsing JSON. See the webhooks subpackage for verification helpers.
//
// Docs: https://depthfeed.com/docs#ep-sports
type SportsWebhooksService struct {
	client *Client
}

// Create registers an HTTPS receiver with league and injury-status
// filters. The signing secret is shown once in the response.
// Path: POST /v3/sports/webhooks
func (s *SportsWebhooksService) Create(ctx context.Context, req *SportsWebhookCreate) (*SportsWebhook, *ResponseMetadata, error) {
	if req == nil || req.URL == "" {
		return nil, nil, ErrEmptyPathSegment
	}
	var out SportsWebhook
	meta, err := s.client.post(ctx, "/sports/webhooks", req, &out)
	return &out, meta, err
}

// List returns subscriptions and delivery health.
// Path: GET /v3/sports/webhooks
func (s *SportsWebhooksService) List(ctx context.Context) ([]SportsWebhook, *ResponseMetadata, error) {
	var out []SportsWebhook
	meta, err := s.client.getSingle(ctx, "/sports/webhooks", nil, &out)
	return out, meta, err
}

// Test requests a signed test event for a subscription.
// Path: POST /v3/sports/webhooks/{sub_id}/test
func (s *SportsWebhooksService) Test(ctx context.Context, subID string) (*ResponseMetadata, error) {
	esc, err := pathEscape(subID)
	if err != nil {
		return nil, err
	}
	meta, err := s.client.post(ctx, "/sports/webhooks/"+esc+"/test", nil, nil)
	return meta, err
}

// Delete revokes a subscription.
// Path: DELETE /v3/sports/webhooks/{sub_id}
func (s *SportsWebhooksService) Delete(ctx context.Context, subID string) (*ResponseMetadata, error) {
	esc, err := pathEscape(subID)
	if err != nil {
		return nil, err
	}
	meta, err := s.client.delete(ctx, "/sports/webhooks/"+esc, nil)
	return meta, err
}
