package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/server/serializer"
	sessionpkg "github.com/mdouchement/standardfile/internal/server/session"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/mdouchement/standardfile/pkg/libsf"
	"github.com/pkg/errors"
)

type (
	sess struct {
		db       database.Client
		sessions sessionpkg.Manager
	}

	refreshSessionParams struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	deleteSessionParams struct {
		ID string `json:"uuid"`
	}
)

// List lists all active sessions for the current user.
func (s *sess) List(c echo.Context) error {
	session := currentSession(c)
	user := currentUser(c)

	sessions, err := s.db.FindActiveSessionsByUserID(user.ID)
	if err != nil && !s.db.IsNotFound(err) {
		return errors.Wrap(err, "could not get active sessions")
	}

	for _, s := range sessions {
		if s.ID == session.ID {
			s.Current = true
			break
		}
	}

	return c.JSON(http.StatusOK, serializer.Sessions(sessions))
}

// Refresh obtains a new pair of access token and refresh token.
func (s *sess) Refresh(c echo.Context) error {
	// Filter params
	var params refreshSessionParams
	if err := c.Bind(&params); err != nil {
		return c.JSON(http.StatusBadRequest, sferror.NewWithTagCode(
			http.StatusBadRequest,
			"invalid-parameters",
			"Invalid request body.",
		))
	}

	if params.AccessToken == "" || params.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, sferror.NewWithTagCode(
			http.StatusBadRequest,
			"invalid-parameters",
			"Please provide all required parameters.",
		))
	}

	sida, access, erra := s.sessions.ParseToken(params.AccessToken)
	sidr, refresh, errr := s.sessions.ParseToken(params.RefreshToken)
	if erra != nil || errr != nil || sida != sidr {
		return c.JSON(http.StatusBadRequest, sferror.NewWithTagCode(
			http.StatusBadRequest,
			"invalid-parameters",
			"The provided parameters are not valid.",
		))
	}

	// Retrieve session
	session, err := s.db.FindSessionByTokens(sida, access, refresh)
	if err != nil {
		if s.db.IsNotFound(err) {
			return c.JSON(http.StatusBadRequest, sferror.NewWithTagCode(
				http.StatusBadRequest,
				"invalid-parameters",
				"The provided parameters are not valid.",
			))
		}
		return errors.Wrap(err, "could not get refresh session")
	}

	// Regenerate tokens
	if err = s.sessions.Regenerate(session); err != nil {
		return c.JSON(http.StatusBadRequest, sferror.NewWithTagCode(
			http.StatusBadRequest,
			"expired-refresh-token",
			"The refresh token has expired.",
		))
	}

	access, err = s.sessions.Token(session, sessionpkg.TypeAccessToken)
	if err != nil {
		return errors.Wrap(err, "could not generate access token")
	}
	refresh, err = s.sessions.Token(session, sessionpkg.TypeRefreshToken)
	if err != nil {
		return errors.Wrap(err, "could not generate refresh token")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"session": echo.Map{
			"access_token":       access,
			"refresh_token":      refresh,
			"access_expiration":  libsf.UnixMillisecond(s.sessions.AccessTokenExprireAt(session).UTC()),
			"refresh_expiration": libsf.UnixMillisecond(session.ExpireAt.UTC()),
		},
	})
}

// Delete terminates the specified session by UUID.
func (s *sess) Delete(c echo.Context) error {
	// Filter params
	params := deleteSessionParams{
		ID: c.Param("id"), // Handle /v1/sessions/:id
	}
	if params.ID == "" {
		// Handle /session
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, sferror.New("Could not get session UUID."))
		}
	}

	if params.ID == "" {
		return c.JSON(http.StatusBadRequest, sferror.New("Please provide the session identifier."))
	}

	if params.ID == currentSession(c).ID {
		return c.JSON(http.StatusBadRequest, sferror.New("You can not delete your current session."))
	}

	// Retrieve session
	session, err := s.db.FindSessionByUserID(params.ID, currentUser(c).ID)
	if err != nil {
		if s.db.IsNotFound(err) {
			return c.JSON(http.StatusBadRequest, sferror.New("No session exists with the provided identifier."))
		}
		return errors.Wrap(err, "could not get user session")
	}

	if err = s.db.Delete(session); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// DeleteAll terminates all sessions, except the current one.
func (s *sess) DeleteAll(c echo.Context) error {
	sessions, err := s.db.FindSessionsByUserID(currentUser(c).ID)
	if err != nil && !s.db.IsNotFound(err) {
		return err
	}

	current := currentSession(c)
	for _, session := range sessions {
		if session.ID == current.ID {
			continue
		}

		if err = s.db.Delete(session); err != nil {
			return err
		}
	}

	return c.NoContent(http.StatusNoContent)
}
