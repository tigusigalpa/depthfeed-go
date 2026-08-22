package depthfeed

import (
	"net/url"
	"strconv"
)

// ListOptions holds the common query parameters shared by every collection
// route: limit, cursor, include_count, start_time, end_time, and interval.
// See https://depthfeed.com/docs#pagination.
type ListOptions struct {
	// Limit is the max results per page. Markets max 100, snapshots/trades
	// max 1000. Defaults: 50 / 100.
	Limit int
	// Cursor is the opaque keyset cursor from the prior page. Send it back
	// unchanged.
	Cursor string
	// IncludeCount adds total_count (costs a COUNT — opt-in).
	IncludeCount bool
	// StartTime bounds the window (ms-epoch or ISO-8601). Older than your
	// plan window → 403.
	StartTime Timestamp
	// EndTime bounds the window (ms-epoch or ISO-8601).
	EndTime Timestamp
	// Interval downsamples snapshots to one book per bucket (snapshots
	// only). Omit for full resolution.
	Interval Interval
}

// ApplyToValues writes the common options into q, omitting zero/empty
// values so they are not sent on the wire.
func (o *ListOptions) ApplyToValues(q url.Values) {
	if o == nil {
		return
	}
	if o.Limit > 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Cursor != "" {
		q.Set("cursor", o.Cursor)
	}
	if o.IncludeCount {
		q.Set("include_count", "true")
	}
	if o.StartTime.IsSet() {
		q.Set("start_time", o.StartTime.String())
	}
	if o.EndTime.IsSet() {
		q.Set("end_time", o.EndTime.String())
	}
	if s := o.Interval.String(); s != "" {
		q.Set("interval", s)
	}
}

// Bool returns a pointer to b, for the `resolved` bool filter which is
// tri-state (omitted / true / false).
func Bool(b bool) *bool { return &b }

// applyBoolFilter sets a bool query parameter from a *bool.
func applyBoolFilter(q url.Values, key string, v *bool) {
	if v == nil {
		return
	}
	if *v {
		q.Set(key, "true")
	} else {
		q.Set(key, "false")
	}
}

// applyStr sets a string query parameter when non-empty.
func applyStr(q url.Values, key, v string) {
	if v != "" {
		q.Set(key, v)
	}
}

// applyInt sets an int query parameter when positive.
func applyInt(q url.Values, key string, v int) {
	if v > 0 {
		q.Set(key, strconv.Itoa(v))
	}
}
