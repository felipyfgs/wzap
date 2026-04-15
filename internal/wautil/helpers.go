package wautil

import (
	"math"
	"time"
)

// StringPtr returns a pointer to s, or nil if s is empty.
func StringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// IntPtr returns a pointer to value.
func IntPtr(value int) *int {
	return &value
}

// UnixTimePtr converts a Unix timestamp (seconds) to a *time.Time, returning nil for non-positive values.
func UnixTimePtr(timestamp int64) *time.Time {
	if timestamp <= 0 {
		return nil
	}
	t := time.Unix(timestamp, 0)
	return &t
}

// Uint64ToInt64 converts a uint64 to int64, returning 0 if the value overflows.
func Uint64ToInt64(value uint64) int64 {
	if value > uint64(math.MaxInt64) {
		return 0
	}
	return int64(value)
}

// FirstNonEmpty returns the first non-empty string from values, or "" if all are empty.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
