package server_test

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/appleboy/gofight/v2"
	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fastjson"
)

func TestRequestRegistration20200115(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
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

		//
		assert.Equal(t, params["version"], string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, params["email"], string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, params["pw_nonce"], string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, 0, v.Get("user", "pw_cost").GetInt())
		//
		assert.Regexp(t, `^v2.local.[A-Za-z0-9+_\-]+$`, string(v.Get("session", "access_token").GetStringBytes()))
		assert.Regexp(t, `^v2.local.[A-Za-z0-9+_\-]+$`, string(v.Get("session", "refresh_token").GetStringBytes()))

		assert.InEpsilon(t, time.Now().Add(ctrl.AccessTokenExpirationTime).UnixNano()/int64(time.Millisecond), v.GetInt64("session", "access_expiration"), 500)
		assert.InEpsilon(t, time.Now().Add(ctrl.RefreshTokenExpirationTime).UnixNano()/int64(time.Millisecond), v.GetInt64("session", "refresh_expiration"), 500)

		//
		//

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

func TestRequestParams20200115(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
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

	createUser(ctrl)

	r.GET("/auth/params?email=george.abitbol@nowhere.lan").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.JSONEq(t, `{"identifier":"george.abitbol@nowhere.lan", "pw_cost":110000, "pw_nonce":"nonce42", "version":"003"}`, r.Body.String())
	})
}

func TestRequestLogin20200115(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ctrl.Database, ctrl.SigningKey, ctrl.SessionSecret, ctrl.AccessTokenExpirationTime, ctrl.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ctrl)

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

		//
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost, v.Get("user", "pw_cost").GetInt())
		//
		assert.Regexp(t, `^v2.local.[A-Za-z0-9+_\-]+$`, string(v.Get("session", "access_token").GetStringBytes()))
		assert.Regexp(t, `^v2.local.[A-Za-z0-9+_\-]+$`, string(v.Get("session", "refresh_token").GetStringBytes()))

		assert.InEpsilon(t, sessions.AccessTokenExprireAt(session).UnixNano()/int64(time.Millisecond), v.GetInt64("session", "access_expiration"), 1000)
		assert.InEpsilon(t, session.ExpireAt.UnixNano()/int64(time.Millisecond), v.GetInt64("session", "refresh_expiration"), 1000)

		//
		//

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

func TestRequestLogout20200115(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	_, session := createUserWithSession(ctrl)

	//

	r.POST("/auth/sign_out").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, session),
	}

	r.POST("/auth/sign_out").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusNoContent, r.Code, r.Body.String())
	})

	r.POST("/auth/sign_out").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})
}

func TestRequestLoginPKCE20200115(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ctrl.Database, ctrl.SigningKey, ctrl.SessionSecret, ctrl.AccessTokenExpirationTime, ctrl.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ctrl)

	params := gofight.D{
		"api":   libsf.APIVersion20200115,
		"email": "george.abitbol@nowhere.lan",
	}
	r.POST("/v2/login-params").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Please provide the code challenge parameter"}}`, r.Body.String())
	})

	params["code_challenge"] = "MTFjYmFiZmNhODU5MTJlNWYxMzNhOGY0YWI2OWY4MzQ1ZTZhMDZlNDVjOTU5NjQ0YWQ5ZmFlOTA5NWY4MmZmNA"
	r.POST("/v2/login-params").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		//
		assert.Equal(t, user.Version, string(v.Get("version").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("identifier").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("pw_nonce").GetStringBytes()))
	})

	r.POST("/v2/login-params").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		//
		assert.Equal(t, user.Version, string(v.Get("version").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("identifier").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("pw_nonce").GetStringBytes()))
	})

	params["password"] = "password42"
	r.POST("/v2/login").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Invalid login credentials."}}`, r.Body.String())
	})

	params["code_verifier"] = "90308e36cbb7051f2f97634f794e5e323fb8d06d6076c1ed0f7e45bb704ebce1"
	r.POST("/v2/login").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		//
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost, v.Get("user", "pw_cost").GetInt())
		//
		assert.Regexp(t, `^v2.local.[A-Za-z0-9+_\-]+$`, string(v.Get("session", "access_token").GetStringBytes()))
		assert.Regexp(t, `^v2.local.[A-Za-z0-9+_\-]+$`, string(v.Get("session", "refresh_token").GetStringBytes()))

		assert.InEpsilon(t, sessions.AccessTokenExprireAt(session).UnixNano()/int64(time.Millisecond), v.GetInt64("session", "access_expiration"), 1000)
		assert.InEpsilon(t, session.ExpireAt.UnixNano()/int64(time.Millisecond), v.GetInt64("session", "refresh_expiration"), 1000)

		//
		//

		timestamp, err := time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.CreatedAt.UTC(), timestamp.UTC())

		timestamp, err = time.Parse(time.RFC3339, string(v.Get("user", "updated_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.UpdatedAt.UTC(), timestamp.UTC())
	})

}

func TestRequestUpdate20200115(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	r.POST("/auth/update").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	sessions := session.NewManager(ctrl.Database, ctrl.SigningKey, ctrl.SessionSecret, ctrl.AccessTokenExpirationTime, ctrl.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ctrl)
	header := gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, session),
	}
	params := gofight.D{
		"api":     libsf.APIVersion20200115,
		"pw_cost": user.PasswordCost * 2,
	}

	r.POST("/auth/update").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		//
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost*2, v.Get("user", "pw_cost").GetInt())
		//
		sid, token, err := sessions.ParseToken(string(v.Get("session", "access_token").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, session.ID, sid)
		assert.Equal(t, session.AccessToken, token)

		sid, token, err = sessions.ParseToken(string(v.Get("session", "refresh_token").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, session.ID, sid)
		assert.Equal(t, session.RefreshToken, token)

		assert.InEpsilon(t, sessions.AccessTokenExprireAt(session).UnixNano()/int64(time.Millisecond), v.GetInt64("session", "access_expiration"), 1000)
		assert.InEpsilon(t, session.ExpireAt.UnixNano()/int64(time.Millisecond), v.GetInt64("session", "refresh_expiration"), 1000)

		//
		//

		timestamp, err := time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, user.CreatedAt.UTC(), timestamp.UTC())

		timestamp, err = time.Parse(time.RFC3339Nano, string(v.Get("user", "updated_at").GetStringBytes()))
		assert.NoError(t, err)
		assert.WithinDuration(t, user.UpdatedAt.UTC(), timestamp.UTC(), 2*time.Second)
	})

	//

	session.ExpireAt = time.Now()
	err := ctrl.Database.Save(session)
	assert.NoError(t, err)

	r.POST("/auth/update").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	//

	session.ExpireAt = time.Now().Add(time.Hour)
	err = ctrl.Database.Save(session)
	assert.NoError(t, err)

	r.POST("/auth/update").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, sferror.StatusExpiredAccessToken, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"expired-access-token","message":"The provided access token has expired."}}`, r.Body.String())
	})
}

func TestRequestUpdatePassword20200115(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	r.POST("/auth/change_pw").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	sessions := session.NewManager(ctrl.Database, ctrl.SigningKey, ctrl.SessionSecret, ctrl.AccessTokenExpirationTime, ctrl.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ctrl)
	header := gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, session),
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

		user, err = ctrl.Database.FindUser(user.ID) // reload user
		assert.NoError(t, err)

		//
		assert.Equal(t, user.Version, string(v.Get("user", "version").GetStringBytes()))
		assert.Regexp(t, `^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`, string(v.Get("user", "uuid").GetStringBytes()))
		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.PasswordNonce, string(v.Get("user", "pw_nonce").GetStringBytes()))
		assert.Equal(t, user.PasswordCost, v.Get("user", "pw_cost").GetInt())
		//
		sid, token, err := sessions.ParseToken(string(v.Get("session", "access_token").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, session.ID, sid)
		assert.Equal(t, session.AccessToken, token)

		sid, token, err = sessions.ParseToken(string(v.Get("session", "refresh_token").GetStringBytes()))
		assert.NoError(t, err)
		assert.Equal(t, session.ID, sid)
		assert.Equal(t, session.RefreshToken, token)

		assert.InEpsilon(t, sessions.AccessTokenExprireAt(session).UnixNano()/int64(time.Millisecond), v.GetInt64("session", "access_expiration"), 1000)
		assert.InEpsilon(t, session.ExpireAt.UnixNano()/int64(time.Millisecond), v.GetInt64("session", "refresh_expiration"), 1000)

		//
		//

		timestamp, err := time.Parse(time.RFC3339, string(v.Get("user", "created_at").GetStringBytes()))
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

	// If no email available, ensure no email updates
	params["current_password"] = "yolo!"
	r.PUT("/v1/users/"+user.ID+"/attributes/credentials").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		user, err := ctrl.Database.FindUser(user.ID) // reload user
		assert.NoError(t, err)

		assert.Equal(t, user.Email, "george.abitbol@nowhere.lan")
	})

	// If a new email is provided, check it's correctly updated
	params["current_password"] = "yolo!"
	params["new_email"] = "test@test.de"
	r.PUT("/v1/users/"+user.ID+"/attributes/credentials").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		v, err := fastjson.Parse(r.Body.String())
		assert.NoError(t, err)

		user, err = ctrl.Database.FindUser(user.ID) // reload user
		assert.NoError(t, err)

		assert.Equal(t, user.Email, string(v.Get("user", "email").GetStringBytes()))
		assert.Equal(t, user.Email, params["new_email"])
	})

	// If a new email is provided, but already used
	user2, _ := createUserWithSession(ctrl)
	params["current_password"] = "yolo!"
	params["new_email"] = user2.Email
	r.PUT("/v1/users/"+user.ID+"/attributes/credentials").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code, r.Body.String())
		assert.JSONEq(t, `{"error":{"message":"The email you entered is already taken. Please try again."}}`, r.Body.String())
	})

	// If the user id parameter is different from the request's bearer token
	params["current_password"] = "yolo!"
	params["new_email"] = "test@test.de"
	r.PUT("/v1/users/DIFFERENT-ID-THAN-IN-BEARER-TOKEN/attributes/credentials").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code, r.Body.String())
		assert.JSONEq(t, `{"error":{"message":"The given ID is not the user's one."}}`, r.Body.String())
	})
}
