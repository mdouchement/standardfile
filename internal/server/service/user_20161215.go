package service

import (
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server/serializer"
)

type userService20161215 struct {
	userServiceBase
}

func (s *userService20161215) Register(params RegisterParams) (Render, error) {
	return s.register(params, s.SuccessfulAuthentication)
}

func (s *userService20161215) Login(params LoginParams) (Render, error) {
	return s.login(params, s.SuccessfulAuthentication)
}

func (s *userService20161215) Update(user *model.User, params UpdateUserParams) (Render, error) {
	return s.update(user, params, s.SuccessfulAuthentication)
}

func (s *userService20161215) Password(user *model.User, params UpdatePasswordParams) (Render, error) {
	return s.password(user, params, s.SuccessfulAuthentication)
}

func (s *userService20161215) SuccessfulAuthentication(u *model.User, _ Params) (Render, error) {
	return echo.Map{
		"user":  serializer.User(u),
		"token": s.CreateJWT(u),
	}, nil
}

func (s *userService20161215) CreateJWT(u *model.User) string {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_uuid"] = u.ID
	// claims["pw_hash"] = fmt.Sprintf("%x", sha256.Sum256([]byte(u.Password))) // See readme
	claims["iss"] = "github.com/mdouchement/standardfile"
	claims["iat"] = time.Now().Unix() // Unix Timestamp in seconds

	t, err := token.SignedString(s.sessions.JWTSigningKey())
	if err != nil {
		log.Fatalf("could not generate token: %s", err)
	}
	return t
}
