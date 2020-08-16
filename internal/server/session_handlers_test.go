package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/appleboy/gofight"
	"github.com/gofrs/uuid"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/stretchr/testify/assert"
)

type SessionList struct {
	ID         string    `json:"uuid"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	UserAgent  string    `json:"user_agent"`
	APIVersion string    `json:"api_version"`
	Current    bool      `json:"current"`
}

func TestRequestSessionList(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ioc.Database, ioc.SigningKey, ioc.AccessTokenExpirationTime, ioc.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ioc)
	for i := 0; i < 2; i++ {
		s := sessions.Generate()
		s.UserID = user.ID
		s.APIVersion = session.APIVersion
		s.UserAgent = session.UserAgent
		err := ioc.Database.Save(s)
		assert.NoError(t, err)
	}
	s := sessions.Generate()
	s.APIVersion = session.APIVersion
	s.UserID = "another-user-id"
	s.UserAgent = "trololo"
	err := ioc.Database.Save(s)
	assert.NoError(t, err)

	//

	r.GET("/sessions").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + session.AccessToken,
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
	Token   string `json:"token"`
	Session struct {
		ExpireAt     time.Time `json:"expire_at"`
		RefreshToken string    `json:"refresh_token"`
		ValidUntil   time.Time `json:"valid_until"`
	} `json:"session"`
}

func TestRequestSessionRegenerate(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ioc.Database, ioc.SigningKey, ioc.AccessTokenExpirationTime, ioc.RefreshTokenExpirationTime)
	_, session := createUserWithSession(ioc)

	//

	r.POST("/session/refresh").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + session.AccessToken,
	}

	params := gofight.D{}

	r.POST("/session/refresh").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Please provide all required parameters."}}`, r.Body.String())
	})

	params["access_token"] = session.AccessToken
	r.POST("/session/refresh").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusBadRequest, r.Code)
		assert.JSONEq(t, `{"error":{"message":"Please provide all required parameters."}}`, r.Body.String())
	})

	params["refresh_token"] = session.RefreshToken
	r.POST("/session/refresh").SetHeader(header).SetJSON(params).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)

		var refresh SessionRefresh
		err := json.Unmarshal(r.Body.Bytes(), &refresh)
		assert.NoError(t, err)

		assert.NotEmpty(t, refresh.Token)
		assert.NotEqual(t, session.AccessToken, refresh.Token)
		assert.NotEmpty(t, refresh.Session.RefreshToken)
		assert.NotEqual(t, session.RefreshToken, refresh.Session.RefreshToken)

		assert.Greater(t, refresh.Session.ValidUntil.UnixNano(), session.ExpireAt.UnixNano())
		assert.InEpsilon(t, session.CreatedAt.UnixNano(), session.ExpireAt.Add(-ioc.RefreshTokenExpirationTime).UnixNano(), 1000)
		assert.Greater(t, refresh.Session.ExpireAt.UnixNano(), sessions.AccessTokenExprireAt(session).UnixNano())
		assert.InEpsilon(t, session.CreatedAt.UnixNano(), sessions.AccessTokenExprireAt(session).Add(-ioc.AccessTokenExpirationTime).UnixNano(), 1000)
	})
}

func TestRequestSessionDelete(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ioc.Database, ioc.SigningKey, ioc.AccessTokenExpirationTime, ioc.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ioc)

	session2 := sessions.Generate()
	session2.UserID = user.ID
	session2.APIVersion = session.APIVersion
	session2.UserAgent = session.UserAgent
	err := ioc.Database.Save(session2)
	assert.NoError(t, err)

	//

	r.DELETE("/session").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + session.AccessToken,
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

		_, err = ioc.Database.FindSession(session2.ID) // reload session
		assert.True(t, ioc.Database.IsNotFound(err))
	})
}

func TestRequestSessionDeleteAll(t *testing.T) {
	engine, ioc, r, cleanup := setup()
	defer cleanup()

	sessions := session.NewManager(ioc.Database, ioc.SigningKey, ioc.AccessTokenExpirationTime, ioc.RefreshTokenExpirationTime)
	user, session := createUserWithSession(ioc)

	for i := 0; i < 2; i++ {
		s := sessions.Generate()
		s.UserID = user.ID
		s.APIVersion = session.APIVersion
		s.UserAgent = session.UserAgent
		err := ioc.Database.Save(s)
		assert.NoError(t, err)
	}

	//

	r.DELETE("/session/all").Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.JSONEq(t, `{"error":{"tag":"invalid-auth", "message":"Invalid login credentials."}}`, r.Body.String())
	})

	header := gofight.H{
		"Authorization": "Bearer " + session.AccessToken,
	}

	r.DELETE("/session/all").SetHeader(header).Run(engine, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusNoContent, r.Code)

		sessions, err := ioc.Database.FindSessionsByUserID(user.ID)
		assert.NoError(t, err)

		assert.Len(t, sessions, 1)
		assert.Equal(t, session.ID, sessions[0].ID)
	})
}
