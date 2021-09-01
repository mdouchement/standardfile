package server

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/server/service"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/mdouchement/standardfile/pkg/libsf"
)

// auth contains all authentication handlers.
type auth struct {
	db       database.Client
	sessions session.Manager
}

///// Register
////
//

// Register handler is used to register the user.
// https://standardfile.org/#api-auth
func (h *auth) Register(c echo.Context) error {
	// Filter params
	var params service.RegisterParams
	if err := c.Bind(&params); err != nil {
		return c.JSON(http.StatusUnauthorized, sferror.New("Could not get user's params."))
	}
	params.UserAgent = c.Request().UserAgent()
	params.Session = currentSession(c)

	if params.Email == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No email provided."))
	}
	if params.RegistrationPassword == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No password provided."))
	}
	if params.PasswordNonce == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No nonce provided."))
	}
	if libsf.VersionLesser(libsf.APIVersion20200115, params.APIVersion) && params.PasswordCost <= 0 {
		return c.JSON(http.StatusUnauthorized, sferror.New("No password cost provided."))
	}

	service := service.NewUser(h.db, h.sessions, params.APIVersion)
	register, err := service.Register(params)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, register)
}

///// Params
////
//

// Params used for password generation.
// https://standardfile.org/#get-auth-params
func (h *auth) Params(c echo.Context) error {
	var params service.AuthParams

	// Fetch params from URL queries
	params.Email = c.QueryParam("email")
	if params.Email == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No email provided."))
	}
	params.APIVersion = c.QueryParam("api")
	params.UserAgent = c.Request().UserAgent()
	params.Session = currentSession(c)

	// TODO 2FA
	// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/auth_controller.rb#L16

	service := service.NewUser(h.db, h.sessions, params.APIVersion)
	auth, err := service.AuthParams(params)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, auth)
}

///// Login
////
//

// Login used for authenticates a user and returns a JWT.
// https://standardfile.org/#post-auth-sign_in
func (h *auth) Login(c echo.Context) error {
	// Filter params
	var params service.LoginParams
	if err := c.Bind(&params); err != nil {
		log.Println("Could not get parameters:", err)
		return c.JSON(http.StatusBadRequest, sferror.New("Could not get credentials."))
	}
	params.UserAgent = c.Request().UserAgent()
	params.Session = currentSession(c)

	if params.Email == "" || params.Password == "" {
		return c.JSON(http.StatusBadRequest, sferror.New("No email or password provided."))
	}

	// TODO 2FA
	// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/auth_controller.rb#L16

	service := service.NewUser(h.db, h.sessions, params.APIVersion)
	login, err := service.Login(params)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, login)
}

///// Logout
////
//

// Logout used for terminates the current session.
func (h *auth) Logout(c echo.Context) error {
	session := currentSession(c)
	if session != nil {
		err := h.db.Delete(session)
		if err != nil && h.db.IsNotFound(err) {
			return err
		}
	}

	return c.NoContent(http.StatusNoContent)
}

///// Update
////
//

// Update used to updates a user.
func (h *auth) Update(c echo.Context) error {
	// Filter params
	var params service.UpdateUserParams
	if err := c.Bind(&params); err != nil {
		log.Println("Could not get parameters:", err)
		return c.JSON(http.StatusUnauthorized, sferror.New("Could not get parameters."))
	}
	params.UserAgent = c.Request().UserAgent()
	params.Session = currentSession(c)

	service := service.NewUser(h.db, h.sessions, params.APIVersion)
	update, err := service.Update(currentUser(c), params)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, update)
}

///// Update Password
////
//

// UpdatePassword used to updates a user's password.
// https://standardfile.org/#post-auth-change_pw
func (h *auth) UpdatePassword(c echo.Context) error {
	// Filter params
	var params service.UpdatePasswordParams
	if err := c.Bind(&params); err != nil {
		log.Println("Could not get parameters:", err)
		return c.JSON(http.StatusUnauthorized, sferror.New("Could not get parameters."))
	}
	params.UserAgent = c.Request().UserAgent()
	params.Session = currentSession(c)

	// Check CurrentPassword presence.
	if params.CurrentPassword == "" {
		return c.JSON(http.StatusUnauthorized,
			sferror.New("Your current password is required to change your password. Please update your application if you do not see this option."))
	}

	// Check NewPassword presence.
	if params.NewPassword == "" {
		return c.JSON(http.StatusUnauthorized,
			sferror.New("Your new password is required to change your password. Please update your application if you do not see this option."))
	}

	service := service.NewUser(h.db, h.sessions, params.APIVersion)
	password, err := service.Password(currentUser(c), params)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, password)
}
