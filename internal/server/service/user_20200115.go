package service

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server/serializer"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/mdouchement/standardfile/internal/sferror"
)

type userService20200115 struct {
	userService20161215
}

func (s *userService20200115) Register(params RegisterParams) (Render, error) {
	return s.register(params, s.SuccessfulAuthentication)
}

func (s *userService20200115) Login(params LoginParams) (Render, error) {
	return s.login(params, s.SuccessfulAuthentication)
}

func (s *userService20200115) Update(user *model.User, params UpdateUserParams) (Render, error) {
	return s.update(user, params, s.SuccessfulAuthentication)
}

func (s *userService20200115) Password(user *model.User, params UpdatePasswordParams) (Render, error) {
	// FIXME: Reference implementation added a restrictive condition
	// https://github.com/standardnotes/syncing-server/pull/56/files#diff-21301a75c96c49e2bf016f4c63206521R12
	return s.password(user, params, s.SuccessfulAuthentication)
}

func (s *userService20200115) SuccessfulAuthentication(u *model.User, params Params) (Render, error) {
	if !session.UserSupportsSessions(u) {
		return s.userService20161215.SuccessfulAuthentication(u, params)
	}

	var err error
	session := params.Session
	if session == nil {
		session, err = s.CreateSession(u, params)
		if err != nil {
			return nil, err
		}
	}

	return echo.Map{
		"user": serializer.User(u),
		"session": echo.Map{
			"expire_at":     s.sessions.AccessTokenExprireAt(session).UTC(),
			"refresh_token": session.RefreshToken,
			"valid_until":   session.ExpireAt.UTC(),
		},
		"token": session.AccessToken,
	}, nil
}

func (s *userService20200115) CreateSession(u *model.User, params Params) (*model.Session, error) {
	session := s.sessions.Generate()
	session.UserID = u.ID
	session.APIVersion = params.APIVersion
	session.UserAgent = params.UserAgent

	if err := s.db.Save(session); err != nil {
		return nil, sferror.NewWithTagCode(http.StatusBadRequest, "", "Could not create a session.")
	}

	return session, nil
}
