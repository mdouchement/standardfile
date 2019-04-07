package libsf

import (
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// TimeFromToken retrieves datetime from cursor/sync token.
func TimeFromToken(token string) time.Time {
	raw, err := base64.URLEncoding.DecodeString(token) // meh, there aren't none ASCII characters in a Unix timestamp.
	if err != nil {
		log.Println("GetTimeFromToken:", err)
		return time.Now().UTC()
	}

	parts := strings.Split(string(raw), ":")
	if parts[0] == "1" {
		// Do not support v1 `1:474536275' (Unix timestamp in seconds)
		log.Println("GetTimeFromToken: unsupported v1 token date")
		return time.Now().UTC()
	}

	// v2 token `1:4745362752134567' (Unix timestamp in nanoseconds)
	timestamp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		log.Println("GetTimeFromToken:", err)
		return time.Now().UTC()
	}
	return time.Unix(0, timestamp).UTC()
}

// TokenFromTime generates cursor/sync token for given time.
func TokenFromTime(t time.Time) (token string) {
	token = fmt.Sprintf("2:%d", t.UTC().UnixNano())

	// meh, there aren't none ASCII characters in a Unix timestamp.
	return base64.URLEncoding.EncodeToString([]byte(token))
}
