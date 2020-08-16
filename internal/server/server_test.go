package server_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/appleboy/gofight"
	"github.com/labstack/echo/v4"
	argon2 "github.com/mdouchement/simple-argon2"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/stretchr/testify/assert"
)

func TestRequestHome(t *testing.T) {
	engine, _, r, cleanup := setup()
	defer cleanup()

	r.GET("/").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.JSONEq(t, `{"version":"test"}`, r.Body.String())
	})
}

func TestRequestVersion(t *testing.T) {
	engine, _, r, cleanup := setup()
	defer cleanup()

	r.GET("/version").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.JSONEq(t, `{"version":"test"}`, r.Body.String())
	})
}

func setup() (engine *echo.Echo, ioc server.IOC, r *gofight.RequestConfig, cleanup func()) {
	tmpfile, err := ioutil.TempFile("", "standardfile.*.db")
	if err != nil {
		panic(err)
	}
	filename := tmpfile.Name()
	tmpfile.Close()

	db, err := database.StormOpen(filename)
	if err != nil {
		panic(err)
	}

	ioc = server.IOC{
		Version:                    "test",
		Database:                   db,
		NoRegistration:             false,
		SigningKey:                 []byte("secret"),
		AccessTokenExpirationTime:  60 * 24 * time.Hour,
		RefreshTokenExpirationTime: 365 * 24 * time.Hour,
	}
	engine = server.EchoEngine(ioc)

	return engine, ioc, gofight.New(), func() {
		db.Close()
		os.RemoveAll(filename)
	}
}

func createUser(ioc server.IOC) *model.User {
	var err error
	t := time.Now()

	user := model.NewUser()
	user.CreatedAt = &t
	user.UpdatedAt = &t
	user.Email = "george.abitbol@nowhere.lan"
	user.Version = model.VersionLatest
	user.Password, err = argon2.GenerateFromPasswordString("password42", argon2.Default)
	user.PasswordCost = 110000
	user.PasswordNonce = "nonce42"
	user.PasswordUpdatedAt = time.Now().Add(-12 * time.Hour).Unix()
	if err != nil {
		panic(err)
	}
	err = ioc.Database.Save(user)
	if err != nil {
		panic(err)
	}

	return user
}

func createUserWithSession(ioc server.IOC) (*model.User, *model.Session) {
	var err error

	user := model.NewUser()
	user.Email = "george.abitbol@nowhere.lan"
	user.Version = model.Version4
	user.Password, err = argon2.GenerateFromPasswordString("password42", argon2.Default)
	user.PasswordCost = 110000
	user.PasswordNonce = "nonce42"
	user.PasswordUpdatedAt = time.Now().Add(-12 * time.Hour).Unix()
	if err != nil {
		panic(err)
	}
	err = ioc.Database.Save(user)
	if err != nil {
		panic(err)
	}

	session := &model.Session{
		APIVersion:   "20200115",
		UserAgent:    "Go-http-client/1.1",
		UserID:       user.ID,
		ExpireAt:     time.Now().Add(ioc.RefreshTokenExpirationTime).UTC(),
		AccessToken:  session.SecureToken(24),
		RefreshToken: session.SecureToken(24),
	}
	err = ioc.Database.Save(session)
	if err != nil {
		panic(err)
	}

	return user, session
}
