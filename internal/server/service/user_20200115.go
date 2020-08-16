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
	success := s.SuccessfulAuthentication // Upgrading to version 004 requires to reencrypt all data client side.
	if !session.UserUpgradingToSessions(user, params.Version) {
		success = s.userService20161215.SuccessfulAuthentication
	}

	return s.password(user, params, success)
}

func (s *userService20200115) SuccessfulAuthentication(u *model.User, params Params) (Render, error) {
	if !session.UserSupportsSessions(u) {
		return s.userService20161215.SuccessfulAuthentication(u, params)
	}

	session, err := s.CreateSession(u, params)
	if err != nil {
		return nil, err
	}

	return echo.Map{
		"user": serializer.User(u),
		"session": echo.Map{
			"expire_at":     s.sessions.AccessTokenExprireAt(session),
			"refresh_token": session.RefreshToken,
			"valid_until":   session.ExpireAt,
		},
		"token": session.AccessToken,
	}, nil
}

func (s *userService20200115) CreateSession(u *model.User, params Params) (*model.Session, error) {
	session := &model.Session{
		APIVersion:   params.APIVersion,
		UserAgent:    params.UserAgent,
		UserID:       u.ID,
		AccessToken:  session.SecureToken(24),
		RefreshToken: session.SecureToken(24),
	}
	if err := s.db.Save(session); err != nil {
		return nil, sferror.NewWithTagCode(http.StatusBadRequest, "", "Could not create a session.")
	}

	return session, nil
}
