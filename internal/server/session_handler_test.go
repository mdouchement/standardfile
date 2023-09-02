package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/appleboy/gofight/v2"
	"github.com/gofrs/uuid"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/stretchr/testify/assert"
)

func TestRequestSessionMiddleware(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	user, validsession := createUserWithSession(ctrl)

	//
	// No token.
	//

	r.GET("/sessions").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	//
	// Valid token encryption but does not existing in database.
	//

	invalidsession := &model.Session{
		APIVersion:   "20200115",
		UserAgent:    "Go-http-client/1.1",
		UserID:       "fake_id",
		ExpireAt:     time.Now().Add(ctrl.RefreshTokenExpirationTime).UTC(),
		AccessToken:  session.SecureToken(8),
		RefreshToken: session.SecureToken(8),
	}

	header := gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, invalidsession),
	}

	r.GET("/sessions").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code, r.Body.String())
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	//
	// Expired access token.
	//

	original := ctrl.AccessTokenExpirationTime

	invalidsession = &model.Session{
		APIVersion:   "20200115",
		UserAgent:    "Go-http-client/1.1",
		UserID:       user.ID,
		ExpireAt:     time.Now().AddDate(0, 0, 1).UTC(),
		AccessToken:  session.SecureToken(8),
		RefreshToken: session.SecureToken(8),
	}
	err := ctrl.Database.Save(invalidsession)
	assert.NoError(t, err)

	ctrl.AccessTokenExpirationTime = 0

	header = gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, invalidsession),
	}

	r.GET("/sessions").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, sferror.StatusExpiredAccessToken, r.Code, r.Body.String())
		assert.JSONEq(t, `{"error":{"tag":"expired-access-token","message":"The provided access token has expired."}}`, r.Body.String())
	})

	ctrl.AccessTokenExpirationTime = original

	//
	// Expired refresh token.
	//

	invalidsession = &model.Session{
		APIVersion:   "20200115",
		UserAgent:    "Go-http-client/1.1",
		UserID:       user.ID,
		ExpireAt:     time.Now().UTC(),
		AccessToken:  session.SecureToken(8),
		RefreshToken: session.SecureToken(8),
	}
	err = ctrl.Database.Save(invalidsession)
	assert.NoError(t, err)

	header = gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, invalidsession),
	}

	r.GET("/sessions").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	//
	// Successful.
	//

	header = gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, validsession),
	}

	r.GET("/sessions").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})
}

type SessionList struct {
	ID         string    `json:"uuid"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	UserAgent  string    `json:"user_agent"`
	APIVersion string    `json:"api_version"`
	Current    bool      `json:"current"`
}

func TestRequestSessionList(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ctrl.Database, ctrl.SigningKey, ctrl.SessionSecret, ctrl.AccessTokenExpirationTime, ctrl.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ctrl)
	for i := 0; i < 2; i++ {
		s := sessions.Generate()
		s.UserID = user.ID
		s.APIVersion = session.APIVersion
		s.UserAgent = session.UserAgent
		err := ctrl.Database.Save(s)
		assert.NoError(t, err)
	}
	s := sessions.Generate()
	s.APIVersion = session.APIVersion
	s.UserID = "another-user-id"
	s.UserAgent = "trololo"
	err := ctrl.Database.Save(s)
	assert.NoError(t, err)

	//

	r.GET("/sessions").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, session),
	}

	r.GET("/sessions").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		var list []SessionList
		err = json.Unmarshal(r.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.Len(t, list, 3)

		for _, s := range list {
			assert.Equal(t, "Go-http-client/1.1", s.UserAgent)
			assert.Equal(t, "20200115", s.APIVersion)
			if s.Current {
				assert.Equal(t, session.ID, s.ID)
			}
		}
	})
}

type SessionRefresh struct {
	Session struct {
		AccessToken       string `json:"access_token"`
		RefreshToken      string `json:"refresh_token"`
		AccessExpiration  int64  `json:"access_expiration"`
		RefreshExpiration int64  `json:"refresh_expiration"`
	} `json:"session"`
}

func TestRequestSessionRegenerate(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ctrl.Database, ctrl.SigningKey, ctrl.SessionSecret, ctrl.AccessTokenExpirationTime, ctrl.RefreshTokenExpirationTime)
	user, ses := createUserWithSession(ctrl)

	//

	r.POST("/session/refresh").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Invalid request body.", "tag":"invalid-parameters"}}`, r.Body.String())
	})

	params := gofight.D{}

	r.POST("/session/refresh").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Please provide all required parameters.", "tag":"invalid-parameters"}}`, r.Body.String())
	})

	params["access_token"] = "fake-token"
	r.POST("/session/refresh").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Please provide all required parameters.", "tag":"invalid-parameters"}}`, r.Body.String())
	})

	params["refresh_token"] = "fake-token"
	r.POST("/session/refresh").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"The provided parameters are not valid.", "tag":"invalid-parameters"}}`, r.Body.String())
	})

	params["access_token"] = accessToken(ctrl, ses)
	r.POST("/session/refresh").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"The provided parameters are not valid.", "tag":"invalid-parameters"}}`, r.Body.String())
	})

	params["refresh_token"] = refreshToken(ctrl, ses)
	r.POST("/session/refresh").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		var refresh SessionRefresh
		err := json.Unmarshal(r.Body.Bytes(), &refresh)
		assert.NoError(t, err)

		fmt.Println(r.Body.String())

		assert.NotEmpty(t, refresh.Session.AccessToken)
		assert.NotEqual(t, ses.AccessToken, refresh.Session.AccessToken)
		assert.NotEmpty(t, refresh.Session.RefreshToken)
		assert.NotEqual(t, ses.RefreshToken, refresh.Session.RefreshToken)

		assert.Greater(t, refresh.Session.RefreshExpiration, ses.ExpireAt.UnixMilli())
		assert.InEpsilon(t, ses.CreatedAt.UnixNano(), ses.ExpireAt.Add(-ctrl.RefreshTokenExpirationTime).UnixNano(), 1000)
		assert.Greater(t, refresh.Session.AccessExpiration, sessions.AccessTokenExprireAt(ses).UnixMilli())
		assert.InEpsilon(t, ses.CreatedAt.UnixNano(), sessions.AccessTokenExprireAt(ses).Add(-ctrl.AccessTokenExpirationTime).UnixNano(), 1000)
	})

	//
	// Valid token encryption but does not existing in database.
	//

	ses = &model.Session{
		APIVersion:   "20200115",
		UserAgent:    "Go-http-client/1.1",
		UserID:       "fake_id",
		ExpireAt:     time.Now().Add(ctrl.RefreshTokenExpirationTime).UTC(),
		AccessToken:  session.SecureToken(8),
		RefreshToken: session.SecureToken(8),
	}

	params["access_token"] = accessToken(ctrl, ses)
	params["refresh_token"] = refreshToken(ctrl, ses)
	r.POST("/session/refresh").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"The provided parameters are not valid.", "tag":"invalid-parameters"}}`, r.Body.String())
	})

	//
	// Expired refresh token.
	//

	ses = &model.Session{
		APIVersion:   "20200115",
		UserAgent:    "Go-http-client/1.1",
		UserID:       user.ID,
		ExpireAt:     time.Now().UTC(),
		AccessToken:  session.SecureToken(8),
		RefreshToken: session.SecureToken(8),
	}
	err := ctrl.Database.Save(ses)
	assert.NoError(t, err)

	params["access_token"] = accessToken(ctrl, ses)
	params["refresh_token"] = refreshToken(ctrl, ses)
	r.POST("/session/refresh").SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"The refresh token has expired.", "tag":"expired-refresh-token"}}`, r.Body.String())
	})
}

func TestRequestSessionDelete(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ctrl.Database, ctrl.SigningKey, ctrl.SessionSecret, ctrl.AccessTokenExpirationTime, ctrl.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ctrl)

	session2 := sessions.Generate()
	session2.UserID = user.ID
	session2.APIVersion = session.APIVersion
	session2.UserAgent = session.UserAgent
	err := ctrl.Database.Save(session2)
	assert.NoError(t, err)

	//

	r.DELETE("/session").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, session),
	}
	params := gofight.D{}

	r.DELETE("/session").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Please provide the session identifier."}}`, r.Body.String())
	})

	params["uuid"] = ""
	r.DELETE("/session").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Please provide the session identifier."}}`, r.Body.String())
	})

	params["uuid"] = session.ID
	r.DELETE("/session").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"You can not delete your current session."}}`, r.Body.String())
	})

	params["uuid"] = uuid.Must(uuid.NewV4()).String()
	r.DELETE("/session").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"No session exists with the provided identifier."}}`, r.Body.String())
	})

	params["uuid"] = session2.ID
	r.DELETE("/session").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusNoContent, r.Code)

		_, err = ctrl.Database.FindSession(session2.ID) // reload session
		assert.True(t, ctrl.Database.IsNotFound(err))
	})
}

func TestRequestSessionDeleteAll(t *testing.T) {
	engine, ctrl, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ctrl.Database, ctrl.SigningKey, ctrl.SessionSecret, ctrl.AccessTokenExpirationTime, ctrl.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ctrl)

	for i := 0; i < 2; i++ {
		s := sessions.Generate()
		s.UserID = user.ID
		s.APIVersion = session.APIVersion
		s.UserAgent = session.UserAgent
		err := ctrl.Database.Save(s)
		assert.NoError(t, err)
	}

	//

	r.DELETE("/session/all").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + accessToken(ctrl, session),
	}

	r.DELETE("/session/all").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusNoContent, r.Code)

		sessions, err := ctrl.Database.FindSessionsByUserID(user.ID)
		assert.NoError(t, err)

		assert.Len(t, sessions, 1)
		assert.Equal(t, session.ID, sessions[0].ID)
	})
}
