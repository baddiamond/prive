package store

import "time"

// toMillis stores times as Unix milliseconds; the zero time becomes 0.
func toMillis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// fromMillis is the inverse of toMillis and returns UTC times.
func fromMillis(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
