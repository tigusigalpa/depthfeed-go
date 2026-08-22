package depthfeed

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Timestamp is an API timestamp value. DepthFeed accepts either
// milliseconds-since-epoch (e.g. 1780774485780) or ISO-8601
// (e.g. 2026-06-07T02:27:09Z). Timestamp preserves full millisecond
// precision and serializes back to a string the API accepts.
//
// Use NewTimestamp to build one from a time.Time, epoch millis, or an
// ISO-8601 string. The zero value serializes as the empty string and is
// omitted from query parameters by the option helpers.
type Timestamp struct {
	ms  int64
	set bool
}

// NewTimestampFromTime builds a Timestamp from a time.Time, preserving
// millisecond precision.
func NewTimestampFromTime(t time.Time) Timestamp {
	return Timestamp{ms: t.UnixMilli(), set: true}
}

// NewTimestampFromMillis builds a Timestamp from epoch milliseconds.
func NewTimestampFromMillis(ms int64) Timestamp {
	return Timestamp{ms: ms, set: true}
}

// NewTimestamp parses an ISO-8601 string or an epoch-millis string. An
// empty string returns the zero Timestamp (unset).
func NewTimestamp(s string) (Timestamp, error) {
	if s == "" {
		return Timestamp{}, nil
	}
	// Try epoch millis first (all digits, possibly negative).
	if isAllDigits(s) {
		ms, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return Timestamp{}, fmt.Errorf("depthfeed: invalid timestamp %q: %w", s, err)
		}
		return Timestamp{ms: ms, set: true}, nil
	}
	// Try a few common ISO-8601 layouts.
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return Timestamp{ms: t.UnixMilli(), set: true}, nil
		}
	}
	return Timestamp{}, fmt.Errorf("depthfeed: timestamp %q is not ms-epoch or ISO-8601", s)
}

// IsSet reports whether the Timestamp has a value.
func (t Timestamp) IsSet() bool { return t.set }

// Millis returns the epoch-millisecond value.
func (t Timestamp) Millis() int64 { return t.ms }

// Time returns the value as a time.Time (UTC).
func (t Timestamp) Time() time.Time {
	return time.UnixMilli(t.ms).UTC()
}

// String returns the value the API accepts: epoch-millis when set, empty
// string when unset. Epoch-millis is the safest wire form because it is
// unambiguous and preserves sub-second precision exactly.
func (t Timestamp) String() string {
	if !t.set {
		return ""
	}
	return strconv.FormatInt(t.ms, 10)
}

// MarshalJSON serializes the timestamp as a JSON string (epoch-millis).
func (t Timestamp) MarshalJSON() ([]byte, error) {
	if !t.set {
		return json.Marshal("")
	}
	return json.Marshal(t.String())
}

// UnmarshalJSON accepts a JSON string (ISO-8601 or epoch-ms) or a JSON
// number (epoch-ms).
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	// Number?
	if data[0] != '"' {
		var n json.Number
		if err := json.Unmarshal(data, &n); err != nil {
			return err
		}
		ms, err := n.Int64()
		if err != nil {
			f, err2 := n.Float64()
			if err2 != nil {
				return err
			}
			ms = int64(f)
		}
		t.ms = ms
		t.set = true
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := NewTimestamp(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r < '0' || r > '9' {
			if i == 0 && r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

// Interval is a snapshot downsampling duration (30s, 1m, 5m, 1h … up to
// 1d). Omit for full resolution. See https://depthfeed.com/docs#pagination.
type Interval string

// Documented interval units. A bare number of seconds is also accepted.
const (
	IntervalRaw Interval = "raw"
	Interval30s Interval = "30s"
	Interval1m  Interval = "1m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval1h  Interval = "1h"
	Interval4h  Interval = "4h"
	Interval1d  Interval = "1d"
)

// String returns the interval value, omitting "raw" (the API treats an
// omitted interval as full resolution).
func (i Interval) String() string {
	s := string(i)
	if s == "" || strings.EqualFold(s, "raw") || s == "0" {
		return ""
	}
	return s
}
