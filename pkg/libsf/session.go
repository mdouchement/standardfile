package libsf

import (
	"encoding/json"
	"time"
)

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

func (s *Session) UnmarshalJSON(data []byte) error {
	session := struct {
		AccessToken       string `json:"access_token"`
		RefreshToken      string `json:"refresh_token"`
		AccessExpiration  int64  `json:"access_expiration"`
		RefreshExpiration int64  `json:"refresh_expiration"`
	}{}

	err := json.Unmarshal(data, &session)
	if err != nil {
		return err
	}

	s.AccessToken = session.AccessToken
	s.RefreshToken = session.RefreshToken
	s.AccessExpiration = time.UnixMilli(session.AccessExpiration)
	s.RefreshExpiration = time.UnixMilli(session.RefreshExpiration)
	return nil
}

func (s Session) MarshalJSON() ([]byte, error) {
	session := struct {
		AccessToken       string `json:"access_token"`
		RefreshToken      string `json:"refresh_token"`
		AccessExpiration  int64  `json:"access_expiration"`
		RefreshExpiration int64  `json:"refresh_expiration"`
	}{
		AccessToken:       s.AccessToken,
		RefreshToken:      s.RefreshToken,
		AccessExpiration:  s.AccessExpiration.UnixMilli(),
		RefreshExpiration: s.RefreshExpiration.UnixMilli(),
	}

	return json.Marshal(session)
}
