package model

import (
	"time"
)

// A Session represents a database record.
type Session struct {
	Base `msgpack:",inline" storm:"inline"`

	ExpireAt     time.Time `msgpack:"expire_at"`
	UserID       string    `msgpack:"user_id"       storm:"index"`
	UserAgent    string    `msgpack:"user_agent"`
	APIVersion   string    `msgpack:"api_version"`
	AccessToken  string    `msgpack:"access_token"  storm:"unique"`
	RefreshToken string    `msgpack:"refresh_token" storm:"unique"`
}
