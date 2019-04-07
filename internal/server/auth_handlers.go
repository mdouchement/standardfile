package server

import (
	"log"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	argon2 "github.com/mdouchement/simple-argon2"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/pkg/errors"
)

type (
	// auth contains all authentication handlers.
	auth struct {
		db         database.Client
		signingKey []byte
	}

	credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	updateAuth struct {
		PasswordCost  int    `json:"pw_cost"`
		PasswordNonce string `json:"pw_nonce"`
		PasswordSalt  string `json:"pw_salt"`
		Version       string `json:"version"`
	}

	updatePassword struct {
		updateAuth
		Identifier      string `json:"identifier"`
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
)

///// Register
////
//

// Register handler is used to register the user.
// https://standardfile.org/#api-auth
func (h *auth) Register(c echo.Context) error {
	user := model.NewUser()

	// Filter params
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusUnauthorized, sferror.New("Could not get user's params."))
	}

	if user.Email == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No email provided."))
	}
	if user.RegistrationPassword == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No password provided."))
	}
	if user.PasswordNonce == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No nonce provided."))
	}
	if user.PasswordCost <= 0 {
		return c.JSON(http.StatusUnauthorized, sferror.New("No password cost provided."))
	}

	// Check if the email is free to use.
	u, err := h.db.FindUserByMail(user.Email)
	if err != nil && !h.db.IsNotFound(err) {
		return errors.Wrap(err, "could not get access to database")
	}
	if u != nil {
		// StatusUnauthorized is used in the reference server implem.
		// (even if no authentication is needed here..)
		return c.JSON(http.StatusUnauthorized, sferror.New("This email is already registered."))
	}

	// Crypt password
	user.Password, err = argon2.GenerateFromPasswordString(user.RegistrationPassword, argon2.Default)
	if err != nil {
		return errors.Wrap(err, "could not store user password safe")
	}
	user.PasswordUpdatedAt = time.Now().Unix()
	user.RegistrationPassword = "" // Disable password rendering

	// Persist the model
	if err := h.db.Save(user); err != nil {
		return errors.Wrap(err, "could not persist user")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"user":  user,
		"token": h.TokenFromUser(user),
	})
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

	// Check if the user exists.
	user, err := h.db.FindUserByMail(email)
	if err != nil {
		if h.db.IsNotFound(err) {
			return c.JSON(http.StatusUnauthorized, sferror.New("Bad email provided."))
		}
		return errors.Wrap(err, "could not get user")
	}

	// TODO 2FA
	// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/auth_controller.rb#L16

	// Render
	params := echo.Map{
		"identifier": user.Email,
		"pw_cost":    user.PasswordCost,
		"version":    user.Version,
	}

	switch user.Version {
	case model.Version2:
		params["pw_salt"] = user.PasswordSalt
	case model.Version3:
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
	var params credentials
	if err := c.Bind(&params); err != nil {
		log.Println("Could not get parameters:", err)
		return c.JSON(http.StatusUnauthorized, sferror.New("Could not get credentials."))
	}

	if params.Email == "" || params.Password == "" {
		return c.JSON(http.StatusUnauthorized, sferror.New("No email or password provided."))
	}

	// TODO 2FA
	// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/auth_controller.rb#L16

	// Retrieve user
	user, err := h.db.FindUserByMail(params.Email)
	if err != nil {
		if h.db.IsNotFound(err) {
			return c.JSON(http.StatusUnauthorized, sferror.New("Invalid email or password."))
		}
		return errors.Wrap(err, "could not get user")
	}

	// Verify password
	if err = argon2.CompareHashAndPasswordString(user.Password, params.Password); err != nil {
		if err == argon2.ErrMismatchedHashAndPassword {
			return c.JSON(http.StatusUnauthorized, sferror.New("Invalid email or password."))
		}
		return errors.Wrap(err, "could not validate password")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"user":  user,
		"token": h.TokenFromUser(user),
	})
}

///// Update
////
//

// Update used to updates a user.
func (h *auth) Update(c echo.Context) error {
	// Filter params
	var params updateAuth
	if err := c.Bind(&params); err != nil {
		log.Println("Could not get parameters:", err)
		return c.JSON(http.StatusUnauthorized, sferror.New("Could not get parameters."))
	}

	// Update
	user := currentUser(c)
	h.apply(user, params)

	// Persist the model
	if err := h.db.Save(user); err != nil {
		return errors.Wrap(err, "could not persist user")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"user":  user,
		"token": h.TokenFromUser(user),
	})
}

///// Update Password
////
//

// UpdatePassword used to updates a user's password.
// https://standardfile.org/#post-auth-change_pw
func (h *auth) UpdatePassword(c echo.Context) error {
	// Filter params
	var params updatePassword
	if err := c.Bind(&params); err != nil {
		log.Println("Could not get parameters:", err)
		return c.JSON(http.StatusUnauthorized, sferror.New("Could not get parameters."))
	}

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

	// Verify CurrentPassword
	user := currentUser(c)
	if err := argon2.CompareHashAndPasswordString(user.Password, params.CurrentPassword); err != nil {
		if err == argon2.ErrMismatchedHashAndPassword {
			return c.JSON(http.StatusUnauthorized, sferror.New("The current password you entered is incorrect. Please try again."))
		}
		return errors.Wrap(err, "could not validate password")
	}

	// Crypt & update password
	pw, err := argon2.GenerateFromPasswordString(user.RegistrationPassword, argon2.Default)
	if err != nil {
		return errors.Wrap(err, "could not store user password safe")
	}
	user.Password = pw
	user.PasswordUpdatedAt = time.Now().Unix()

	h.apply(user, params.updateAuth)

	// Persist the model
	if err := h.db.Save(user); err != nil {
		return errors.Wrap(err, "could not persist user")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"user":  user,
		"token": h.TokenFromUser(user),
	})
}

/////////////////////
//                 //
// Helpers         //
//                 //
/////////////////////

// TokenFromUser returns a JWT token for the given user.
func (h *auth) TokenFromUser(u *model.User) string {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_uuid"] = u.ID
	// claims["pw_hash"] = fmt.Sprintf("%x", sha256.Sum256([]byte(u.Password))) // See readme
	claims["iss"] = "github.com/mdouchement/standardfile"
	claims["iat"] = time.Now().Unix() // Unix Timestamp in seconds

	t, err := token.SignedString(h.signingKey)
	if err != nil {
		log.Fatalf("could not generate token: %s", err)
	}
	return t
}

// updates given user with given params.
// works like strong_parameter.
func (h *auth) apply(user *model.User, params updateAuth) {
	if params.PasswordCost > 0 {
		user.PasswordCost = params.PasswordCost
	}

	if params.PasswordNonce != "" {
		user.PasswordNonce = params.PasswordNonce
	}

	if params.PasswordSalt != "" {
		user.PasswordSalt = params.PasswordSalt
	}

	if params.Version != "" {
		user.Version = params.Version
	}
}
