package session

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mdouchement/middlewarex"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/o1egl/paseto/v2"
	"github.com/pkg/errors"
)

// Defines token types.
const (
	TypeAccessToken  = "access_token"
	TypeRefreshToken = "refresh_token"
)

type (
	// A Manager manages sessions.
	Manager interface {
		JWTSigningKey() []byte
		SessionSecret() []byte
		// Token generates the session's token for the given type t.
		Token(session *model.Session, t string) (string, error)
		// ParseToken parses the given raw token and returns the session_id and token.
		ParseToken(token string) (string, string, error)
		// Generate creates a new session without user information.
		Generate() *model.Session
		// Validate validates an access token.
		Validate(userID, token string) (*model.Session, error)
		// AccessTokenExprireAt returns the expiration date of the access token.
		AccessTokenExprireAt(session *model.Session) time.Time
		// Regenerate regenerates the session's tokens.
		Regenerate(session *model.Session) error
		// UserFromToken the user for the given token.
		UserFromToken(token interface{}) (*model.User, error)
	}

	manager struct {
		db database.Client
		// JWT params
		signingKey []byte
		// Session params
		sessionSecret              []byte
		accessTokenExpirationTime  time.Duration
		refreshTokenExpirationTime time.Duration
	}
)

// NewManager returns a new manager.
func NewManager(db database.Client, signingKey, sessionSecret []byte, accessTokenExpirationTime, refreshTokenExpirationTime time.Duration) Manager {
	return &manager{
		db:                         db,
		signingKey:                 signingKey,
		sessionSecret:              sessionSecret,
		accessTokenExpirationTime:  accessTokenExpirationTime,
		refreshTokenExpirationTime: refreshTokenExpirationTime,
	}
}

func (m *manager) JWTSigningKey() []byte {
	return m.signingKey
}

func (m *manager) SessionSecret() []byte {
	return m.sessionSecret
}

func (m *manager) Token(session *model.Session, t string) (string, error) {
	iat := session.ExpireAt.Add(-m.refreshTokenExpirationTime)

	claims := &paseto.JSONToken{
		Issuer:     "standardfile",
		Audience:   t,
		Subject:    session.ID,
		IssuedAt:   iat.UTC(),
		NotBefore:  iat.UTC(),
		Expiration: time.Now().Add(-72 * time.Hour).UTC(),
	}

	switch t {
	case TypeAccessToken:
		claims.Jti = session.AccessToken
		claims.Expiration = m.AccessTokenExprireAt(session)
	case TypeRefreshToken:
		claims.Jti = session.RefreshToken
		claims.Expiration = session.ExpireAt
	}

	return paseto.Encrypt(m.sessionSecret, claims, []byte{})
}

func (m *manager) ParseToken(token string) (string, string, error) {
	var tk middlewarex.Token
	err := paseto.Decrypt(token, m.sessionSecret, &tk.JSONToken, &tk.Footer)
	return tk.Subject, tk.Jti, err
}

func (m *manager) Generate() *model.Session {
	return &model.Session{
		ExpireAt:     time.Now().Add(m.refreshTokenExpirationTime).UTC(),
		AccessToken:  SecureToken(8),
		RefreshToken: SecureToken(8),
	}
}

func (m *manager) Validate(id, token string) (*model.Session, error) {
	// Check if there is an active session.
	session, err := m.db.FindSessionByAccessToken(id, token)
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
	if m.isSessionExpired(session) {
		return nil, sferror.NewWithTagCode(http.StatusUnauthorized, "invalid-auth", "Invalid login credentials.")
	}

	if m.isAccessTokenExpired(session) {
		return nil, sferror.NewWithTagCode(sferror.StatusExpiredAccessToken, "expired-access-token", "The provided access token has expired.")
	}

	return session, nil
}

func (m *manager) AccessTokenExprireAt(session *model.Session) time.Time {
	return session.ExpireAt.Add(-m.refreshTokenExpirationTime).Add(m.accessTokenExpirationTime)
}

func (m *manager) Regenerate(session *model.Session) error {
	if m.isSessionExpired(session) {
		return sferror.NewWithTagCode(
			http.StatusBadRequest,
			"expired-refresh-token",
			"The refresh token has expired.",
		)
	}

	session.AccessToken = SecureToken(8)
	session.RefreshToken = SecureToken(8)
	session.ExpireAt = time.Now().Add(m.refreshTokenExpirationTime)

	return errors.Wrap(m.db.Save(session), "could not save session after refreshing session")
}

func (m *manager) UserFromToken(token interface{}) (*model.User, error) {
	if jwt, ok := token.(*jwt.Token); ok {
		return m.JWT(jwt)
	}
	return m.Paseto(token.(middlewarex.Token))
}

func (m *manager) Paseto(token middlewarex.Token) (*model.User, error) {
	session, err := m.Validate(token.Subject, token.Jti)
	if err != nil {
		return nil, err
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

func (m *manager) isSessionExpired(session *model.Session) bool {
	return session.ExpireAt.Before(time.Now())
}

func (m *manager) isAccessTokenExpired(session *model.Session) bool {
	return m.AccessTokenExprireAt(session).Before(time.Now())
}
