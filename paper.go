package depthfeed

import "context"

// PaperService provides access to the Paper Trading signal webhook.
// The token (shown once at create/rotate) is the auth. Fills always price
// from the live book at signal time — a payload can never set its own
// price. Use ClientID for idempotent client-side retry handling.
//
// Note: this endpoint uses https://api.depthfeed.com/paper/hook/{token},
// NOT the /v3 base path. Token-in-path authorization; no API key header.
//
// Docs: https://depthfeed.com/docs#paper-trading
type PaperService struct {
	client *Client
}

// Send posts a Paper Trading signal. The token is the per-strategy URL
// token (pt_...). Retry is caller-controlled: pass a stable ClientID in
// the signal so a retried POST is handled idempotently server-side. The
// SDK does NOT automatically retry this call.
// Path: POST /paper/hook/{token}
func (s *PaperService) Send(ctx context.Context, token string, signal *PaperSignal) (*PaperFill, *ResponseMetadata, error) {
	esc, err := pathEscape(token)
	if err != nil {
		return nil, nil, err
	}
	if signal == nil || signal.Action == "" || signal.Venue == "" || signal.Market == "" {
		return nil, nil, ErrEmptyPathSegment
	}
	var out PaperFill
	meta, err := s.client.postNoAuth(ctx, s.client.paperURL(), "/paper/hook/"+esc, signal, &out)
	return &out, meta, err
}
