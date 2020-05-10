package server_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/appleboy/gofight"
	"github.com/mdouchement/standardfile/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fastjson"
)

func TestRequestRegistration(t *testing.T) {
	engine, _, r, cleanup := setup()
	defer cleanup()

	params := gofight.D{
		"version": "003",
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
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No password cost provided."}}`, r.Body.String())
	})

	params["pw_cost"] = 110000
	r.POST("/auth").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		assert.Regexp(t, `.*\..*\..*`, string(v.Get("token").GetStringBytes()))
		assert.Equal(t, params["version"], string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, params["email"], string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, params["pw_nonce"], string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, params["pw_cost"], v.Get("user", "pw_cost").GetInt())

		timestamp, err := time.Parse("2006-01-02T15:04:05.999Z", string(v.Get("user", "created_at").GetStringBytes()))
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

func TestRequestParams(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	r.GET("/auth/params").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No email provided."}}`, r.Body.String())
	})

	r.GET("/auth/params?email=nobody@nowhere.lan").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Bad email provided."}}`, r.Body.String())
	})

	createUser(ioc)

	r.GET("/auth/params?email=george.abitbol@nowhere.lan").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.JSONEq(t, `{"identifier":"george.abitbol@nowhere.lan", "pw_cost":110000, "pw_nonce":"nonce42", "version":"003"}`, r.Body.String())
	})
}

func TestRequestLogin(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()
	user := createUser(ioc)

	r.POST("/auth/sign_in").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Could not get credentials."}}`, r.Body.String())
	})

	params := gofight.D{
		"email":    "",
		"password": "",
	}

	r.POST("/auth/sign_in").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No email or password provided."}}`, r.Body.String())
	})

	params["email"] = "george.abitbol@nowhere.lan"
	r.POST("/auth/sign_in").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No email or password provided."}}`, r.Body.String())
	})

	params["password"] = "password42"
	r.POST("/auth/sign_in").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		assert.Regexp(t, `.*\..*\..*`, string(v.Get("token").GetStringBytes()))
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost, v.Get("user", "pw_cost").GetInt())

		timestamp, err := time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
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

func TestRequestUpdate(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	r.POST("/auth/update").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"missing or malformed jwt"}}`, r.Body.String())
	})

	user := createUser(ioc)
	header := gofight.H{
		"Authorization": "Bearer " + server.TokenFromUser(ioc, user),
	}
	params := gofight.D{
		"pw_cost": user.PasswordCost * 2,
	}

	r.POST("/auth/update").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		assert.Equal(t, server.TokenFromUser(ioc, user), string(v.Get("token").GetStringBytes()))
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost*2, v.Get("user", "pw_cost").GetInt())

		timestamp, err := time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.CreatedAt.UTC(), timestamp.UTC())

		timestamp, err = time.Parse(time.RFC3339Nano, string(v.Get("user", "updated_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.WithinDuration(t, user.UpdatedAt.UTC(), timestamp.UTC(), 2*time.Second)
	})
}

func TestRequestUpdatePassword(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	r.POST("/auth/change_pw").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"missing or malformed jwt"}}`, r.Body.String())
	})

	user := createUser(ioc)
	header := gofight.H{
		"Authorization": "Bearer " + server.TokenFromUser(ioc, user),
	}
	params := gofight.D{
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

	params["current_password"] = "password42"
	r.POST("/auth/change_pw").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		assert.Equal(t, server.TokenFromUser(ioc, user), string(v.Get("token").GetStringBytes()))
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost, v.Get("user", "pw_cost").GetInt())

		timestamp, err := time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.CreatedAt.UTC(), timestamp.UTC())

		timestamp, err = time.Parse(time.RFC3339Nano, string(v.Get("user", "updated_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.WithinDuration(t, user.UpdatedAt.UTC(), timestamp.UTC(), 2*time.Second)
	})

	r.POST("/auth/change_pw").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Revoked token.","tag":"invalid-auth"}}`, r.Body.String())
	})
}
