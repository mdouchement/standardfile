package serializer

import "github.com/mdouchement/standardfile/internal/model"

// User serializes the render of a user.
func User(m *model.User) map[string]interface{} {
	r := map[string]interface{}{
		"uuid":       m.ID,
		"created_at": m.CreatedAt.UTC(),
		"updated_at": m.UpdatedAt.UTC(),
		"email":      m.Email,
		"version":    m.Version,
		"pw_cost":    m.PasswordCost,
	}

	switch m.Version {
	case model.Version2:
		r["pw_salt"] = m.PasswordSalt
		r["pw_auth"] = m.PasswordAuth
	case model.Version3:
		fallthrough
	case model.Version4:
		r["pw_nonce"] = m.PasswordNonce
	}

	return r
}
