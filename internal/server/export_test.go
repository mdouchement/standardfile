package server

import (
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server/session"
)

// This file is only for test purpose and is only loaded by test framework.

// TokenFromUser returns JWT tokens.
func TokenFromUser(ctrl IOC, u *model.User) string {
	sessions := session.NewManager(
		ctrl.Database,
		ctrl.SigningKey,
		ctrl.AccessTokenExpirationTime,
		ctrl.RefreshTokenExpirationTime,
	)

	a := &auth{sessions: sessions}
	return "a.TokenFromUser(u)"
}
