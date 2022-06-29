package model

import (
	"time"
)

// A PKCE represents a database record.
type PKCE struct {
	Base `msgpack:",inline" storm:"inline"`

	CodeChallenge string    `msgpack:"code_challenge"       storm:"index"`
	ExpireAt      time.Time `msgpack:"expire_at"`
}
