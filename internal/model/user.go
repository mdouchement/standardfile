package model

const (
	// Version2 is the client version.
	Version2 = "002"
	// Version3 is the client version.
	Version3 = "003"
	// VersionLatest is the client version.
	VersionLatest = Version3
)

// A User represents a database record and the rendered API response.
type User struct {
	Base `msgpack:",inline" storm:"inline"`

	// Standardfile fields
	Email         string `json:"email"              msgpack:"email"    storm:"unique"`
	Password      string `json:"-"                  msgpack:"password,omitempty"`
	PasswordCost  int    `json:"pw_cost"            msgpack:"pw_cost"`
	PasswordNonce string `json:"pw_nonce,omitempty" msgpack:"pw_nonce,omitempty"`
	PasswordAuth  string `json:"pw_auth,omitempty"  msgpack:"pw_auth,omitempty"`
	Version       string `json:"version"            msgpack:"version"`

	// V2 flields compatibility
	PasswordSalt string `json:"pw_salt,omitempty"  msgpack:"pw_salt,omitempty"`

	// Custom fields
	RegistrationPassword string `json:"password,omitempty" msgpack:"-"`
	PasswordUpdatedAt    int64  `json:"-"                  msgpack:"password_updated_at"`
}

// NewUser returns a new user with default params.
func NewUser() *User {
	return &User{
		// https://github.com/standardfile/ruby-server/blob/master/app/controllers/api/auth_controller.rb#L70
		// Version3 is provided by client and overrided before inserting record.
		Version: Version2,
	}
}
