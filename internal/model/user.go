package model

import "github.com/mdouchement/standardfile/pkg/libsf"

// A User represents a database record.
type User struct {
	Base `msgpack:",inline" storm:"inline"`

	// Standardfile fields
	Email         string `msgpack:"email"    storm:"unique"`
	Password      string `msgpack:"password,omitempty"`
	PasswordCost  int    `msgpack:"pw_cost,omitempty"`
	PasswordNonce string `msgpack:"pw_nonce,omitempty"`
	PasswordAuth  string `msgpack:"pw_auth,omitempty"`
	Version       string `msgpack:"version"`

	// V2 flields compatibility
	PasswordSalt string `msgpack:"pw_salt,omitempty"`

	// Custom fields
	PasswordUpdatedAt int64 `msgpack:"password_updated_at"`
}

// NewUser returns a new user with default params.
func NewUser() *User {
	return &User{
		// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/auth_controller.rb#L70
		// Version3 is provided by client and overrided before inserting record.
		Version: libsf.ProtocolVersion2,
	}
}
