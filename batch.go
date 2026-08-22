package depthfeed

import (
	"context"
	"strings"
)

// BatchService provides access to POST /v3/batch. Run up to 10 GET /v3/
// requests in one round-trip. Each sub-request is dispatched independently
// and gets its own status.
//
// A contract's depth and the underlying's price bars in one round-trip is
// the pairing this exists for — /v3/bars is a sub-request like any other.
// Over 10 sub-requests → 400 BATCH_TOO_LARGE; a non-GET or non-/v3/ path
// gets a per-sub 400 BATCH_INVALID_PATH.
//
// Docs: https://depthfeed.com/docs#ep-batch
type BatchService struct {
	client *Client
}

// NewBatchRequest builds a BatchRequest from subrequest ids and paths.
// Each path must be a GET /v3/ path (with optional query string). At most
// 10 subrequests are allowed.
func NewBatchRequest(subrequests ...BatchSubrequest) (*BatchRequest, error) {
	if len(subrequests) > 10 {
		return nil, ErrBatchTooLarge
	}
	for _, sr := range subrequests {
		if !strings.EqualFold(sr.Method, "GET") {
			return nil, ErrBatchInvalidPath
		}
		if !strings.HasPrefix(sr.Path, "/v3/") {
			return nil, ErrBatchInvalidPath
		}
	}
	return &BatchRequest{Requests: subrequests}, nil
}

// Run executes a batch. The outer response is 200; each subresponse
// carries its own status and body. Batch contains only GET subrequests
// and is eligible for retry.
// Path: POST /v3/batch
func (s *BatchService) Run(ctx context.Context, req *BatchRequest) (*BatchResponse, *ResponseMetadata, error) {
	if req == nil {
		return nil, nil, ErrBatchTooLarge
	}
	if len(req.Requests) > 10 {
		return nil, nil, ErrBatchTooLarge
	}
	for _, sr := range req.Requests {
		if !strings.EqualFold(sr.Method, "GET") || !strings.HasPrefix(sr.Path, "/v3/") {
			return nil, nil, ErrBatchInvalidPath
		}
	}
	var out BatchResponse
	// Batch contains only GET subrequests, so it is eligible for retry.
	meta, err := s.client.postRetryable(ctx, "/batch", req, &out)
	return &out, meta, err
}
