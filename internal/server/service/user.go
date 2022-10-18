package service

import (
	"net/http"
	"time"

	argon2 "github.com/mdouchement/simple-argon2"
	"github.com/mdouchement/standardfile/internal/database"
	"github.com/mdouchement/standardfile/internal/model"
	"github.com/mdouchement/standardfile/internal/server/session"
	"github.com/mdouchement/standardfile/internal/sferror"
	"github.com/pkg/errors"
)

type (
	// A Render is an arbitrary payload serializable in JSON by the API.
	Render interface{}

	// A UserService is a service used for handle API versioning of the user.
	UserService interface {
		Register(params RegisterParams) (Render, error)
		Login(params LoginParams) (Render, error)
		Update(user *model.User, params UpdateUserParams) (Render, error)
		Password(user *model.User, params UpdatePasswordParams) (Render, error)
	}

	// RegisterParams are used to register a user.
	RegisterParams struct {
		Params
		Email                string `json:"email"`
		RegistrationPassword string `json:"password"`
		PasswordNonce        string `json:"pw_nonce"`
		PasswordCost         int    `json:"pw_cost"` // Before 202000115
		Version              string `json:"version"`
		Created              string `json:"created"`     // Since 20200115
		Identifier           string `json:"identifier"`  // Since 20200115
		Origination          string `json:"origination"` // Since 20200115
	}

	// LoginParams are used to login a user.
	LoginParams struct {
		Params
		Email         string `json:"email"`
		Password      string `json:"password"`
		CodeChallenge string `json:"code_challenge"`
		CodeVerifier  string `json:"code_verifier"`
	}

	// UpdateUserParams are used to update a user.
	UpdateUserParams struct {
		Params
		PasswordCost  int    `json:"pw_cost"`
		PasswordNonce string `json:"pw_nonce"`
		PasswordSalt  string `json:"pw_salt"`
		Version       string `json:"version"`
	}

	// UpdatePasswordParams are used to update user's password.
	UpdatePasswordParams struct {
		UpdateUserParams
		Identifier      string `json:"identifier"`
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		NewEmail        string `json:"new_email"`
	}

	// Success handler that generates response payload.
	success func(u *model.User, p Params, r M) (Render, error)

	userServiceBase struct {
		db       database.Client
		sessions session.Manager
	}
)

// NewUser returns a new UserService.
func NewUser(db database.Client, sessions session.Manager, version string) (s UserService) {
	switch version {
	case "20200115":
		s = &userService20200115{
			userService20161215{
				userServiceBase: userServiceBase{
					db:       db,
					sessions: sessions,
				},
			},
		}
	case "20190520":
		fallthrough
	case "20161215":
		fallthrough
	default:
		s = &userService20161215{
			userServiceBase: userServiceBase{
				db:       db,
				sessions: sessions,
			},
		}
	}

	return s
}

func (s *userServiceBase) register(params RegisterParams, success success, response M) (Render, error) {
	// Check if the email is free to use.
	u, err := s.db.FindUserByMail(params.Email)
	if err != nil && !s.db.IsNotFound(err) {
		return nil, errors.Wrap(err, "could not get access to database")
	}
	if u != nil {
		// StatusUnauthorized is used in the reference server implem.
		// (even if no authentication is needed here..)
		return nil, sferror.NewWithTagCode(http.StatusUnauthorized, "", "This email is already registered.")
	}

	// Initialize user
	user := model.NewUser()
	user.Email = params.Email
	user.PasswordNonce = params.PasswordNonce
	user.PasswordCost = params.PasswordCost
	if params.Version != "" {
		user.Version = params.Version
	}

	// Crypt password
	user.Password, err = argon2.GenerateFromPasswordString(params.RegistrationPassword, argon2.Default)
	if err != nil {
		return nil, errors.Wrap(err, "could not store user password safe")
	}
	user.PasswordUpdatedAt = time.Now().Unix()

	// Persist the model
	if err := s.db.Save(user); err != nil {
		return nil, errors.Wrap(err, "could not persist user")
	}

	return success(user, params.Params, response)
}

func (s *userServiceBase) login(params LoginParams, success success, response M) (Render, error) {
	// Retrieve user
	user, err := s.db.FindUserByMail(params.Email)
	if err != nil {
		if s.db.IsNotFound(err) {
			return nil, sferror.NewWithTagCode(http.StatusUnauthorized, "", "Invalid email or password.")
		}
		return nil, errors.Wrap(err, "could not get user")
	}

	// Verify password
	if err = argon2.CompareHashAndPasswordString(user.Password, params.Password); err != nil {
		if err == argon2.ErrMismatchedHashAndPassword {
			return nil, sferror.NewWithTagCode(http.StatusUnauthorized, "", "Invalid email or password.")
		}
		return nil, errors.Wrap(err, "could not validate password")
	}

	return success(user, params.Params, response)
}

func (s *userServiceBase) update(user *model.User, params UpdateUserParams, success success, response M) (Render, error) {
	s.apply(user, params)

	if err := s.db.Save(user); err != nil {
		return nil, errors.Wrap(err, "could not persist user")
	}
	return success(user, params.Params, response)
}

func (s *userServiceBase) password(user *model.User, params UpdatePasswordParams, success success, response M) (Render, error) {
	// Verify CurrentPassword
	if err := argon2.CompareHashAndPasswordString(user.Password, params.CurrentPassword); err != nil {
		if err == argon2.ErrMismatchedHashAndPassword {
			return nil, sferror.NewWithTagCode(http.StatusUnauthorized, "", "The current password you entered is incorrect. Please try again.")
		}
		return nil, errors.Wrap(err, "could not validate password")
	}

	// Crypt & update password
	pw, err := argon2.GenerateFromPasswordString(params.NewPassword, argon2.Default)
	if err != nil {
		return nil, errors.Wrap(err, "could not store user password safe")
	}
	user.Password = pw
	user.PasswordUpdatedAt = time.Now().Unix()

	// Only update email, when parameter available
	if params.NewEmail != "" {
		user.Email = params.NewEmail
	}

	s.apply(user, params.UpdateUserParams)

	if err := s.db.Save(user); err != nil {
		return nil, errors.Wrap(err, "could not persist user")
	}
	return success(user, params.Params, response)
}

// updates given user with given params.
// works like strong_parameter.
func (s *userServiceBase) apply(u *model.User, params UpdateUserParams) {
	if params.PasswordCost > 0 {
		u.PasswordCost = params.PasswordCost
	}

	if params.PasswordNonce != "" {
		u.PasswordNonce = params.PasswordNonce
	}

	if params.PasswordSalt != "" {
		u.PasswordSalt = params.PasswordSalt
	}

	if params.Version != "" {
		u.Version = params.Version
	}
}
