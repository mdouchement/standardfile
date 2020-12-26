package libsf

import "time"

// A Session contains details about a session.
type Session struct {
	AccessToken       string    `json:"access_token"`
	RefreshToken      string    `json:"refresh_token"`
	AccessExpiration  time.Time `json:"access_expiration"`
	RefreshExpiration time.Time `json:"refresh_expiration"`
}

// Defined returns true if session's fields are defined.
func (s Session) Defined() bool {
	return s.AccessToken != "" && s.RefreshToken != "" &&
		!s.AccessExpiration.IsZero() && !s.RefreshExpiration.IsZero()
}

// AccessExpiredAt returns true if the access token is expired at the given time.
func (s Session) AccessExpiredAt(t time.Time) bool {
	return !s.Defined() || t.After(s.AccessExpiration)
}

// AccessExpired returns true if the access token is expired.
func (s Session) AccessExpired() bool {
	return s.AccessExpiredAt(time.Now())
}

// RefreshExpired returns true if the refresh token is expired.
func (s Session) RefreshExpired() bool {
	return !s.Defined() || time.Now().After(s.RefreshExpiration)
}
