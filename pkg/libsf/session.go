package libsf

import (
	"encoding/json"
	"strconv"
	"strings"
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

type sessionTime time.Time

func (t *sessionTime) UnmarshalJSON(s []byte) error {
	r := strings.Replace(string(s), `"`, ``, -1)
	tt, err := time.Parse(time.RFC3339, r)
	if err != nil {
		q, err := strconv.ParseInt(string(s), 10, 64)
		if err != nil {
			return err
		}
		*(*time.Time)(t) = time.Unix(q/1000, 0)
		return nil
	}
	*(*time.Time)(t) = tt
	return nil
}

func (s *Session) UnmarshalJSON(raw []byte) error {
	st := &struct {
		AccessToken       string      `json:"access_token"`
		RefreshToken      string      `json:"refresh_token"`
		AccessExpiration  sessionTime `json:"access_expiration"`
		RefreshExpiration sessionTime `json:"refresh_expiration"`
	}{}

	err := json.Unmarshal(raw, st)
	if err != nil {
		return err
	}

	s.AccessToken = st.AccessToken
	s.RefreshToken = st.RefreshToken
	s.AccessExpiration = time.Time(st.AccessExpiration)
	s.RefreshExpiration = time.Time(st.RefreshExpiration)

	return nil
}
