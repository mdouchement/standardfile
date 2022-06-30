package server

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"

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
	// Fetch params from URL queries
	email := c.QueryParam("email")
	if email == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No email provided."))
	}

	return h.params(c, email)
}

func (h *auth) ParamsPKCE(c echo.Context) error {
	var params service.LoginParams
	if err := c.Bind(&params); err != nil {
		log.Println("Could not get parameters:", err)
		return c.JSON(http.StatusBadRequest, sferror.New("Could not get credentials."))
	}
	params.UserAgent = c.Request().UserAgent()
	params.Session = currentSession(c)

	if params.Email == "" {
		return c.JSON(http.StatusBadRequest, sferror.New("Please provide an email address."))
	}

	if params.CodeChallenge == "" {
		return c.JSON(http.StatusBadRequest, sferror.New("Please provide the code challenge parameter"))
	}

	if err := h.db.StorePKCE(params.CodeChallenge); err != nil {
		log.Println("Could not store code challenge:", err)
		return c.JSON(http.StatusBadRequest, sferror.New("Could not store code challenge."))
	}

	return h.params(c, params.Email)
}

func (h *auth) params(c echo.Context, email string) error {
	// Check if the user exists.
	user, err := h.db.FindUserByMail(email)
	if err != nil {
		hostname, _ := os.Hostname()
		return c.JSON(http.StatusOK, echo.Map{
			"identifier": email,
			"nonce":      sha256.Sum256([]byte(email + hostname)),
			"version":    libsf.ProtocolVersion4,
		})
	}

	// TODO 2FA
	// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/auth_controller.rb#L16

	// Render
	params := echo.Map{
		"identifier": user.Email,
		"version":    user.Version,
	}

	switch user.Version {
	case libsf.ProtocolVersion2:
		params["pw_cost"] = user.PasswordCost
		params["pw_salt"] = user.PasswordSalt
	case libsf.ProtocolVersion3:
		params["pw_cost"] = user.PasswordCost
		params["pw_nonce"] = user.PasswordNonce
	case libsf.ProtocolVersion4:
		params["pw_nonce"] = user.PasswordNonce
	}

	return c.JSON(http.StatusOK, params)
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

	return h.login(c, params)

}

func (h *auth) login(c echo.Context, params service.LoginParams) error {

	// TODO 2FA
	// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/auth_controller.rb#L16

	service := service.NewUser(h.db, h.sessions, params.APIVersion)
	login, err := service.Login(params)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, login)
}

func (h *auth) LoginPKCE(c echo.Context) error {
	// Filter params
	var params service.LoginParams
	if err := c.Bind(&params); err != nil {
		log.Println("Could not get parameters:", err)
		return c.JSON(http.StatusBadRequest, sferror.New("Could not get credentials."))
	}
	params.UserAgent = c.Request().UserAgent()
	params.Session = currentSession(c)

	if params.Email == "" || params.Password == "" || params.CodeVerifier == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("Invalid login credentials."))
	}

	hash := sha256.Sum256([]byte(params.CodeVerifier))
	base64_hash := fmt.Sprintf("%x", hash[:])
	res, err := h.db.RemovePKCE(base64.RawStdEncoding.EncodeToString([]byte(base64_hash)))
	if err != nil {
		log.Println("Could not retrieve code challenge:", err)
		return c.JSON(http.StatusBadRequest, sferror.New("Could not get credentials."))
	}
	if !res {
		return c.JSON(http.StatusBadRequest, sferror.New("Challenge not matched."))
	}

	return h.login(c, params)
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
