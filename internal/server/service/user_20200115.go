package service

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server/serializer"
	"github.com/mdouchement/standardfile/internal/server/session"
	sessionpkg "github.com/mdouchement/standardfile/internal/server/session"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/pkg/errors"
)

type userService20200115 struct {
	userService20161215
}

func (s *userService20200115) Register(params RegisterParams) (Render, error) {
	return s.register(params, s.SuccessfulAuthentication, nil)
}

func (s *userService20200115) Login(params LoginParams) (Render, error) {
	return s.login(params, s.SuccessfulAuthentication, nil)
}

func (s *userService20200115) Update(user *model.User, params UpdateUserParams) (Render, error) {
	return s.update(user, params, s.SuccessfulAuthentication, nil)
}

func (s *userService20200115) Password(user *model.User, params UpdatePasswordParams) (Render, error) {
	// FIXME: Reference implementation creates a session only if the user uses the 004 version.
	// As version 004 as been released too early by mistake, this code seems now useless.
	// https://github.com/standardnotes/syncing-server/pull/56/files#diff-21301a75c96c49e2bf016f4c63206521R12
	// `upgrading_protocol_version && new_protocol_version == @user_class::SESSIONS_PROTOCOL_VERSION`

	return s.password(user, params, s.SuccessfulAuthentication, M{
		"key_params": s.KeyParams(user),
	})
}

func (s *userService20200115) SuccessfulAuthentication(u *model.User, params Params, response M) (Render, error) {
	if !session.UserSupportsSessions(u) {
		return s.userService20161215.SuccessfulAuthentication(u, params, response)
	}

	if response == nil {
		response = M{}
	}

	var err error
	session := params.Session
	if session == nil {
		session, err = s.CreateSession(u, params)
		if err != nil {
			return nil, err
		}
	}

	access, err := s.sessions.Token(session, sessionpkg.TypeAccessToken)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate access token")
	}
	refresh, err := s.sessions.Token(session, sessionpkg.TypeRefreshToken)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate refresh token")
	}

	response["user"] = serializer.User(u)
	response["session"] = echo.Map{
		"access_token":       access,
		"refresh_token":      refresh,
		"access_expiration":  s.sessions.AccessTokenExprireAt(session).UTC().UnixMilli(),
		"refresh_expiration": session.ExpireAt.UTC().UnixMilli(),
	}
	return response, nil
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

func (s *userService20200115) KeyParams(u *model.User) M {
	params := M{
		"version":    u.Version,
		"identifier": u.Email,
	}

	switch u.Version {
	case libsf.ProtocolVersion2:
		params["email"] = u.Email
		params["pw_salt"] = u.PasswordSalt
		params["pw_auth"] = u.PasswordAuth
	case libsf.ProtocolVersion3:
		params["pw_cost"] = u.PasswordCost
		params["pw_nonce"] = u.PasswordNonce
	case libsf.ProtocolVersion4:
		params["pw_nonce"] = u.PasswordNonce
		// params["created"] = u.kp_created TODO:
		// params["origination"] = u.kp_origination
	}

	return params
}
