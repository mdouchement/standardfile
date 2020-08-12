package session

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/pkg/errors"
)

type (
	// A Manager manages sessions.
	Manager interface {
		JWTSigningKey() []byte
		UserFromToken(token interface{}) (*model.User, error)
	}

	manager struct {
		db database.Client
		// JWT params
		signingKey []byte
		// Session params
		accessTokenExpirationTime  time.Duration
		refreshTokenExpirationTime time.Duration
	}
)

// NewManager returns a new manager.
func NewManager(db database.Client, signingKey []byte, accessTokenExpirationTime, refreshTokenExpirationTime time.Duration) Manager {
	return &manager{
		db:                         db,
		signingKey:                 signingKey,
		accessTokenExpirationTime:  accessTokenExpirationTime,
		refreshTokenExpirationTime: refreshTokenExpirationTime,
	}
}

func (m *manager) JWTSigningKey() []byte {
	return m.signingKey
}

func (m *manager) UserFromToken(token interface{}) (*model.User, error) {
	if jwt, ok := token.(*jwt.Token); ok {
		return m.JWT(jwt)
	}
	return m.SessionToken(token.(string))
}

func (m *manager) SessionToken(token string) (*model.User, error) {
	session, err := m.db.FindSessionByAccessToken(token)
	if err != nil {
		if m.db.IsNotFound(err) {
			return nil, sferror.NewWithTagCode(
				http.StatusUnauthorized,
				"invalid-auth",
				"Invalid login credentials.",
			)
		}
		return nil, errors.Wrap(err, "could not get access to database")
	}

	// Validate session.
	now := time.Now()
	if session.ExpireAt.After(now) {
		return nil, sferror.NewWithTagCode(sferror.StatusExpiredAccessToken, "invalid-auth", "Invalid login credentials.")
	}

	expireAt := session.ExpireAt.
		Add(-m.refreshTokenExpirationTime).
		Add(m.accessTokenExpirationTime)
	if expireAt.After(now) {
		return nil, sferror.NewWithTagCode(sferror.StatusExpiredAccessToken, "expired-access-token", "The provided access token has expired.")
	}

	// Get current_user.
	user, err := m.db.FindUser(session.UserID)
	if err != nil {
		if m.db.IsNotFound(err) {
			return nil, sferror.NewWithTagCode(
				http.StatusUnauthorized,
				"invalid-auth",
				"Invalid login credentials.",
			)
		}
		return nil, errors.Wrap(err, "could not get access to database")
	}

	return user, nil
}

func (m *manager) JWT(token *jwt.Token) (*model.User, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		panic("token implementation has wrong type of claims")
	}

	// Get current_user.
	user, err := m.db.FindUser(claims["user_uuid"].(string))
	if err != nil {
		if m.db.IsNotFound(err) {
			return nil, sferror.NewWithTagCode(
				http.StatusUnauthorized,
				"invalid-auth",
				"Invalid login credentials.",
			)
		}
		return nil, errors.Wrap(err, "could not get access to database")
	}

	// Check if password has changed since token was generated.
	var iat int64
	switch v := claims["iat"].(type) {
	case float64:
		iat = int64(v)
	case json.Number:
		iat, _ = v.Int64()
	default:
		panic("unsuported iat underlying type")
	}

	if iat < user.PasswordUpdatedAt {
		return nil, sferror.NewWithTagCode(http.StatusUnauthorized, "invalid-auth", "Revoked token.")
	}

	return user, nil
}
