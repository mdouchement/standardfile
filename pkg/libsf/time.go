package libsf

import "time"

// UnixMillisecond returns a unix timestamp in milliseconds.
func UnixMillisecond(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// FromUnixMillisecond returns a time based on the given unix timestamp in milliseconds.
func FromUnixMillisecond(t int64) time.Time {
	return time.Unix(0, t*int64(time.Millisecond))
}
