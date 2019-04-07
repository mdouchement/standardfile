package server

import "github.com/mdouchement/standardfile/internal/model"

// This file is only for test purpose and is only loaded by test framework.

// TokenFromUser returns JWT tokens.
func TokenFromUser(ioc IOC, u *model.User) string {
	a := &auth{signingKey: ioc.SigningKey}
	return a.TokenFromUser(u)
}
