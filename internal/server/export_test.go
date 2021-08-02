package server

import (
	"log"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/mdouchement/standardfile/internal/model"
)

// This file is only for test purpose and is only loaded by test framework.

// CreateJWT returns a JWT token.
func CreateJWT(ctrl Controller, u *model.User) string {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_uuid"] = u.ID
	// claims["pw_hash"] = fmt.Sprintf("%x", sha256.Sum256([]byte(u.Password))) // See readme
	claims["iss"] = "github.com/mdouchement/standardfile"
	claims["iat"] = time.Now().Unix() // Unix Timestamp in seconds

	t, err := token.SignedString(ctrl.SigningKey)
	if err != nil {
		log.Fatalf("could not generate token: %s", err)
	}
	return t
}
