package server_test

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/appleboy/gofight"
	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fastjson"
)

func TestRequestRegistration20200115(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	params := gofight.D{
		"api":     libsf.APIVersion20200115,
		"version": libsf.ProtocolVersion4,
	}
	r.POST("/auth").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No email provided."}}`, r.Body.String())
	})

	params["email"] = "george.abitbol@nowhere.lan"
	r.POST("/auth").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No password provided."}}`, r.Body.String())
	})

	params["password"] = "password42"
	r.POST("/auth").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No nonce provided."}}`, r.Body.String())
	})

	params["pw_nonce"] = "nonce42"
	r.POST("/auth").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		assert.Regexp(t, `^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+$`, string(v.Get("token").GetStringBytes()))
		//
		assert.Equal(t, params["version"], string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, params["email"], string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, params["pw_nonce"], string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, 0, v.Get("user", "pw_cost").GetInt())
		//
		assert.Regexp(t, `^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+$`, string(v.Get("session", "refresh_token").GetStringBytes()))

		timestamp, err := time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("session", "expire_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.InEpsilon(t, time.Now().Add(ioc.AccessTokenExpirationTime).UnixNano(), timestamp.UnixNano(), 500)

		timestamp, err = time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("session", "valid_until").GetStringBytes()))
		assert.NoError(t, err)
		assert.InEpsilon(t, time.Now().Add(ioc.RefreshTokenExpirationTime).UnixNano(), timestamp.UnixNano(), 500)

		//
		//

		timestamp, err = time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("user", "created_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Less(t, time.Since(timestamp).Nanoseconds(), (500 * time.Millisecond).Nanoseconds())

		timestamp, err = time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("user", "updated_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Less(t, time.Since(timestamp).Nanoseconds(), (500 * time.Millisecond).Nanoseconds())
	})

	r.POST("/auth").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"This email is already registered."}}`, r.Body.String())
	})
}

func TestRequestParams20200115(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	r.GET("/auth/params").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No email provided."}}`, r.Body.String())
	})

	r.GET("/auth/params?email=nobody@nowhere.lan").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		hostname, err := os.Hostname()
		assert.NoError(t, err)

		params := echo.Map{
			"identifier": "nobody@nowhere.lan",
			"nonce":      sha256.Sum256([]byte("nobody@nowhere.lan" + hostname)),
			"version":    libsf.ProtocolVersion4,
		}

		payload, err := json.Marshal(params)
		assert.NoError(t, err)

		assert.JSONEq(t, string(payload), r.Body.String())
	})

	createUser(ioc)

	r.GET("/auth/params?email=george.abitbol@nowhere.lan").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.JSONEq(t, `{"identifier":"george.abitbol@nowhere.lan", "pw_cost":110000, "pw_nonce":"nonce42", "version":"003"}`, r.Body.String())
	})
}

func TestRequestLogin20200115(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ioc.Database, ioc.SigningKey, ioc.AccessTokenExpirationTime, ioc.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ioc)

	r.POST("/auth/sign_in").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Could not get credentials."}}`, r.Body.String())
	})

	params := gofight.D{
		"api":      libsf.APIVersion20200115,
		"email":    "",
		"password": "",
	}

	r.POST("/auth/sign_in").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No email or password provided."}}`, r.Body.String())
	})

	params["email"] = "george.abitbol@nowhere.lan"
	r.POST("/auth/sign_in").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No email or password provided."}}`, r.Body.String())
	})

	params["password"] = "password42"
	r.POST("/auth/sign_in").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		assert.Regexp(t, `^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+$`, string(v.Get("token").GetStringBytes()))
		//
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost, v.Get("user", "pw_cost").GetInt())
		//
		assert.Regexp(t, `^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+$`, string(v.Get("session", "refresh_token").GetStringBytes()))

		timestamp, err := time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("session", "expire_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.InEpsilon(t, sessions.AccessTokenExprireAt(session).UnixNano(), timestamp.UnixNano(), 1000)

		timestamp, err = time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("session", "valid_until").GetStringBytes()))
		assert.NoError(t, err)
		assert.InEpsilon(t, session.ExpireAt.UnixNano(), timestamp.UnixNano(), 1000)

		//
		//

		timestamp, err = time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.CreatedAt.UTC(), timestamp.UTC())

		timestamp, err = time.Parse(time.RFC3339, string(v.Get("user", "updated_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.UpdatedAt.UTC(), timestamp.UTC())
	})

	params["email"] = "nobody@nowhere.lan"
	r.POST("/auth/sign_in").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Invalid email or password."}}`, r.Body.String())
	})
}

func TestRequestLogout20200115(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	_, session := createUserWithSession(ioc)

	//

	r.POST("/auth/sign_out").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + session.AccessToken,
	}

	r.POST("/auth/sign_out").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusNoContent, r.Code)
	})

	r.POST("/auth/sign_out").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})
}

func TestRequestUpdate20200115(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	r.POST("/auth/update").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	sessions := session.NewManager(ioc.Database, ioc.SigningKey, ioc.AccessTokenExpirationTime, ioc.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ioc)
	header := gofight.H{
		"Authorization": "Bearer " + session.AccessToken,
	}
	params := gofight.D{
		"api":     libsf.APIVersion20200115,
		"pw_cost": user.PasswordCost * 2, // TODO: rm?
	}

	r.POST("/auth/update").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		assert.Equal(t, session.AccessToken, string(v.Get("token").GetStringBytes()))
		//
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost*2, v.Get("user", "pw_cost").GetInt())
		//
		assert.Equal(t, session.RefreshToken, string(v.Get("session", "refresh_token").GetStringBytes()))

		timestamp, err := time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("session", "expire_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.InEpsilon(t, sessions.AccessTokenExprireAt(session).UnixNano(), timestamp.UnixNano(), 1000)

		timestamp, err = time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("session", "valid_until").GetStringBytes()))
		assert.NoError(t, err)
		assert.InEpsilon(t, session.ExpireAt.UnixNano(), timestamp.UnixNano(), 1000)

		//
		//

		timestamp, err = time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.CreatedAt.UTC(), timestamp.UTC())

		timestamp, err = time.Parse(time.RFC3339Nano, string(v.Get("user", "updated_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.WithinDuration(t, user.UpdatedAt.UTC(), timestamp.UTC(), 2*time.Second)
	})

	//

	session.ExpireAt = time.Now()
	err := ioc.Database.Save(session)
	assert.NoError(t, err)

	r.POST("/auth/update").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	//

	session.ExpireAt = time.Now().Add(time.Hour)
	err = ioc.Database.Save(session)
	assert.NoError(t, err)

	r.POST("/auth/update").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, sferror.StatusExpiredAccessToken, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"expired-access-token","message":"The provided access token has expired."}}`, r.Body.String())
	})
}

func TestRequestUpdatePassword20200115(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	r.POST("/auth/change_pw").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	sessions := session.NewManager(ioc.Database, ioc.SigningKey, ioc.AccessTokenExpirationTime, ioc.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ioc)
	header := gofight.H{
		"Authorization": "Bearer " + session.AccessToken,
	}
	params := gofight.D{
		"api":        libsf.APIVersion20200115,
		"identifier": user.Email,
	}

	r.POST("/auth/change_pw").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Your current password is required to change your password. Please update your application if you do not see this option."}}`, r.Body.String())
	})

	params["current_password"] = "trololo"
	r.POST("/auth/change_pw").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Your new password is required to change your password. Please update your application if you do not see this option."}}`, r.Body.String())
	})

	params["new_password"] = "yolo!"
	r.POST("/auth/change_pw").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"The current password you entered is incorrect. Please try again."}}`, r.Body.String())
	})

	time.Sleep(time.Second) // Ensure session issued at is older than 1s

	params["current_password"] = "password42"
	r.POST("/auth/change_pw").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		user, err = ioc.Database.FindUser(user.ID) // reload user
		assert.NoError(t, err)

		assert.Equal(t, session.AccessToken, string(v.Get("token").GetStringBytes()))
		//
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost, v.Get("user", "pw_cost").GetInt())
		assert.Equal(t, session.RefreshToken, string(v.Get("session", "refresh_token").GetStringBytes()))

		timestamp, err := time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("session", "expire_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.InEpsilon(t, sessions.AccessTokenExprireAt(session).UnixNano(), timestamp.UnixNano(), 1000)

		timestamp, err = time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("session", "valid_until").GetStringBytes()))
		assert.NoError(t, err)
		assert.InEpsilon(t, session.ExpireAt.UnixNano(), timestamp.UnixNano(), 1000)

		//
		//

		timestamp, err = time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.CreatedAt.UTC(), timestamp.UTC())

		timestamp, err = time.Parse(time.RFC3339Nano, string(v.Get("user", "updated_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.WithinDuration(t, user.UpdatedAt.UTC(), timestamp.UTC(), 2*time.Second)
	})

	r.POST("/auth/change_pw").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"The current password you entered is incorrect. Please try again."}}`, r.Body.String())
	})
}
